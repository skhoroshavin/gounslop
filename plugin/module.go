package plugin

import (
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"github.com/skhoroshavin/gounslop/pkg/gounslop"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("gounslop", newGounslopPlugin)
}

func newGounslopPlugin(conf any) (register.LinterPlugin, error) {
	cfg, err := register.DecodeSettings[gounslop.Config](conf)
	if err != nil {
		return nil, fmt.Errorf("gounslop: invalid settings: %w", err)
	}

	analyzers, err := gounslop.BuildAnalyzers(cfg)
	if err != nil {
		return nil, err
	}

	return &gounslopPlugin{analyzers: analyzers}, nil
}

// gounslopPlugin wraps all gounslop analyzers into a single golangci-lint LinterPlugin.
type gounslopPlugin struct {
	analyzers []*analysis.Analyzer
}

func (p *gounslopPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return p.analyzers, nil
}

func (p *gounslopPlugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
