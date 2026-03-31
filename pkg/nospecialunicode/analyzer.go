package nospecialunicode

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "nospecialunicode",
	Doc:      "disallow special unicode punctuation and whitespace characters in strings",
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

		var value string
		if lit.Kind == token.CHAR {
			// Unquote rune literal
			v, _, _, err := strconv.UnquoteChar(lit.Value[1:len(lit.Value)-1], '\'')
			if err != nil {
				return
			}
			value = string(v)
		} else {
			var err error
			value, err = strconv.Unquote(lit.Value)
			if err != nil {
				return
			}
		}

		for _, bc := range bannedChars {
			if strings.ContainsRune(value, bc.char) {
				pass.Reportf(lit.Pos(), "String contains %s (U+%04X). Use the ASCII equivalent.", bc.name, bc.char)
			}
		}
	})

	return nil, nil
}

type bannedChar struct {
	char rune
	name string
}

var bannedChars = []bannedChar{
	{'\u201C', "left double quotation mark"},
	{'\u201D', "right double quotation mark"},
	{'\u2018', "left single quotation mark"},
	{'\u2019', "right single quotation mark"},
	{'\u00A0', "non-breaking space"},
	{'\u202F', "narrow no-break space"},
	{'\u2007', "figure space"},
	{'\u2008', "punctuation space"},
	{'\u2009', "thin space"},
	{'\u200A', "hair space"},
	{'\u200B', "zero-width space"},
	{'\u2002', "en space"},
	{'\u2003', "em space"},
	{'\u205F', "medium mathematical space"},
	{'\u3000', "ideographic space"},
	{'\uFEFF', "zero-width no-break space"},
	{'\u2013', "en dash"},
	{'\u2014', "em dash"},
	{'\u2026', "horizontal ellipsis"},
}

// FormatDiagnostic returns the expected diagnostic message for a banned character.
// Exported for use in tests.
func FormatDiagnostic(name string, char rune) string {
	return fmt.Sprintf("String contains %s (U+%04X). Use the ASCII equivalent.", name, char)
}
