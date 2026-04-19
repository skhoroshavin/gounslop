package plugin

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/golangci/plugin-module-register/register"
	"github.com/skhoroshavin/gounslop/pkg/boundarycontrol"
	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
)

// AnalyzerNames is the list of all analyzer names registered by the gounslop plugin.
var AnalyzerNames = []string{
	"boundarycontrol",
	"nospecialunicode",
	"nounicodeescape",
	"readfriendlyorder",
}

func init() {
	register.Plugin("gounslop", newGounslopPlugin)
}

func newGounslopPlugin(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[gounslopSettings](conf)
	if err != nil {
		return nil, fmt.Errorf("gounslop: invalid settings: %w", err)
	}

	for _, name := range s.Disable {
		if !slices.Contains(AnalyzerNames, name) {
			return nil, fmt.Errorf("gounslop: unknown analyzer %q in disable list", name)
		}
	}

	if !slices.Contains(s.Disable, "boundarycontrol") && s.Architecture != nil {
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

		if err := boundarycontrol.Analyzer.Flags.Set("architecture", string(architectureJSON)); err != nil {
			return nil, err
		}
	}

	var analyzers []*analysis.Analyzer
	for _, name := range AnalyzerNames {
		if slices.Contains(s.Disable, name) {
			continue
		}
		analyzers = append(analyzers, allAnalyzers[name])
	}

	return &gounslopPlugin{analyzers: analyzers}, nil
}

var allAnalyzers = map[string]*analysis.Analyzer{
	"boundarycontrol":   boundarycontrol.Analyzer,
	"nospecialunicode":  nospecialunicode.Analyzer,
	"nounicodeescape":   nounicodeescape.Analyzer,
	"readfriendlyorder": readfriendlyorder.Analyzer,
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

type gounslopSettings struct {
	Disable      []string                                 `json:"disable"`
	Architecture map[string]boundarycontrolPolicySettings `json:"architecture"`
}

type boundarycontrolPolicySettings struct {
	Imports []string `json:"imports"`
	Exports []string `json:"exports"`
	Shared  bool     `json:"shared"`
	Mode    *string  `json:"mode"`
}

func (s gounslopSettings) toConfig() (boundarycontrol.Config, error) {
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
			Exports: append([]string(nil), policy.Exports...),
			Shared:  policy.Shared,
		}
	}

	return cfg, nil
}
