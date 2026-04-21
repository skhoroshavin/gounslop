package astutil

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func WalkStringLiterals(pass *analysis.Pass, visit func(lit *ast.BasicLit)) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{(*ast.BasicLit)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		lit := n.(*ast.BasicLit)
		if lit.Kind != token.STRING && lit.Kind != token.CHAR {
			return
		}
		visit(lit)
	})
}
