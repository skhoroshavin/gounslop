package plugin

import (
	"github.com/golangci/plugin-module-register/register"
	"github.com/skhoroshavin/gounslop/pkg/nodeepimports"
	"github.com/skhoroshavin/gounslop/pkg/nofalsesharing"
	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("nospecialunicode", newSyntaxPlugin(nospecialunicode.Analyzer))
	register.Plugin("nounicodeescape", newSyntaxPlugin(nounicodeescape.Analyzer))
	register.Plugin("nodeepimports", newNodeeepimportsPlugin)
	register.Plugin("nofalsesharing", newNofalsesharingPlugin)
	register.Plugin("readfriendlyorder", newTypesPlugin(readfriendlyorder.Analyzer))
}

func newSyntaxPlugin(a *analysis.Analyzer) register.NewPlugin {
	return func(_ any) (register.LinterPlugin, error) {
		return &simplePlugin{analyzer: a, loadMode: register.LoadModeSyntax}, nil
	}
}

func newTypesPlugin(a *analysis.Analyzer) register.NewPlugin {
	return func(_ any) (register.LinterPlugin, error) {
		return &simplePlugin{analyzer: a, loadMode: register.LoadModeTypesInfo}, nil
	}
}

func newNodeeepimportsPlugin(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[nodeepimportsSettings](conf)
	if err != nil {
		return nil, err
	}
	if s.ModuleRoot != "" {
		if err := nodeepimports.Analyzer.Flags.Set("module-root", s.ModuleRoot); err != nil {
			return nil, err
		}
	}
	return &simplePlugin{analyzer: nodeepimports.Analyzer, loadMode: register.LoadModeTypesInfo}, nil
}

func newNofalsesharingPlugin(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[nofalsesharingSettings](conf)
	if err != nil {
		return nil, err
	}
	a := nofalsesharing.Analyzer
	for name, val := range map[string]string{
		"shared-dirs": s.SharedDirs,
		"mode":        s.Mode,
		"module-root": s.ModuleRoot,
	} {
		if val != "" {
			if err := a.Flags.Set(name, val); err != nil {
				return nil, err
			}
		}
	}
	return &simplePlugin{analyzer: a, loadMode: register.LoadModeTypesInfo}, nil
}

// simplePlugin wraps an analyzer into a golangci-lint LinterPlugin.
type simplePlugin struct {
	analyzer *analysis.Analyzer
	loadMode string
}

func (p *simplePlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{p.analyzer}, nil
}

func (p *simplePlugin) GetLoadMode() string {
	return p.loadMode
}

type nodeepimportsSettings struct {
	ModuleRoot string `json:"module-root"`
}

type nofalsesharingSettings struct {
	SharedDirs string `json:"shared-dirs"`
	Mode       string `json:"mode"`
	ModuleRoot string `json:"module-root"`
}
