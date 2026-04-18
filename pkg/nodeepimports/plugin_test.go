package nodeepimports_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

type NodeepimportsE2ESuite struct {
	ruletest.Suite
}

func TestPluginE2E(t *testing.T) {
	s := new(NodeepimportsE2ESuite)
	s.Linter = "nodeepimports"
	s.ModulePath = "example.com/mod"
	suite.Run(t, s)
}

func (s *NodeepimportsE2ESuite) TestOneLevelDeepImportPasses() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
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

func (s *NodeepimportsE2ESuite) TestDeepImportFlagged() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"import _ \"example.com/mod/feature/child/deep\"",
	)
	s.GivenFile("feature/child/child.go",
		"package child",
		"",
		"var X = 1",
	)
	s.GivenFile("feature/child/deep/deep.go",
		"package deep",
		"",
		"var Y = 2",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("too deep")
}

func (s *NodeepimportsE2ESuite) TestDifferentTopLevelScopePasses() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/featureb/other/deep\"",
	)
	s.GivenFile("featureb/other/deep/deep.go",
		"package deep",
		"",
		"var Z = 3",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *NodeepimportsE2ESuite) TestFileImportFromChildScopePasses() {
	s.GivenConfig(map[string]any{
		"module-root": "example.com/mod",
	})
	s.GivenFile("feature/child/child.go",
		"package child",
		"",
		"import _ \"example.com/mod/feature/child/deep\"",
	)
	s.GivenFile("feature/child/deep/deep.go",
		"package deep",
		"",
		"var Y = 2",
	)
	s.LintFile("feature/child/child.go")
	s.ShouldPass()
}
