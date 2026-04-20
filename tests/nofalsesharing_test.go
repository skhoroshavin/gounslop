package tests

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/gounslop"
	"github.com/skhoroshavin/gounslop/tests/rule"
	"github.com/stretchr/testify/suite"
)

type NofalsesharingE2ESuite struct {
	rule.Suite
}

func (s *NofalsesharingE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.ModulePath = "example.com/mod"
}

func TestNofalsesharingE2E(t *testing.T) {
	suite.Run(t, new(NofalsesharingE2ESuite))
}

func (s *NofalsesharingE2ESuite) TestRemovedModeSettingFailsClearly() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
				Mode:   gounslop.StrPtr("dir"),
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"func Use() {}",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("architecture[\"shared\"].mode is unsupported")
}

func (s *NofalsesharingE2ESuite) TestExactSharedSelectorMarksSubtreeAsShared() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared/lib": {
				Shared: true,
			},
			"feature/api": {
				Imports: []string{"shared/lib/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/lib/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/lib/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/lib/http/http.go", "X only used by: feature/api", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestWildcardSharedSelectorMarksDirectChildSubtreeAsShared() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared/*": {
				Shared: true,
			},
			"feature/api": {
				Imports: []string{"shared/contracts/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/contracts/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/contracts/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/contracts/http/http.go", "X only used by: feature/api", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestSelectorWithoutSharedFlagIsNotSharedDeclaration() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared/lib": {},
			"feature/api": {
				Imports: []string{"shared/lib/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/lib/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/lib/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldPass()
}

func (s *NofalsesharingE2ESuite) TestSharedPackageWithSingleConsumerFails() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
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
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestSharedPackageWithMultipleConsumersPasses() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
			"featureb": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *NofalsesharingE2ESuite) TestSharedPackageTestFilesDoNotIncreaseConsumerCount() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featurea/consumer_test.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestSharedPackageWithNoConsumersFails() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{},
			},
		},
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
	s.ShouldFailWith("shared/util.go", "Value not used by any entity", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestTwoFilesInSamePackageCountAsOneConsumer() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer_a.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featurea/consumer_b.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer_a.go")
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *NofalsesharingE2ESuite) TestDifferentSymbolsUsedByDifferentConsumersFailSeparately() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
			"featureb": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.A()",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.B()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func A() string { return \"a\" }",
		"func B() string { return \"b\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith(
		"shared/util.go",
		"A only used by: featurea",
		"B only used by: featureb",
		"Must be used by 2+ entities",
	)
}

func (s *NofalsesharingE2ESuite) TestInternalSharedPackageReferenceCountsAsConsumer() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
		"func useValue() string { return Value() }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *NofalsesharingE2ESuite) TestSharedPackageWithNoExportedSymbolsProducesNoDiagnostics() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *NofalsesharingE2ESuite) TestExportedSymbolFormsUsingTypeInfo() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"shared": {
				Shared: true,
			},
			"featurea": {
				Imports: []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.FuncValue()",
		"var _ shared.Widget",
		"var _ = shared.SharedValue",
		"const _ = shared.SharedConst",
		"var _ = shared.Worker{}.Do()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func FuncValue() string { return \"x\" }",
		"type Widget struct{}",
		"var SharedValue = 1",
		"const SharedConst = \"x\"",
		"type Worker struct{}",
		"func (Worker) Do() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith(
		"shared/util.go",
		"FuncValue only used by: featurea",
		"Widget only used by: featurea",
		"SharedValue only used by: featurea",
		"SharedConst only used by: featurea",
		"Worker.Do only used by: featurea",
		"Must be used by 2+ entities",
	)
}
