package nounicodeescape_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

const nounicodeescapeModulePath = "example.com/mod"

type NounicodeescapeE2ESuite struct {
	suite.Suite
}

func TestPluginE2E(t *testing.T) {
	suite.Run(t, new(NounicodeescapeE2ESuite))
}

func (s *NounicodeescapeE2ESuite) TestLiteralUnicodePasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "literal Unicode passes",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\u2014\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestEscapeFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "unicode escape flagged",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\\u2014\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"\\uXXXX",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestRawStringNotFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "raw string not flagged",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = `\\u2014`\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestLongEscapeFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "long unicode escape flagged",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\\U00002014\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"\\uXXXX",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestControlCharEscapeNoFix() {
	s.runScenario(ruletest.Scenario{
		Name:       "control character escape flagged without fix",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\\u0001\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"\\uXXXX",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestDoubleQuoteEscapeNoFix() {
	s.runScenario(ruletest.Scenario{
		Name:       "double quote escape in string no fix",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\\u0022\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"\\uXXXX",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestUnicodeEscapeFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "unicode escape fix replaces with literal",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"\\u2014\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"\u2014\"\n}\n",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) TestMixedSafeUnsafeFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "mixed safe/unsafe escapes: entire literal stays escaped",
		ModulePath: nounicodeescapeModulePath,
		Linter:     "nounicodeescape",
		Files: map[string]string{
			"main.go": "package main\n\nfunc main() {\n\t_ = \"hello \\u2014\\u0001 world\"\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			FixedFiles: map[string]string{
				"main.go": "package main\n\nfunc main() {\n\t_ = \"hello \\u2014\\u0001 world\"\n}\n",
			},
		},
	})
}

func (s *NounicodeescapeE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}

func (s *NounicodeescapeE2ESuite) runFixScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.ExecuteFix(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
