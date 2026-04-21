package readfriendlyorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "readfriendlyorder",
		Doc:      "enforce reading-friendly code organization",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      run,
	}
}

func run(pass *analysis.Pass) (any, error) {
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
		reportMethodOrdering(pass, file, src)
		reportTestOrdering(pass, file, src)
	}

	return nil, nil
}

func reportInitOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	firstNonInitFuncIdx, firstNonInitFuncName, violationDecl := findInitViolation(file)
	if violationDecl == nil {
		return
	}
	fixCtx := newFixContext(pass.Fset, file, src)
	diag := analysis.Diagnostic{
		Pos:     violationDecl.Pos(),
		Message: fmt.Sprintf("Place init() before %q for visibility.", firstNonInitFuncName),
	}
	fix := fixCtx.buildSwapFix(file.Decls[firstNonInitFuncIdx], violationDecl)
	if fix != nil {
		diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	pass.Report(diag)
}

func findInitViolation(file *ast.File) (int, string, ast.Decl) {
	var firstNonInitFuncIdx int
	var firstNonInitFuncName string
	var foundNonInit bool
	var violationDecl ast.Decl

	for i, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if fn.Name.Name != "init" {
			if !foundNonInit {
				firstNonInitFuncIdx = i
				firstNonInitFuncName = fn.Name.Name
				foundNonInit = true
			}
			continue
		}
		if foundNonInit {
			violationDecl = d
		}
	}

	return firstNonInitFuncIdx, firstNonInitFuncName, violationDecl
}

func reportTopLevelOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	ctx := &topLevelReorderContext{
		pass:  pass,
		file:  file,
		src:   src,
		decls: collectTopLevelDecls(file),
	}
	declNames := make(nameSet)
	for _, d := range ctx.decls {
		declNames[d.name] = true
	}
	ctx.refs = (&refFinder{pass: pass, declNames: declNames}).buildGraph(file)
	ctx.cyclicNames = ctx.refs.findCyclicNames(ctx.decls)

	violations := ctx.collectViolations()
	if len(violations) == 0 {
		return
	}

	fix := ctx.computeReorderFix()
	reportViolations(pass, violations, fix)
}

func (ctx *topLevelReorderContext) collectViolations() []violation {
	var violations []violation
	for _, d := range ctx.decls {
		if d.exported || d.name == "init" || ctx.cyclicNames[d.name] {
			continue
		}
		if ctx.refs.isUsedAtPackageScope(ctx.decls, d) {
			continue
		}
		consumer := ctx.refs.findFirstConsumer(ctx.decls, d)
		if consumer == nil {
			continue
		}
		violations = append(violations, violation{d, consumer})
	}
	return violations
}

func reportViolations(pass *analysis.Pass, violations []violation, fix *analysis.SuggestedFix) {
	for i, v := range violations {
		msg := buildViolationMessage(v)
		diag := analysis.Diagnostic{
			Pos:     v.d.node.Pos(),
			Message: msg,
		}
		if i == 0 && fix != nil {
			diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
		}
		pass.Report(diag)
	}
}

func buildViolationMessage(v violation) string {
	if v.d.kind == declKindConst || v.d.kind == declKindVar {
		return fmt.Sprintf("Place constant %q below the top-level symbol %q that uses it.",
			v.d.name, v.consumer.name)
	}
	return fmt.Sprintf("Place helper %q below the top-level symbol %q that depends on it.",
		v.d.name, v.consumer.name)
}

func (ctx *topLevelReorderContext) computeReorderFix() *analysis.SuggestedFix {
	fileDecls := collectNonImportDecls(ctx.file)
	if ctx.hasMultiSpecViolation(fileDecls) {
		return nil
	}

	declToFileIdx := buildDeclToFileIndex(fileDecls)
	children, moved := ctx.buildConsumerTree(declToFileIdx)
	newOrder := computeDFSOrder(fileDecls, children, moved)
	return newFixContext(ctx.pass.Fset, ctx.file, ctx.src).computeReorderFix(fileDecls, newOrder)
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

func (ctx *topLevelReorderContext) hasMultiSpecViolation(fileDecls []ast.Decl) bool {
	for _, d := range fileDecls {
		if ctx.isViolatingMultiSpecDecl(d) {
			return true
		}
	}
	return false
}

func (ctx *topLevelReorderContext) isViolatingMultiSpecDecl(d ast.Decl) bool {
	gd, ok := d.(*ast.GenDecl)
	if !ok || len(gd.Specs) <= 1 {
		return false
	}
	for _, spec := range gd.Specs {
		if ctx.isViolatingSpec(spec) {
			return true
		}
	}
	return false
}

func (ctx *topLevelReorderContext) isViolatingSpec(spec ast.Spec) bool {
	for _, dd := range ctx.decls {
		if dd.node != spec {
			continue
		}
		consumer := ctx.refs.findFirstConsumer(ctx.decls, dd)
		if consumer != nil && !dd.exported && !ctx.cyclicNames[dd.name] {
			return true
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

func (ctx *topLevelReorderContext) buildConsumerTree(declToFileIdx map[string]int) (helperPlacement, movedDecls) {
	children := make(helperPlacement)
	moved := make(movedDecls)

	for _, d := range ctx.decls {
		if d.exported || d.name == "init" || ctx.cyclicNames[d.name] {
			continue
		}
		if ctx.refs.isUsedAtPackageScope(ctx.decls, d) {
			continue
		}
		consumer := ctx.refs.findFirstConsumer(ctx.decls, d)
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

func computeDFSOrder(fileDecls []ast.Decl, children helperPlacement, moved movedDecls) []int {
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

func collectTopLevelDecls(file *ast.File) []decl {
	var decls []decl
	idx := 0
	for _, d := range file.Decls {
		switch n := d.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil {
				continue
			}
			decls = append(decls, decl{
				name:     n.Name.Name,
				node:     n,
				index:    idx,
				exported: n.Name.IsExported(),
				kind:     declKindFunc,
			})
			idx++
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			specDecls, newIdx := collectSpecDecls(n, idx)
			decls = append(decls, specDecls...)
			idx = newIdx
		}
	}
	return decls
}

func collectSpecDecls(gd *ast.GenDecl, idx int) ([]decl, int) {
	var decls []decl
	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			decls = append(decls, processValueSpec(s, gd.Tok, idx)...)
		case *ast.TypeSpec:
			decls = append(decls, decl{
				name:     s.Name.Name,
				node:     spec,
				index:    idx,
				exported: s.Name.IsExported(),
				kind:     declKindType,
			})
		}
		idx++
	}
	return decls, idx
}

func processValueSpec(spec *ast.ValueSpec, tok token.Token, idx int) []decl {
	var decls []decl
	kind := declKindVar
	if tok == token.CONST {
		kind = declKindConst
	}
	for _, name := range spec.Names {
		if name.Name == "_" {
			continue
		}
		decls = append(decls, decl{
			name:     name.Name,
			node:     spec,
			index:    idx,
			exported: name.IsExported(),
			kind:     kind,
		})
	}
	return decls
}

type refFinder struct {
	pass      *analysis.Pass
	declNames nameSet
}

func (finder *refFinder) buildGraph(file *ast.File) refGraph {
	refs := make(refGraph)
	for _, d := range file.Decls {
		switch n := d.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil {
				continue
			}
			refs[n.Name.Name] = finder.findIn(n, n.Name.Name)
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			finder.findInGenDecl(n, refs)
		}
	}
	return refs
}

func (finder *refFinder) findInGenDecl(gd *ast.GenDecl, refs refGraph) {
	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, name := range s.Names {
				if name.Name == "_" {
					continue
				}
				refs[name.Name] = finder.findIn(s, name.Name)
			}
		case *ast.TypeSpec:
			refs[s.Name.Name] = finder.findIn(s, s.Name.Name)
		}
	}
}

func (finder *refFinder) findIn(node ast.Node, self string) nameSet {
	found := make(nameSet)
	ast.Inspect(node, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == self || !finder.declNames[ident.Name] {
			return true
		}
		obj := finder.pass.TypesInfo.Uses[ident]
		if obj == nil {
			return true
		}
		if _, ok := obj.(*types.PkgName); ok {
			return true
		}
		if obj.Parent() == finder.pass.Pkg.Scope() {
			found[ident.Name] = true
		}
		return true
	})
	return found
}

func (refs refGraph) findCyclicNames(decls []decl) nameSet {
	helperNames := make(nameSet)
	for _, d := range decls {
		if !d.exported {
			helperNames[d.name] = true
		}
	}

	cyclic := make(nameSet)
	for name := range helperNames {
		if refs.canReachSelf(name) {
			cyclic[name] = true
		}
	}
	return cyclic
}

func (refs refGraph) canReachSelf(start string) bool {
	visited := make(nameSet)
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

func (refs refGraph) findFirstConsumer(decls []decl, helper decl) *decl {
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

func (refs refGraph) isUsedAtPackageScope(decls []decl, helper decl) bool {
	for _, d := range decls {
		if d.index <= helper.index {
			continue
		}
		if (d.kind == declKindVar || d.kind == declKindConst) && refs[d.name][helper.name] {
			return true
		}
	}
	return false
}

type violation struct {
	d        decl
	consumer *decl
}

type topLevelReorderContext struct {
	pass        *analysis.Pass
	file        *ast.File
	src         []byte
	decls       []decl
	refs        refGraph
	cyclicNames nameSet
}

// decl represents a top-level declaration with metadata.
type decl struct {
	name     string
	node     ast.Node
	index    int
	exported bool
	kind     declKind
}

type declKind string

const (
	declKindFunc  declKind = "func"
	declKindType  declKind = "type"
	declKindVar   declKind = "var"
	declKindConst declKind = "const"
)

type refGraph map[string]nameSet

type nameSet map[string]bool

// helperPlacement maps each declaration's index to the helper indices that follow it in the reordered output.
type helperPlacement map[int][]int

// movedDecls tracks which declaration indices have been relocated to follow their consumer.
type movedDecls map[int]bool
