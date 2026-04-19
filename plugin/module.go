package plugin

import (
	"encoding/json"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"github.com/skhoroshavin/gounslop/pkg/boundarycontrol"
	"github.com/skhoroshavin/gounslop/pkg/nodeepimports"
	"github.com/skhoroshavin/gounslop/pkg/nofalsesharing"
	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("boundarycontrol", newBoundarycontrolPlugin)
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

func newBoundarycontrolPlugin(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[boundarycontrolSettings](conf)
	if err != nil {
		return nil, err
	}

	cfg := boundarycontrol.Config{
		ModuleRoot: strings.TrimRight(strings.TrimSpace(s.ModuleRoot), "/"),
		Selectors:  make([]boundarycontrol.SelectorPolicy, 0, len(s.Selectors)),
	}
	for _, selector := range s.Selectors {
		cfg.Selectors = append(cfg.Selectors, boundarycontrol.SelectorPolicy{
			Selector: selector.Selector,
			Imports:  append([]string(nil), selector.Imports...),
		})
	}

	if err := boundarycontrol.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	selectorsJSON, err := json.Marshal(cfg.Selectors)
	if err != nil {
		return nil, err
	}

	a := boundarycontrol.Analyzer
	for name, val := range map[string]string{
		"module-root": cfg.ModuleRoot,
		"selectors":   string(selectorsJSON),
	} {
		if err := a.Flags.Set(name, val); err != nil {
			return nil, err
		}
	}

	return &simplePlugin{analyzer: a, loadMode: register.LoadModeTypesInfo}, nil
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

type boundarycontrolSettings struct {
	ModuleRoot string                            `json:"module-root"`
	Selectors  []boundarycontrolSelectorSettings `json:"selectors"`
}

type boundarycontrolSelectorSettings struct {
	Selector string   `json:"selector"`
	Imports  []string `json:"imports"`
}

type nofalsesharingSettings struct {
	SharedDirs string `json:"shared-dirs"`
	Mode       string `json:"mode"`
	ModuleRoot string `json:"module-root"`
}
