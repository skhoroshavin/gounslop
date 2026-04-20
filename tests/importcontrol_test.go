package tests

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/gounslop"
	"github.com/skhoroshavin/gounslop/tests/rule"
	"github.com/stretchr/testify/suite"
)

type ImportcontrolE2ESuite struct {
	rule.Suite
}

func (s *ImportcontrolE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.ModulePath = "example.com/mod"
}

func TestImportcontrolE2E(t *testing.T) {
	suite.Run(t, new(ImportcontrolE2ESuite))
}

func (s *ImportcontrolE2ESuite) TestInvalidKeySelectorFailsClearly() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("unsupported key selector")
}

func (s *ImportcontrolE2ESuite) TestExactKeyOwnsSubtree() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestWildcardKeyOwnsDirectChildSubtree() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestWildcardDoesNotOwnParentPackage() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("undeclared importcontrol import")
}

func (s *ImportcontrolE2ESuite) TestExactSelectorOverridesWildcardOwner() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestWildcardOverridesParentExactForChildSubtree() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestUnmatchedImporterHasEmptyImportList() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("undeclared importcontrol import")
}

func (s *ImportcontrolE2ESuite) TestUnmatchedPackageIsAllowed() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{},
	})
	s.GivenFile("shared/contracts/contracts.go",
		"package contracts",
		"",
		"var X = 1",
	)
	s.LintFile("shared/contracts/contracts.go")
	s.ShouldPass()
}

func (s *ImportcontrolE2ESuite) TestImportSelectorExactMatchesOnlyExactPackage() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("undeclared importcontrol import")
}

func (s *ImportcontrolE2ESuite) TestImportSelectorChildWildcardMatchesDirectChildOnly() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestImportSelectorSelfOrChildMatchesParentAndDirectChildOnly() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("undeclared importcontrol import")
}

func (s *ImportcontrolE2ESuite) TestIntegratedDeepImportStillFailsWithinSameScope() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestImmediateChildImportRemainsAllowedWithoutPolicy() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{},
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

func (s *ImportcontrolE2ESuite) TestExternalImportIsIgnored() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"fmt\"",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldPass()
}

func (s *ImportcontrolE2ESuite) TestDifferentTopLevelImportStillUsesBoundarycontrol() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
	s.ShouldFailWith("undeclared importcontrol import")
}

func (s *ImportcontrolE2ESuite) TestNearestGoModDefinesModuleScope() {
	s.WriteRootGoMod = false
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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

func (s *ImportcontrolE2ESuite) TestNestedModuleImportIsIgnoredForParentModule() {
	s.ModulePath = "example.com/root"
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
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
