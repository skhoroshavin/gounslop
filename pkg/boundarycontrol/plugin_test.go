package boundarycontrol_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type BoundarycontrolE2ESuite struct {
	ruletest.Suite
}

func (s *BoundarycontrolE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.EnableOnly = []string{"boundarycontrol"}
	s.ModulePath = "example.com/mod"
}

func TestPluginE2E(t *testing.T) {
	s := new(BoundarycontrolE2ESuite)
	suite.Run(t, s)
}

func (s *BoundarycontrolE2ESuite) TestInvalidKeySelectorFailsClearly() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/+": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"func Use() {}",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("boundarycontrol", "unsupported key selector")
}

func (s *BoundarycontrolE2ESuite) TestExactKeyOwnsSubtree() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/api": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/api/internal/consumer.go",
		"package internal",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/internal/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestWildcardKeyOwnsDirectChildSubtree() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/*": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/payments/internal/consumer.go",
		"package internal",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/payments/internal/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestWildcardDoesNotOwnParentPackage() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/*": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("undeclared boundarycontrol import")
}

func (s *BoundarycontrolE2ESuite) TestExactSelectorOverridesWildcardOwner() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/*": {
				Imports: []string{"shared/general"},
			},
			"feature/api": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/api/internal/consumer.go",
		"package internal",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/internal/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestWildcardOverridesParentExactForChildSubtree() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature": {
				Imports: []string{"shared/root"},
			},
			"feature/*": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/payments/consumer.go",
		"package payments",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/payments/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestUnmatchedImporterHasEmptyImportList() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"shared": {
				Imports: []string{"."},
			},
		},
	})
	s.GivenFile("unknown/feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("unknown/feature/consumer.go")
	s.ShouldFailWith("undeclared boundarycontrol import")
}

func (s *BoundarycontrolE2ESuite) TestUnmatchedPackageIsAllowed() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{},
	})
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("shared/contracts/contracts.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestImportSelectorExactMatchesOnlyExactPackage() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/api": {
				Imports: []string{"shared/contracts"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import _ \"example.com/mod/shared/contracts/http\"",
	)
	s.GivenFile("shared/contracts/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("undeclared boundarycontrol import")
}

func (s *BoundarycontrolE2ESuite) TestImportSelectorChildWildcardMatchesDirectChildOnly() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/api": {
				Imports: []string{"shared/*"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import _ \"example.com/mod/shared/contracts\"",
	)
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestImportSelectorSelfOrChildMatchesParentAndDirectChildOnly() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature/api": {
				Imports: []string{"shared/+"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import _ \"example.com/mod/shared/contracts/http\"",
	)
	s.GivenFile("shared/contracts/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("undeclared boundarycontrol import")
}

func (s *BoundarycontrolE2ESuite) TestIntegratedDeepImportStillFailsWithinSameScope() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature": {
				Imports: []string{"feature/child/deep"},
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/mod/feature/child/deep\"",
	)
	s.GivenFile("feature/child/deep/deep.go",
		"package deep",
		"",
		"var X = 1",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("too deep")
}

func (s *BoundarycontrolE2ESuite) TestImmediateChildImportRemainsAllowedWithoutPolicy() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/mod/feature/child\"",
	)
	s.GivenFile("feature/child/child.go",
		"package child",
		"",
		"var X = 1",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestExternalImportIsIgnored() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"fmt\"",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestDifferentTopLevelImportStillUsesBoundarycontrol() {
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"featurea": {
				Imports: []string{"shared/+"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/featureb/other/deep\"",
	)
	s.GivenFile("featureb/other/deep/deep.go",
		"package deep",
		"",
		"var X = 1",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("undeclared boundarycontrol import")
}

func (s *BoundarycontrolE2ESuite) TestNearestGoModDefinesModuleScope() {
	s.WriteRootGoMod = false
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"internal/*": {
				Imports: []string{"pkg/contracts"},
			},
		},
	})
	s.GivenFile("tools/go.mod",
		"module example.com/root/tools",
		"",
		"go 1.25.6",
	)
	s.GivenFile("tools/internal/checker/checker.go",
		"package checker",
		"",
		"import _ \"example.com/root/tools/pkg/contracts\"",
	)
	s.GivenFile("tools/pkg/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("tools/internal/checker/checker.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestNestedModuleImportIsIgnoredForParentModule() {
	s.ModulePath = "example.com/root"
	s.GivenConfig(ruletest.GounslopSettings{
		Architecture: map[string]ruletest.PolicySettings{
			"feature": {
				Imports: []string{},
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/root/tools/pkg\"",
	)
	s.GivenFile("tools/go.mod",
		"module example.com/root/tools",
		"",
		"go 1.25.6",
	)
	s.GivenFile("tools/pkg/pkg.go",
		"package pkg",
		"",
		"var X = 1",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldPass()
}
