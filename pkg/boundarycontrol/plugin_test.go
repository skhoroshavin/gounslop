package boundarycontrol_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type BoundarycontrolE2ESuite struct {
	ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
	s := new(BoundarycontrolE2ESuite)
	s.Linter = "boundarycontrol"
	s.ModulePath = "example.com/mod"
	suite.Run(t, s)
}

func (s *BoundarycontrolE2ESuite) TestMissingModuleRootFailsClearly() {
	s.GivenConfig(map[string]any{
		"selectors": []map[string]any{{
			"selector": "feature",
			"imports":  []string{"shared/contracts"},
		}},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"func Use() {}",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("boundarycontrol", "module-root is required")
}

func (s *BoundarycontrolE2ESuite) TestInvalidKeySelectorFailsClearly() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/+",
			"imports":  []string{"shared/contracts"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/api",
			"imports":  []string{"shared/contracts"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/*",
			"imports":  []string{"shared/contracts"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/*",
			"imports":  []string{"shared/contracts"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{
			{
				"selector": "feature/*",
				"imports":  []string{"shared/general"},
			},
			{
				"selector": "feature/api",
				"imports":  []string{"shared/contracts"},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{
			{
				"selector": "feature",
				"imports":  []string{"shared/root"},
			},
			{
				"selector": "feature/*",
				"imports":  []string{"shared/contracts"},
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

func (s *BoundarycontrolE2ESuite) TestDeclarationOrderBreaksTie() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{
			{
				"selector": "feature/*",
				"imports":  []string{"shared/first"},
			},
			{
				"selector": "feature/*",
				"imports":  []string{"shared/second"},
			},
		},
	})
	s.GivenFile("feature/payments/consumer.go",
		"package payments",
		"",
		"import _ \"example.com/mod/shared/first\"",
	)
	s.GivenFile("shared/first/first.go",
		"package first",
		"",
		"var X = 1",
	)
	s.LintFile("feature/payments/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestUnmatchedImporterHasEmptyImportList() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "shared",
			"imports":  []string{"."},
		}},
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

func (s *BoundarycontrolE2ESuite) TestImportSelectorExactMatchesOnlyExactPackage() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/api",
			"imports":  []string{"shared/contracts"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/api",
			"imports":  []string{"shared/*"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature/api",
			"imports":  []string{"shared/+"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{{
			"selector": "feature",
			"imports":  []string{"feature/child/deep"},
		}},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors":   []map[string]any{},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors":   []map[string]any{},
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
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
		"selectors": []map[string]any{
			{
				"selector": "featurea",
				"imports":  []string{"shared/+"},
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
