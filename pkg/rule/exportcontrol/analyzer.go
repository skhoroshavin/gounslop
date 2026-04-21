package exportcontrol

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/skhoroshavin/gounslop/pkg/core/boundary"
	"github.com/skhoroshavin/gounslop/pkg/core/module"
)

func NewAnalyzer(modCache *module.Cache, cfg []boundary.ExportPolicy) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "exportcontrol",
		Doc:  "enforce export contract patterns for top-level declarations",
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, modCache, cfg)
		},
	}
}

func run(pass *analysis.Pass, modCache *module.Cache, cfg []boundary.ExportPolicy) (any, error) {
	info, err := modCache.Discover(pass)
	if err != nil {
		return nil, fmt.Errorf("exportcontrol: %w", err)
	}

	relPath, ok := module.RelativePath(pass.Pkg.Path(), info.Path)
	if !ok {
		return nil, fmt.Errorf("exportcontrol: package %q is outside discovered module %q", pass.Pkg.Path(), info.Path)
	}

	reportExportContractDiagnostics(pass, relPath, cfg)

	return nil, nil
}

func reportExportContractDiagnostics(pass *analysis.Pass, relPath string, cfg []boundary.ExportPolicy) {
	owner, found := boundary.FindExportPolicy(cfg, relPath)
	if !found || len(owner.Exports) == 0 {
		return
	}

	for _, obj := range exportedPackageScopeObjects(pass.Pkg) {
		matched := false
		for _, pattern := range owner.Exports {
			if pattern.MatchString(obj.Name()) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		pass.Reportf(obj.Pos(), "exported declaration %s does not match exportcontrol export contract", obj.Name())
	}
}

func exportedPackageScopeObjects(pkg *types.Package) []types.Object {
	scope := pkg.Scope()
	names := scope.Names()

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
