package tests

import (
	"testing"

	"github.com/skhoroshavin/gounslop/tests/rule"
	"github.com/stretchr/testify/suite"
)

type NounicodeescapeE2ESuite struct {
	rule.Suite
}

func (s *NounicodeescapeE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.ModulePath = "example.com/mod"
}

func TestNounicodeescapeE2E(t *testing.T) {
	suite.Run(t, new(NounicodeescapeE2ESuite))
}

func (s *NounicodeescapeE2ESuite) TestLiteralUnicodePasses() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"é\"",
		"}",
	)
	s.ShouldPass()
}

func (s *NounicodeescapeE2ESuite) TestEscapeFlagged() {
	s.LintCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"\\u00e9\"",
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
		"\t_ = \"\\U000000e9\"",
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
		"\t_ = \"\\u00e9\"",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"é\"",
		"}",
	)
}

func (s *NounicodeescapeE2ESuite) TestMixedSafeUnsafeFix() {
	s.FixCode(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello \\u00e9\\u0001 world\"",
		"}",
	)
	s.ShouldFailWith()
	s.ShouldProduce(
		"package main",
		"",
		"func main() {",
		"\t_ = \"hello \\u00e9\\u0001 world\"",
		"}",
	)
}
