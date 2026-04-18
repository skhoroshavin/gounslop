package nounicodeescape_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type NounicodeescapeE2ESuite struct {
	ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
	s := new(NounicodeescapeE2ESuite)
	s.Linter = "nounicodeescape"
	s.ModulePath = "example.com/mod"
	suite.Run(t, s)
}

func (s *NounicodeescapeE2ESuite) TestLiteralUnicodePasses() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"—\"",
		"}",
	)
	s.ShouldPass()
}

func (s *NounicodeescapeE2ESuite) TestEscapeFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\u2014\"",
		"}",
	)
	s.ShouldFailWith("\\uXXXX")
}

func (s *NounicodeescapeE2ESuite) TestRawStringNotFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = `\\u2014`",
		"}",
	)
	s.ShouldPass()
}

func (s *NounicodeescapeE2ESuite) TestLongEscapeFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\U00002014\"",
		"}",
	)
	s.ShouldFailWith("\\uXXXX")
}

func (s *NounicodeescapeE2ESuite) TestControlCharEscapeNoFix() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\u0001\"",
		"}",
	)
	s.ShouldFailWith("\\uXXXX")
}

func (s *NounicodeescapeE2ESuite) TestDoubleQuoteEscapeNoFix() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\u0022\"",
		"}",
	)
	s.ShouldFailWith("\\uXXXX")
}

func (s *NounicodeescapeE2ESuite) TestUnicodeEscapeFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\u2014\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"—\"",
		"}",
	)
}

func (s *NounicodeescapeE2ESuite) TestMixedSafeUnsafeFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello \\u2014\\u0001 world\"",
		"}",
	)
	s.ShouldFailWith()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello \\u2014\\u0001 world\"",
		"}",
	)
}
