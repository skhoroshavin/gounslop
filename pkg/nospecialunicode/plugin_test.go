package nospecialunicode_test

import (
	"strings"
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type NospecialunicodeE2ESuite struct {
	ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
	s := new(NospecialunicodeE2ESuite)
	s.Linter = "nospecialunicode"
	s.ModulePath = "example.com/mod"
	suite.Run(t, s)
}

func zeroWidthString(value string) string {
	return strings.ReplaceAll(value, `\u200b`, "\u200b")
}

func (s *NospecialunicodeE2ESuite) TestASCIIStringPasses() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"plain ascii text\"",
		"\t_ = \"hello - world ... 'quoted' \\\"double\\\"\"",
		"}",
	)
	s.ShouldPass()
}

func (s *NospecialunicodeE2ESuite) TestSpecialUnicodeFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a — b\"",
		"}",
	)
	s.ShouldFailWith("em dash", "U+2014")
}

func (s *NospecialunicodeE2ESuite) TestRawStringFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = `hello — world`",
		"}",
	)
	s.ShouldFailWith("em dash", "U+2014")
}

func (s *NospecialunicodeE2ESuite) TestMultipleBannedCharacters() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a — b\"",
		"\t_ = \"c – d\"",
		"}",
	)
	s.ShouldFailWith("en dash", "em dash")
}

func (s *NospecialunicodeE2ESuite) TestNonBreakingSpaceFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello world\"",
		"}",
	)
	s.ShouldFailWith("non-breaking space", "U+00A0")
}

func (s *NospecialunicodeE2ESuite) TestZeroWidthSpaceFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		zeroWidthString("\t_ = \"hello\\u200bworld\""),
		"}",
	)
	s.ShouldFailWith("zero-width space", "U+200B")
}

func (s *NospecialunicodeE2ESuite) TestCurlyQuotesFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"“hello”\"",
		"}",
	)
	s.ShouldFailWith("left double quotation mark")
}

func (s *NospecialunicodeE2ESuite) TestRuneLiteralFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = '—'",
		"}",
	)
	s.ShouldFailWith("em dash")
}

func (s *NospecialunicodeE2ESuite) TestMultipleBannedInSingleLiteral() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"“hello”…\"",
		"}",
	)
	s.ShouldFailWith("left double quotation mark")
}

func (s *NospecialunicodeE2ESuite) TestEmDashFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a — b\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a - b\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestEnDashFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a – b\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"a - b\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestEllipsisFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"wait…\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"wait...\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestCurlyQuotesFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"‘hello’\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"'hello'\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestNBSPFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello world\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello world\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestZeroWidthSpaceFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		zeroWidthString("\t_ = \"hello\\u200bworld\""),
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"helloworld\"",
		"}",
	)
}

func (s *NospecialunicodeE2ESuite) TestRawStringFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = `hello — world`",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = `hello - world`",
		"}",
	)
}
