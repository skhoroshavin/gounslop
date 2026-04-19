package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"github.com/skhoroshavin/gounslop/pkg/boundarycontrol"
	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("boundarycontrol", newBoundarycontrolPlugin)
	register.Plugin("nospecialunicode", newSyntaxPlugin(nospecialunicode.Analyzer))
	register.Plugin("nounicodeescape", newSyntaxPlugin(nounicodeescape.Analyzer))
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

func newBoundarycontrolPlugin(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[boundarycontrolSettings](conf)
	if err != nil {
		return nil, fmt.Errorf("boundarycontrol: invalid architecture settings: %w", err)
	}

	cfg, err := s.toConfig()
	if err != nil {
		return nil, err
	}

	if err := boundarycontrol.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	architectureJSON, err := json.Marshal(cfg.Architecture)
	if err != nil {
		return nil, err
	}

	a := boundarycontrol.Analyzer
	if err := a.Flags.Set("architecture", string(architectureJSON)); err != nil {
		return nil, err
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

type boundarycontrolSettings struct {
	Architecture map[string]boundarycontrolPolicySettings `json:"architecture"`
}

type boundarycontrolPolicySettings struct {
	Imports []string `json:"imports"`
	Shared  bool     `json:"shared"`
	Mode    *string  `json:"mode"`
}

func (s boundarycontrolSettings) toConfig() (boundarycontrol.Config, error) {
	cfg := boundarycontrol.Config{
		Architecture: make(map[string]boundarycontrol.Policy, len(s.Architecture)),
	}

	for selector, policy := range s.Architecture {
		if policy.Mode != nil {
			return boundarycontrol.Config{}, fmt.Errorf(
				"boundarycontrol: architecture[%q].mode is unsupported; migrated false-sharing counts consumers by importing package path only",
				selector,
			)
		}

		cfg.Architecture[selector] = boundarycontrol.Policy{
			Imports: append([]string(nil), policy.Imports...),
			Shared:  policy.Shared,
		}
	}

	return cfg, nil
}
