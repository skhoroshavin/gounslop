package gounslop

import (
	"slices"

	"github.com/skhoroshavin/gounslop/pkg/core/module"
	"github.com/skhoroshavin/gounslop/pkg/rule/exportcontrol"
	"github.com/skhoroshavin/gounslop/pkg/rule/importcontrol"
	"github.com/skhoroshavin/gounslop/pkg/rule/nofalsesharing"
	"github.com/skhoroshavin/gounslop/pkg/rule/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/rule/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/rule/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
)

// BuildAnalyzers creates all gounslop analyzers with the given configuration.
// Caches are injected from the root package rather than using package-level singletons.
func BuildAnalyzers(cfg Config) ([]*analysis.Analyzer, error) {
	compiledCfg, err := compileConfig(cfg)
	if err != nil {
		return nil, err
	}

	modCache := &module.Cache{}

	all := []*analysis.Analyzer{
		importcontrol.NewAnalyzer(modCache, compiledCfg.Import),
		exportcontrol.NewAnalyzer(modCache, compiledCfg.Export),
		nofalsesharing.NewAnalyzer(modCache, compiledCfg.Shared),
		readfriendlyorder.NewAnalyzer(),
		nospecialunicode.NewAnalyzer(),
		nounicodeescape.NewAnalyzer(),
	}

	if len(cfg.Disable) == 0 {
		return all, nil
	}

	filtered := make([]*analysis.Analyzer, 0, len(all))
	for _, a := range all {
		if slices.Contains(cfg.Disable, a.Name) {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered, nil
}
