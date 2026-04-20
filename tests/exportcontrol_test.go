package tests

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/gounslop"
	"github.com/skhoroshavin/gounslop/tests/rule"
	"github.com/stretchr/testify/suite"
)

type ExportcontrolE2ESuite struct {
	rule.Suite
}

func (s *ExportcontrolE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.ModulePath = "example.com/mod"
}

func TestExportcontrolE2E(t *testing.T) {
	suite.Run(t, new(ExportcontrolE2ESuite))
}

func (s *ExportcontrolE2ESuite) TestInvalidExportRegexFailsClearly() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"pkg/api": {
				Exports: []string{"("},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"func NewClient() {}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith(`architecture["pkg/api"].exports[0]: invalid regex`)
}

func (s *ExportcontrolE2ESuite) TestExportContractsAllowMatchingTopLevelDeclarations() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"pkg/api": {
				Exports: []string{"^New[A-Z].*$", "^Client$"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"type Client struct{}",
		"",
		"func NewClient() Client {",
		"\treturn Client{}",
		"}",
		"",
		"func buildClient() Client {",
		"\treturn Client{}",
		"}",
		"",
		"func (Client) Build() Client {",
		"\treturn Client{}",
		"}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldPass()
}

func (s *ExportcontrolE2ESuite) TestExportContractsReportViolatingTopLevelDeclaration() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"pkg/api": {
				Exports: []string{"^New[A-Z].*$"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"func BuildClient() {}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("pkg/api/api.go", "BuildClient does not match exportcontrol export contract")
}

func (s *ExportcontrolE2ESuite) TestExportContractsUseFullNameMatching() {
	s.GivenConfig(gounslop.Config{
		Architecture: map[string]gounslop.PolicyConfig{
			"pkg/api": {
				Exports: []string{"Error"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"type ClientError struct{}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("pkg/api/api.go", "ClientError does not match exportcontrol export contract")
}
