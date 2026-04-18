package nofalsesharing_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

const pluginE2EModulePath = "example.com/plugin-e2e"

type PluginE2ESuite struct {
	suite.Suite
}

func TestPluginE2E(t *testing.T) {
	suite.Run(t, new(PluginE2ESuite))
}

func (s *PluginE2ESuite) TestDirMode_SharedBySingleConsumer() {
	s.runScenario(ruletest.Scenario{
		Name:       "dir mode single consumer flagged",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featureb/consumer.go": "package featureb\n\nfunc Use() {}\n",
			"shared/util.go":       "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "dir",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"shared/util.go",
				"only used by: featurea",
				"Must be used by 2+ entities",
			},
		},
	})
}

func (s *PluginE2ESuite) TestDirMode_MultipleConsumersPass() {
	s.runScenario(ruletest.Scenario{
		Name:       "dir mode multiple consumers pass",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featureb/consumer.go": "package featureb\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"shared/util.go":       "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "dir",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *PluginE2ESuite) TestFileMode_TwoConsumersPass() {
	s.runScenario(ruletest.Scenario{
		Name:       "file mode two consumers pass",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featureb/consumer.go": "package featureb\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"shared/util.go":       "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "file",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *PluginE2ESuite) TestFileMode_SingleConsumerFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "file mode single consumer flagged",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featureb/consumer.go": "package featureb\n\nfunc Use() {}\n",
			"shared/util.go":       "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "file",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"only used by",
			},
		},
	})
}

func (s *PluginE2ESuite) TestFileMode_TestFilesDontCountAsConsumers() {
	s.runScenario(ruletest.Scenario{
		Name:       "file mode test files don't count as consumers",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go":      "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featurea/consumer_test.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"shared/util.go":            "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "file",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"only used by",
			},
		},
	})
}

func (s *PluginE2ESuite) TestFileMode_NotImportedByAnyone() {
	s.runScenario(ruletest.Scenario{
		Name:       "file mode not imported by anyone",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nfunc Use() {}\n",
			"shared/util.go":       "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "file",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"not imported by any entity",
			},
		},
	})
}

func (s *PluginE2ESuite) TestDirMode_TwoFilesInSameDir_OneConsumer() {
	s.runScenario(ruletest.Scenario{
		Name:       "dir mode two files in same dir one consumer",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumerA.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featurea/consumerB.go": "package featurea\n\nfunc Use() {}\n",
			"shared/util.go":        "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "dir",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"only used by: featurea",
			},
		},
	})
}

func (s *PluginE2ESuite) TestFileMode_TwoFilesInSameDir_TwoConsumers() {
	s.runScenario(ruletest.Scenario{
		Name:       "file mode two files in same dir two consumers",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumerA.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"featurea/consumerB.go": "package featurea\n\nimport _ \"example.com/plugin-e2e/shared\"\n",
			"shared/util.go":        "package shared\n\nfunc Value() string { return \"x\" }\n",
		},
		Settings: map[string]any{
			"shared-dirs": "shared",
			"mode":        "file",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *PluginE2ESuite) TestInvalidSettingsTypeFailsClearly() {
	s.runScenario(ruletest.Scenario{
		Name:       "invalid settings type fails clearly",
		ModulePath: pluginE2EModulePath,
		Linter:     "nofalsesharing",
		Files: map[string]string{
			"featurea/consumer.go": "package featurea\n\nfunc Use() {}\n",
		},
		Settings: map[string]any{
			"shared-dirs": []any{"shared"},
		},
		Expect: ruletest.Expectation{
			ExitCode: 3,
			OutputContains: []string{
				"nofalsesharing",
				"shared-dirs",
			},
		},
	})
}

func (s *PluginE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
