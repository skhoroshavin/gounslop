package nospecialunicode_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

const nospecialunicodeModulePath = "example.com/mod"

type NospecialunicodeE2ESuite struct {
	suite.Suite
}

func TestPluginE2E(t *testing.T) {
	suite.Run(t, new(NospecialunicodeE2ESuite))
}

func (s *NospecialunicodeE2ESuite) TestASCIIStringPasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "ASCII string passes",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"plain ascii text\"\n\t_ = \"hello - world ... 'quoted' \\\"double\\\"\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestSpecialUnicodeFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "special Unicode flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"a \u2014 b\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"em dash",
				"U+2014",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestRawStringFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "raw string flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = `hello \u2014 world`\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"em dash",
				"U+2014",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestMultipleBannedCharacters() {
	s.runScenario(ruletest.Scenario{
		Name:       "multiple banned characters in separate literals",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"a \u2014 b\"\n\t_ = \"c \u2013 d\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"en dash",
				"em dash",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestNonBreakingSpaceFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "non-breaking space flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"hello\u00A0world\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"non-breaking space",
				"U+00A0",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestZeroWidthSpaceFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "zero-width space flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"hello\u200Bworld\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"zero-width space",
				"U+200B",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestCurlyQuotesFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "curly quotes flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\u201Chello\u201D\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"left double quotation mark",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestRuneLiteralFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "rune literal flagged",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = '\u2014'\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"em dash",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestMultipleBannedInSingleLiteral() {
	s.runScenario(ruletest.Scenario{
		Name:       "multiple banned chars in a single literal",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\u201Chello\u201D\u2026\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"left double quotation mark",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestEmDashFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "em dash fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"a \u2014 b\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"a - b\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestEnDashFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "en dash fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"a \u2013 b\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"a - b\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestEllipsisFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "ellipsis fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"wait\u2026\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"wait...\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestCurlyQuotesFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "curly quotes fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\u2018hello\u2019\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"'hello'\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestNBSPFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "NBSP fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"hello\u00A0world\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"hello world\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestZeroWidthSpaceFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "zero-width space fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"hello\u200Bworld\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"helloworld\"\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) TestRawStringFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "raw string em dash fix",
		ModulePath: nospecialunicodeModulePath,
		Linter:     "nospecialunicode",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = `hello \u2014 world`\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = `hello - world`\n}\n",
			},
		},
	})
}

func (s *NospecialunicodeE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}

func (s *NospecialunicodeE2ESuite) runFixScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.ExecuteFix(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
