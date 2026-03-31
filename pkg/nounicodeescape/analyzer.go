package nounicodeescape

import (
	"go/ast"
	"go/token"
	"regexp"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "nounicodeescape",
	Doc:      "prefer literal unicode characters over escape sequences in strings",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.BasicLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		lit := n.(*ast.BasicLit)
		if lit.Kind != token.STRING && lit.Kind != token.CHAR {
			return
		}

		// Only check interpreted strings (starting with ") and char literals (starting with ')
		// Raw strings (starting with `) cannot contain escape sequences
		if len(lit.Value) > 0 && lit.Value[0] == '`' {
			return
		}

		if unicodeEscapeRe.MatchString(lit.Value) {
			pass.Reportf(lit.Pos(), `Use the actual character instead of a \uXXXX escape sequence.`)
		}
	})

	return nil, nil
}

var unicodeEscapeRe = regexp.MustCompile(`\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}`)
