package nounicodeescape

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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

		if !unicodeEscapeRe.MatchString(lit.Value) {
			return
		}

		diag := analysis.Diagnostic{
			Pos:     lit.Pos(),
			End:     lit.End(),
			Message: `Use the actual character instead of a \uXXXX escape sequence.`,
		}

		if fixed, ok := buildFixedLiteral(lit.Value); ok {
			diag.SuggestedFixes = []analysis.SuggestedFix{{
				Message: "Replace escape sequences with literal characters",
				TextEdits: []analysis.TextEdit{{
					Pos:     lit.Pos(),
					End:     lit.End(),
					NewText: []byte(fixed),
				}},
			}}
		}

		pass.Report(diag)
	})

	return nil, nil
}

// buildFixedLiteral replaces all \uXXXX / \U00XXXXXX escapes with literal characters.
// Uses all-or-nothing: if any escape is unsafe to inline, returns ("", false).
func buildFixedLiteral(raw string) (string, bool) {
	delimiter := raw[0]
	matches := unicodeEscapeRe.FindAllStringIndex(raw, -1)
	if len(matches) == 0 {
		return "", false
	}

	// Check all escapes are safe before replacing any
	for _, m := range matches {
		escape := raw[m[0]:m[1]]
		r, err := decodeEscape(escape)
		if err != nil || !isSafeToInline(r, delimiter) {
			return "", false
		}
	}

	// All safe — build the replacement
	var b strings.Builder
	prev := 0
	for _, m := range matches {
		b.WriteString(raw[prev:m[0]])
		escape := raw[m[0]:m[1]]
		r, _ := decodeEscape(escape)
		b.WriteRune(r)
		prev = m[1]
	}
	b.WriteString(raw[prev:])

	return b.String(), true
}

func decodeEscape(escape string) (rune, error) {
	// Convert \uXXXX or \U00XXXXXX to the rune
	// strconv.Unquote handles these when wrapped in quotes
	quoted := `"` + escape + `"`
	s, err := strconv.Unquote(quoted)
	if err != nil {
		return 0, err
	}
	r := []rune(s)
	if len(r) != 1 {
		return 0, strconv.ErrSyntax
	}
	return r[0], nil
}

var unicodeEscapeRe = regexp.MustCompile(`\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}`)

func isSafeToInline(r rune, delimiter byte) bool {
	// Control characters should stay escaped
	if r < 0x20 || r == 0x7F {
		return false
	}
	// Don't inline characters that would break the string delimiter
	if r == rune(delimiter) {
		return false
	}
	// Don't inline backslash
	if r == '\\' {
		return false
	}
	// Don't inline characters in the "other" Unicode category (invisible/format chars)
	if unicode.Is(unicode.Cf, r) {
		return false
	}
	return true
}
