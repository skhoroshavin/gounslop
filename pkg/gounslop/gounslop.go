package gounslop

import (
	"fmt"
	"slices"

	"github.com/skhoroshavin/gounslop/pkg/analyzer"
	"github.com/skhoroshavin/gounslop/pkg/exportcontrol"
	"github.com/skhoroshavin/gounslop/pkg/importcontrol"
	"github.com/skhoroshavin/gounslop/pkg/nofalsesharing"
	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// BuildAnalyzers creates all gounslop analyzers with the given configuration.
// Caches are injected from the root package rather than using package-level singletons.
func BuildAnalyzers(cfg Config) ([]*analysis.Analyzer, error) {
	if err := validateDisable(cfg.Disable); err != nil {
		return nil, err
	}

	compiledCfg, err := compileConfig(cfg)
	if err != nil {
		return nil, err
	}

	modCache := newModuleContextCache()
	fsCache := nofalsesharing.NewCache()

	all := []*analysis.Analyzer{
		{
			Name:     "importcontrol",
			Doc:      importcontrolDoc,
			Requires: []*analysis.Analyzer{inspect.Analyzer},
			Run: func(pass *analysis.Pass) (any, error) {
				return importcontrol.Run(pass, modCache, compiledCfg)
			},
		},
		{
			Name:     "exportcontrol",
			Doc:      exportcontrolDoc,
			Requires: []*analysis.Analyzer{inspect.Analyzer},
			Run: func(pass *analysis.Pass) (any, error) {
				return exportcontrol.Run(pass, modCache, compiledCfg)
			},
		},
		{
			Name:     "nofalsesharing",
			Doc:      nofalsesharingDoc,
			Requires: []*analysis.Analyzer{inspect.Analyzer},
			Run: func(pass *analysis.Pass) (any, error) {
				return nofalsesharing.Run(pass, modCache, fsCache, compiledCfg)
			},
		},
		readfriendlyorder.Analyzer,
		nospecialunicode.Analyzer,
		nounicodeescape.Analyzer,
	}

	return filterByDisable(all, cfg.Disable), nil
}

func validateDisable(disable []string) error {
	for _, name := range disable {
		if !slices.Contains(analyzerNames, name) {
			return fmt.Errorf("gounslop: unknown analyzer %q in disable list", name)
		}
	}
	return nil
}

var analyzerNames = []string{
	"importcontrol",
	"exportcontrol",
	"nofalsesharing",
	"readfriendlyorder",
	"nospecialunicode",
	"nounicodeescape",
}

func filterByDisable(analyzers []*analysis.Analyzer, disable []string) []*analysis.Analyzer {
	if len(disable) == 0 {
		return analyzers
	}

	filtered := make([]*analysis.Analyzer, 0, len(analyzers))
	for _, a := range analyzers {
		if slices.Contains(disable, a.Name) {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}

func newModuleContextCache() *analyzer.ModuleContextCache {
	return &analyzer.ModuleContextCache{}
}

const (
	importcontrolDoc  = "enforce package import boundaries within the discovered Go module"
	exportcontrolDoc  = "enforce export contract patterns for top-level declarations"
	nofalsesharingDoc = "detect exported symbols in shared packages that are not used by 2+ entities"
)
