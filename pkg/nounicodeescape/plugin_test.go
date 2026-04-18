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

func (s *NounicodeescapeE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
