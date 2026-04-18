package nodeepimports_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

const nodeepimportsModulePath = "example.com/mod"

type NodeepimportsE2ESuite struct {
	suite.Suite
}

func TestPluginE2E(t *testing.T) {
	suite.Run(t, new(NodeepimportsE2ESuite))
}

func (s *NodeepimportsE2ESuite) TestOneLevelDeepImportPasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "one-level deep import passes",
		ModulePath: nodeepimportsModulePath,
		Linter:     "nodeepimports",
		Files: map[string]string{
			"feature/consumer.go":    "package feature\n\nimport _ \"example.com/mod/feature/child\"\n",
			"feature/child/child.go": "package child\n\nvar X = 1\n",
		},
		Settings: map[string]any{
			"module-root": "example.com/mod",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NodeepimportsE2ESuite) TestDeepImportFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "deep import flagged",
		ModulePath: nodeepimportsModulePath,
		Linter:     "nodeepimports",
		Files: map[string]string{
			"feature/consumer.go":        "package feature\n\nimport _ \"example.com/mod/feature/child/deep\"\n",
			"feature/child/child.go":     "package child\n\nvar X = 1\n",
			"feature/child/deep/deep.go": "package deep\n\nvar Y = 2\n",
		},
		Settings: map[string]any{
			"module-root": "example.com/mod",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"too deep",
			},
		},
	})
}

func (s *NodeepimportsE2ESuite) TestDifferentTopLevelScopePasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "different top-level scope passes",
		ModulePath: nodeepimportsModulePath,
		Linter:     "nodeepimports",
		Files: map[string]string{
			"featurea/consumer.go":        "package featurea\n\nimport _ \"example.com/mod/featureb/other/deep\"\n",
			"featureb/other/deep/deep.go": "package deep\n\nvar Z = 3\n",
		},
		Settings: map[string]any{
			"module-root": "example.com/mod",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NodeepimportsE2ESuite) TestFileImportFromChildScopePasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "test file import from child scope passes",
		ModulePath: nodeepimportsModulePath,
		Linter:     "nodeepimports",
		Files: map[string]string{
			"feature/child/child.go":     "package child\n\nimport _ \"example.com/mod/feature/child/deep\"\n",
			"feature/child/deep/deep.go": "package deep\n\nvar Y = 2\n",
		},
		Settings: map[string]any{
			"module-root": "example.com/mod",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *NodeepimportsE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
