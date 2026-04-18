package nofalsesharing_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type PluginE2ESuite struct {
	ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
	s := new(PluginE2ESuite)
	s.Linter = "nofalsesharing"
	s.ModulePath = "example.com/plugin-e2e"
	suite.Run(t, s)
}

func (s *PluginE2ESuite) TestDirMode_SharedBySingleConsumer() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "dir",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "only used by: featurea", "Must be used by 2+ entities")
}

func (s *PluginE2ESuite) TestDirMode_MultipleConsumersPass() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "dir",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *PluginE2ESuite) TestFileMode_TwoConsumersPass() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "file",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *PluginE2ESuite) TestFileMode_SingleConsumerFlagged() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "file",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("only used by")
}

func (s *PluginE2ESuite) TestFileMode_TestFilesDontCountAsConsumers() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "file",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featurea/consumer_test.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("only used by")
}

func (s *PluginE2ESuite) TestFileMode_NotImportedByAnyone() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "file",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("not imported by any entity")
}

func (s *PluginE2ESuite) TestDirMode_TwoFilesInSameDir_OneConsumer() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "dir",
	})
	s.GivenFile("featurea/consumerA.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featurea/consumerB.go",
		"package featurea",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumerA.go")
	s.ShouldFailWith("only used by: featurea")
}

func (s *PluginE2ESuite) TestFileMode_TwoFilesInSameDir_TwoConsumers() {
	s.GivenConfig(map[string]any{
		"shared-dirs": "shared",
		"mode":        "file",
	})
	s.GivenFile("featurea/consumerA.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("featurea/consumerB.go",
		"package featurea",
		"",
		"import _ \"example.com/plugin-e2e/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumerA.go")
	s.ShouldPass()
}

func (s *PluginE2ESuite) TestInvalidSettingsTypeFailsClearly() {
	s.GivenConfig(map[string]any{
		"shared-dirs": []any{"shared"},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"func Use() {}",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("nofalsesharing", "shared-dirs")
}
