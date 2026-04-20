package exportcontrol

import (
	"fmt"
	"go/types"
	"regexp"
	"sort"

	"golang.org/x/tools/go/analysis"

	"github.com/skhoroshavin/gounslop/pkg/analyzer"
)

// Run performs export contract checking for a single package.
func Run(pass *analysis.Pass, modCache *analyzer.ModuleContextCache, cfg analyzer.CompiledConfig) (any, error) {
	moduleCtx, err := modCache.Discover(pass)
	if err != nil {
		return nil, fmt.Errorf("exportcontrol: %w", err)
	}

	importerRel, ok := analyzer.RelativeModulePath(pass.Pkg.Path(), moduleCtx.Path)
	if !ok {
		return nil, fmt.Errorf("exportcontrol: package %q is outside discovered module %q", pass.Pkg.Path(), moduleCtx.Path)
	}

	reportExportContractDiagnostics(pass, importerRel, cfg)

	return nil, nil
}

func reportExportContractDiagnostics(pass *analysis.Pass, importerRel string, cfg analyzer.CompiledConfig) {
	owner, found := analyzer.ResolveOwner(cfg.Policies, importerRel)
	if !found || len(owner.Exports) == 0 {
		return
	}

	for _, obj := range exportedPackageScopeObjects(pass.Pkg) {
		if matchesExportContract(owner.Exports, obj.Name()) {
			continue
		}

		pass.Reportf(obj.Pos(), "exported declaration %s does not match exportcontrol export contract", obj.Name())
	}
}

func exportedPackageScopeObjects(pkg *types.Package) []types.Object {
	scope := pkg.Scope()
	names := scope.Names()
	sort.Strings(names)

	objects := make([]types.Object, 0, len(names))
	for _, name := range names {
		obj := scope.Lookup(name)
		if obj == nil || !obj.Exported() {
			continue
		}

		switch obj.(type) {
		case *types.Func, *types.TypeName, *types.Var, *types.Const:
			objects = append(objects, obj)
		}
	}

	return objects
}

func matchesExportContract(patterns []*regexp.Regexp, name string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}

	return false
}
