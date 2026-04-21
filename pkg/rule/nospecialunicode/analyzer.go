package nospecialunicode

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/skhoroshavin/gounslop/pkg/core/astutil"
)

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "nospecialunicode",
		Doc:      "disallow special unicode punctuation and whitespace characters in strings",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      run,
	}
}

func run(pass *analysis.Pass) (any, error) {
	astutil.WalkStringLiterals(pass, func(lit *ast.BasicLit) {
		value, ok := unquoteLiteral(lit)
		if !ok {
			return
		}

		fixed, fixable := buildFixedLiteral(lit.Kind, lit.Value, value)

		for _, bc := range bannedChars {
			if !strings.ContainsRune(value, bc.char) {
				continue
			}
			diag := analysis.Diagnostic{
				Pos:     lit.Pos(),
				End:     lit.End(),
				Message: formatDiagnostic(bc.name, bc.char),
			}
			if fixable {
				diag.SuggestedFixes = []analysis.SuggestedFix{{
					Message: "Replace with ASCII equivalent",
					TextEdits: []analysis.TextEdit{{
						Pos:     lit.Pos(),
						End:     lit.End(),
						NewText: []byte(fixed),
					}},
				}}
			}
			pass.Report(diag)
		}
	})

	return nil, nil
}

type bannedChar struct {
	char        rune
	name        string
	replacement string
}

var bannedChars = []bannedChar{
	{'\u201C', "left double quotation mark", `"`},
	{'\u201D', "right double quotation mark", `"`},
	{'\u2018', "left single quotation mark", "'"},
	{'\u2019', "right single quotation mark", "'"},
	{'\u00A0', "non-breaking space", " "},
	{'\u202F', "narrow no-break space", " "},
	{'\u2007', "figure space", " "},
	{'\u2008', "punctuation space", " "},
	{'\u2009', "thin space", " "},
	{'\u200A', "hair space", " "},
	{'\u200B', "zero-width space", ""},
	{'\u2002', "en space", " "},
	{'\u2003', "em space", " "},
	{'\u205F', "medium mathematical space", " "},
	{'\u3000', "ideographic space", " "},
	{'\uFEFF', "zero-width no-break space", ""},
	{'\u2013', "en dash", "-"},
	{'\u2014', "em dash", "-"},
	{'\u2026', "horizontal ellipsis", "..."},
}

func unquoteLiteral(lit *ast.BasicLit) (string, bool) {
	if lit.Kind == token.CHAR {
		v, _, _, err := strconv.UnquoteChar(lit.Value[1:len(lit.Value)-1], '\'')
		if err != nil {
			return "", false
		}
		return string(v), true
	}

	if lit.Value[0] == '`' {
		return lit.Value[1 : len(lit.Value)-1], true
	}

	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func buildFixedLiteral(kind token.Token, raw string, value string) (string, bool) {
	delimiter := raw[0]

	anyReplaced := false
	var b strings.Builder
	for _, r := range value {
		repl, isBanned := replacementMap[r]
		if !isBanned {
			b.WriteRune(r)
			continue
		}
		if !isReplacementSafe(repl, delimiter) {
			b.WriteRune(r)
			continue
		}
		b.WriteString(repl)
		anyReplaced = true
	}

	if !anyReplaced {
		return "", false
	}

	replaced := b.String()

	switch {
	case kind == token.CHAR:
		return strconv.QuoteRune([]rune(replaced)[0]), true
	case delimiter == '`':
		return "`" + replaced + "`", true
	default:
		return strconv.Quote(replaced), true
	}
}

// replacementMap builds a char -> replacement lookup.
var replacementMap = func() map[rune]string {
	m := make(map[rune]string, len(bannedChars))
	for _, bc := range bannedChars {
		m[bc.char] = bc.replacement
	}
	return m
}()

func isReplacementSafe(replacement string, delimiter byte) bool {
	if replacement == `"` && delimiter == '"' {
		return false
	}
	if replacement == "'" && delimiter == '\'' {
		return false
	}
	return true
}

func formatDiagnostic(name string, char rune) string {
	return fmt.Sprintf("String contains %s (U+%04X). Use the ASCII equivalent.", name, char)
}
