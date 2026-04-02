package readfriendlyorder

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

func reportMethodOrdering(pass *analysis.Pass, file *ast.File, insp *inspector.Inspector, src []byte) {
	// Collect type declarations and their positions in the file
	typePositions := make(map[string]int) // type name -> declaration order index
	typeDeclIdx := make(map[string]int)   // type name -> file.Decls index
	methods := make(map[string][]methodEntry)
	constructors := make(map[string]*ast.FuncDecl) // type name -> New* func
	constructorDeclIdx := make(map[string]int)     // type name -> file.Decls index of constructor

	declIdx := 0
	for i, d := range file.Decls {
		switch n := d.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					typePositions[ts.Name.Name] = declIdx
					typeDeclIdx[ts.Name.Name] = i
				}
			}
		case *ast.FuncDecl:
			if n.Recv != nil {
				typeName := receiverTypeName(n)
				if typeName != "" {
					methods[typeName] = append(methods[typeName], methodEntry{
						name:     n.Name.Name,
						node:     n,
						index:    declIdx,
						exported: n.Name.IsExported(),
					})
				}
			} else if strings.HasPrefix(n.Name.Name, "New") {
				typeName := n.Name.Name[3:]
				if typeName != "" {
					constructors[typeName] = n
					constructorDeclIdx[typeName] = i
				}
			}
		}
		declIdx++
	}

	for typeName, meths := range methods {
		if len(meths) == 0 {
			continue
		}

		// Check constructor placement
		if ctor, ok := constructors[typeName]; ok {
			if typePos, ok := typePositions[typeName]; ok {
				ctorIdx := -1
				for _, d := range file.Decls {
					if d == ctor {
						break
					}
					ctorIdx++
				}
				ctorIdx++
				if ctorIdx > typePos+1 {
					diag := analysis.Diagnostic{
						Pos: ctor.Pos(),
						Message: fmt.Sprintf("Place constructor %q right after type %q declaration.",
							ctor.Name.Name, typeName),
					}
					// Move constructor to right after type declaration
					if ti, ok := typeDeclIdx[typeName]; ok {
						targetDecl := file.Decls[ti]
						_, insertOffset := declRange(pass.Fset, src, targetDecl)
						fix := buildMoveFix(pass.Fset, file, src, ctor, insertOffset)
						if fix != nil {
							diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
						}
					}
					pass.Report(diag)
				}
			}
		}

		// Check method dependency order
		reportMethodDependencyOrder(pass, meths, src)
	}
}

func reportMethodDependencyOrder(pass *analysis.Pass, meths []methodEntry, src []byte) {
	for _, m := range meths {
		consumer := findFirstMethodConsumer(pass, meths, m)
		if consumer != nil {
			diag := analysis.Diagnostic{
				Pos: m.node.Pos(),
				Message: fmt.Sprintf("Place method %q below method %q that depends on it.",
					m.name, consumer.name),
			}
			file := pass.Files[0] // methods are within a single file
			for _, f := range pass.Files {
				if f.Pos() <= m.node.Pos() && m.node.Pos() < f.End() {
					file = f
					break
				}
			}
			fix := buildSwapFix(pass.Fset, file, src, m.node, consumer.node)
			if fix != nil {
				diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
			}
			pass.Report(diag)
		}
	}
}

func findFirstMethodConsumer(pass *analysis.Pass, meths []methodEntry, method methodEntry) *methodEntry {
	for i := range meths {
		candidate := &meths[i]
		if candidate.index <= method.index {
			continue
		}
		if methodCallsMethod(pass, candidate.node, method.name) {
			return candidate
		}
	}
	return nil
}

type methodEntry struct {
	name     string
	node     *ast.FuncDecl
	index    int
	exported bool
}

func methodCallsMethod(pass *analysis.Pass, caller *ast.FuncDecl, calleeName string) bool {
	if caller.Body == nil {
		return false
	}
	found := false
	ast.Inspect(caller.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == calleeName {
			found = true
		}
		return true
	})
	return found
}

func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	// Handle pointer receiver
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
