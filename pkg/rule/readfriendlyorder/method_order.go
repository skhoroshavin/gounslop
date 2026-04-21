package readfriendlyorder

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func reportMethodOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	ctx := &methodOrderContext{
		pass:   pass,
		file:   file,
		fixCtx: newFixContext(pass.Fset, file, src),
		data:   collectMethodOrderData(file),
	}

	for typeName, meths := range ctx.data.methods {
		if len(meths) == 0 {
			continue
		}

		reportConstructorOrdering(ctx, typeName)
		reportMethodDependencyOrder(pass, ctx.fixCtx, meths)
	}
}

func reportConstructorOrdering(ctx *methodOrderContext, typeName string) {
	ctor, ok := ctx.data.constructors[typeName]
	if !ok {
		return
	}

	typeIdx, hasType := ctx.data.typeDeclIdx[typeName]
	ctorIdx, hasCtor := ctx.data.constructorDeclIdx[typeName]
	if !hasType || !hasCtor || ctorIdx <= typeIdx+1 {
		return
	}

	diag := analysis.Diagnostic{
		Pos: ctor.Pos(),
		Message: fmt.Sprintf("Place constructor %q right after type %q declaration.",
			ctor.Name.Name, typeName),
	}
	targetDecl := ctx.file.Decls[typeIdx]
	_, insertOffset := ctx.fixCtx.declRange(targetDecl)
	fix := ctx.fixCtx.buildMoveFix(ctor, insertOffset)
	if fix != nil {
		diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	ctx.pass.Report(diag)
}

func collectMethodOrderData(file *ast.File) *methodOrderData {
	data := &methodOrderData{
		typeDeclIdx:        make(map[string]int),
		methods:            make(map[string][]methodEntry),
		constructors:       make(map[string]*ast.FuncDecl),
		constructorDeclIdx: make(map[string]int),
	}

	for i, d := range file.Decls {
		data.collectDecl(d, i)
	}

	return data
}

func (data *methodOrderData) collectDecl(d ast.Decl, idx int) {
	switch n := d.(type) {
	case *ast.GenDecl:
		data.collectTypeDecls(n, idx)
	case *ast.FuncDecl:
		data.collectFunc(n, idx)
	}
}

func (data *methodOrderData) collectTypeDecls(decl *ast.GenDecl, idx int) {
	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if ok {
			data.typeDeclIdx[ts.Name.Name] = idx
		}
	}
}

func (data *methodOrderData) collectFunc(fn *ast.FuncDecl, idx int) {
	if fn.Recv != nil {
		data.addMethodEntry(fn, idx)
		return
	}
	if !strings.HasPrefix(fn.Name.Name, "New") {
		return
	}
	data.addConstructor(fn, idx)
}

func (data *methodOrderData) addMethodEntry(fn *ast.FuncDecl, idx int) {
	typeName := receiverTypeName(fn)
	if typeName == "" {
		return
	}
	data.methods[typeName] = append(data.methods[typeName], methodEntry{
		name:  fn.Name.Name,
		node:  fn,
		index: idx,
	})
}

func (data *methodOrderData) addConstructor(fn *ast.FuncDecl, idx int) {
	typeName := fn.Name.Name[3:]
	if typeName == "" {
		return
	}
	data.constructors[typeName] = fn
	data.constructorDeclIdx[typeName] = idx
}

func reportMethodDependencyOrder(pass *analysis.Pass, fixCtx *fixContext, meths []methodEntry) {
	for _, m := range meths {
		consumer := findFirstMethodConsumer(meths, m)
		if consumer != nil {
			diag := analysis.Diagnostic{
				Pos: m.node.Pos(),
				Message: fmt.Sprintf("Place method %q below method %q that depends on it.",
					m.name, consumer.name),
			}
			fix := fixCtx.buildSwapFix(m.node, consumer.node)
			if fix != nil {
				diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
			}
			pass.Report(diag)
		}
	}
}

func findFirstMethodConsumer(meths []methodEntry, method methodEntry) *methodEntry {
	for i := range meths {
		candidate := &meths[i]
		if candidate.index <= method.index {
			continue
		}
		if methodCallsMethod(candidate.node, method.name) {
			return candidate
		}
	}
	return nil
}

type methodOrderContext struct {
	pass   *analysis.Pass
	file   *ast.File
	fixCtx *fixContext
	data   *methodOrderData
}

type methodOrderData struct {
	typeDeclIdx        map[string]int
	methods            map[string][]methodEntry
	constructors       map[string]*ast.FuncDecl
	constructorDeclIdx map[string]int
}

type methodEntry struct {
	name  string
	node  *ast.FuncDecl
	index int
}

func methodCallsMethod(caller *ast.FuncDecl, calleeName string) bool {
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
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
