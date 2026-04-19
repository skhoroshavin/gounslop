package readfriendlyorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "readfriendlyorder",
	Doc:      "enforce reading-friendly code organization",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	for _, file := range pass.Files {
		filename := pass.Fset.File(file.Pos()).Name()
		if ast.IsGenerated(file) || strings.HasSuffix(filename, "_testmain.go") {
			continue
		}

		src, err := readFileSource(pass.Fset, file)
		if err != nil {
			continue
		}

		reportInitOrdering(pass, file, src)
		reportTopLevelOrdering(pass, file, src)
		reportMethodOrdering(pass, file, insp, src)
		reportTestOrdering(pass, file, src)
	}

	return nil, nil
}

func reportInitOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	firstNonInitFuncIdx := -1
	firstNonInitFuncName := ""
	for i, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if fn.Name.Name != "init" {
			if firstNonInitFuncIdx == -1 {
				firstNonInitFuncIdx = i
				firstNonInitFuncName = fn.Name.Name
			}
			continue
		}
		// This is an init() function — check if any non-init func came before it
		if firstNonInitFuncIdx != -1 && firstNonInitFuncIdx < i {
			diag := analysis.Diagnostic{
				Pos:     fn.Pos(),
				Message: fmt.Sprintf("Place init() before %q for visibility.", firstNonInitFuncName),
			}
			fix := buildSwapFix(pass.Fset, file, src, file.Decls[firstNonInitFuncIdx], d)
			if fix != nil {
				diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
			}
			pass.Report(diag)
		}
	}
}

func reportTopLevelOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	decls := collectTopLevelDecls(file)
	refs := buildRefGraph(pass, file, decls)
	cyclicNames := findCyclicNames(decls, refs)

	// Collect violations and compute the fix
	type violation struct {
		d        decl
		consumer *decl
	}
	var violations []violation

	for _, d := range decls {
		if d.exported || d.name == "init" || cyclicNames[d.name] {
			continue
		}
		if hasEagerReference(decls, d, refs) {
			continue
		}
		consumer := findFirstConsumer(decls, d, refs)
		if consumer == nil {
			continue
		}
		violations = append(violations, violation{d, consumer})
	}

	if len(violations) == 0 {
		return
	}

	// Compute the reorder fix for all violations at once
	fix := computeTopLevelReorderFix(pass, file, src, decls, refs, cyclicNames)

	for i, v := range violations {
		var msg string
		if v.d.kind == "const" || v.d.kind == "var" {
			msg = fmt.Sprintf("Place constant %q below the top-level symbol %q that uses it.",
				v.d.name, v.consumer.name)
		} else {
			msg = fmt.Sprintf("Place helper %q below the top-level symbol %q that depends on it.",
				v.d.name, v.consumer.name)
		}
		diag := analysis.Diagnostic{
			Pos:     v.d.node.Pos(),
			Message: msg,
		}
		// Attach fix only to the first diagnostic to avoid conflicting edits
		if i == 0 && fix != nil {
			diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
		}
		pass.Report(diag)
	}
}

// computeTopLevelReorderFix computes the correct declaration order and builds a fix.
func computeTopLevelReorderFix(pass *analysis.Pass, file *ast.File, src []byte,
	decls []decl, refs map[string]map[string]bool, cyclicNames map[string]bool) *analysis.SuggestedFix {

	fileDecls := collectNonImportDecls(file)
	if hasMultiSpecViolation(fileDecls, decls, refs, cyclicNames) {
		return nil
	}

	declToFileIdx := buildDeclToFileIndex(fileDecls)
	children, moved := buildConsumerTree(decls, refs, cyclicNames, declToFileIdx)
	newOrder := computeDFSOrder(fileDecls, children, moved)

	if !orderChanged(newOrder) {
		return nil
	}

	return buildReorderFix(pass.Fset, file, src, fileDecls, newOrder)
}

func collectNonImportDecls(file *ast.File) []ast.Decl {
	var fileDecls []ast.Decl
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			continue
		}
		fileDecls = append(fileDecls, d)
	}
	return fileDecls
}

// hasMultiSpecViolation checks for GenDecls with multiple specs that contain violations.
// Reordering grouped declarations is unsafe.
func hasMultiSpecViolation(fileDecls []ast.Decl, decls []decl, refs map[string]map[string]bool, cyclicNames map[string]bool) bool {
	for _, d := range fileDecls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || len(gd.Specs) <= 1 {
			continue
		}
		for _, spec := range gd.Specs {
			for _, dd := range decls {
				if dd.node != spec {
					continue
				}
				consumer := findFirstConsumer(decls, dd, refs)
				if consumer != nil && !dd.exported && !cyclicNames[dd.name] {
					return true
				}
			}
		}
	}
	return false
}

func buildDeclToFileIndex(fileDecls []ast.Decl) map[string]int {
	declToFileIdx := make(map[string]int)
	for i, fd := range fileDecls {
		switch n := fd.(type) {
		case *ast.FuncDecl:
			declToFileIdx[n.Name.Name] = i
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						declToFileIdx[name.Name] = i
					}
				case *ast.TypeSpec:
					declToFileIdx[s.Name.Name] = i
				}
			}
		}
	}
	return declToFileIdx
}

func buildConsumerTree(decls []decl, refs map[string]map[string]bool, cyclicNames map[string]bool,
	declToFileIdx map[string]int) (map[int][]int, map[int]bool) {
	children := make(map[int][]int)
	moved := make(map[int]bool)

	for _, d := range decls {
		if d.exported || d.name == "init" || cyclicNames[d.name] {
			continue
		}
		if hasEagerReference(decls, d, refs) {
			continue
		}
		consumer := findFirstConsumer(decls, d, refs)
		if consumer == nil {
			continue
		}
		helperIdx := declToFileIdx[d.name]
		consumerIdx := declToFileIdx[consumer.name]
		if helperIdx < consumerIdx {
			children[consumerIdx] = append(children[consumerIdx], helperIdx)
			moved[helperIdx] = true
		}
	}

	return children, moved
}

func computeDFSOrder(fileDecls []ast.Decl, children map[int][]int, moved map[int]bool) []int {
	newOrder := make([]int, 0, len(fileDecls))
	var visit func(idx int)
	visit = func(idx int) {
		newOrder = append(newOrder, idx)
		for _, childIdx := range children[idx] {
			visit(childIdx)
		}
	}

	for i := range fileDecls {
		if !moved[i] {
			visit(i)
		}
	}
	return newOrder
}

func orderChanged(newOrder []int) bool {
	for i, idx := range newOrder {
		if idx != i {
			return true
		}
	}
	return false
}

func collectTopLevelDecls(file *ast.File) []decl {
	var decls []decl
	idx := 0
	for _, d := range file.Decls {
		switch n := d.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil {
				// Methods are handled separately
				continue
			}
			decls = append(decls, decl{
				name:     n.Name.Name,
				node:     n,
				index:    idx,
				exported: n.Name.IsExported(),
				kind:     "func",
			})
			idx++
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			for _, spec := range n.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.Name == "_" {
							continue
						}
						kind := "var"
						if n.Tok == token.CONST {
							kind = "const"
						}
						decls = append(decls, decl{
							name:     name.Name,
							node:     spec,
							index:    idx,
							exported: name.IsExported(),
							kind:     kind,
						})
					}
				case *ast.TypeSpec:
					decls = append(decls, decl{
						name:     s.Name.Name,
						node:     spec,
						index:    idx,
						exported: s.Name.IsExported(),
						kind:     "type",
					})
				}
			}
			idx++
		}
	}
	return decls
}

// buildRefGraph maps each declaration name to the set of other top-level names it references.
func buildRefGraph(pass *analysis.Pass, file *ast.File, decls []decl) map[string]map[string]bool {
	nameSet := make(map[string]bool)
	for _, d := range decls {
		nameSet[d.name] = true
	}

	refs := make(map[string]map[string]bool)

	for _, d := range file.Decls {
		switch n := d.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil {
				continue
			}
			declName := n.Name.Name
			if !nameSet[declName] {
				continue
			}
			refs[declName] = findReferencedNames(pass, n, nameSet, declName)
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			for _, spec := range n.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if !nameSet[name.Name] {
							continue
						}
						refs[name.Name] = findReferencedNamesInValueSpec(pass, s, nameSet, name.Name)
					}
				case *ast.TypeSpec:
					if !nameSet[s.Name.Name] {
						continue
					}
					refs[s.Name.Name] = findReferencedNames(pass, s, nameSet, s.Name.Name)
				}
			}
		}
	}

	return refs
}

func findReferencedNamesInValueSpec(pass *analysis.Pass, spec *ast.ValueSpec, nameSet map[string]bool, self string) map[string]bool {
	found := make(map[string]bool)
	// Check type expression
	if spec.Type != nil {
		for k, v := range findReferencedNames(pass, spec.Type, nameSet, self) {
			found[k] = v
		}
	}
	// Check value expressions
	for _, val := range spec.Values {
		for k, v := range findReferencedNames(pass, val, nameSet, self) {
			found[k] = v
		}
	}
	return found
}

func findReferencedNames(pass *analysis.Pass, node ast.Node, nameSet map[string]bool, self string) map[string]bool {
	found := make(map[string]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == self || !nameSet[ident.Name] {
			return true
		}
		obj := pass.TypesInfo.Uses[ident]
		if obj == nil {
			return true
		}
		if _, ok := obj.(*types.PkgName); ok {
			return true
		}
		if obj.Parent() == pass.Pkg.Scope() {
			found[ident.Name] = true
		}
		return true
	})
	return found
}

func findCyclicNames(decls []decl, refs map[string]map[string]bool) map[string]bool {
	// Only consider unexported helpers
	helperNames := make(map[string]bool)
	for _, d := range decls {
		if !d.exported {
			helperNames[d.name] = true
		}
	}

	cyclic := make(map[string]bool)
	for name := range helperNames {
		if canReachSelf(name, refs) {
			cyclic[name] = true
		}
	}
	return cyclic
}

func canReachSelf(start string, refs map[string]map[string]bool) bool {
	visited := make(map[string]bool)
	queue := make([]string, 0)
	for dep := range refs[start] {
		queue = append(queue, dep)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == start {
			return true
		}
		if visited[current] {
			continue
		}
		visited[current] = true
		for dep := range refs[current] {
			queue = append(queue, dep)
		}
	}
	return false
}

func findFirstConsumer(decls []decl, helper decl, refs map[string]map[string]bool) *decl {
	for i := range decls {
		d := &decls[i]
		if d.index <= helper.index {
			continue
		}
		if refs[d.name][helper.name] {
			return d
		}
	}
	return nil
}

func hasEagerReference(decls []decl, helper decl, refs map[string]map[string]bool) bool {
	// A reference is "eager" if it's used at package scope (not inside a function body).
	// In Go, this means it's used in a var initializer or const expression.
	// If a later var/const declaration uses this helper at package level, it's eager.
	for _, d := range decls {
		if d.index <= helper.index {
			continue
		}
		if (d.kind == "var" || d.kind == "const") && refs[d.name][helper.name] {
			return true
		}
	}
	return false
}

// decl represents a top-level declaration with metadata.
type decl struct {
	name     string
	node     ast.Node
	index    int
	exported bool
	kind     string // "func", "var", "const", "type"
}
