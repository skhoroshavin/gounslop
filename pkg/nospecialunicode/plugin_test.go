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

func (s *NospecialunicodeE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
