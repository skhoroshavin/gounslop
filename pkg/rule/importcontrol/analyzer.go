package importcontrol

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/skhoroshavin/gounslop/pkg/core/boundary"
	"github.com/skhoroshavin/gounslop/pkg/core/module"
)

func NewAnalyzer(modCache *module.Cache, cfg []boundary.ImportPolicy) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "importcontrol",
		Doc:      "enforce package import boundaries within the discovered Go module",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, modCache, cfg)
		},
	}
}

func run(pass *analysis.Pass, modCache *module.Cache, cfg []boundary.ImportPolicy) (any, error) {
	info, err := modCache.Discover(pass)
	if err != nil {
		return nil, fmt.Errorf("importcontrol: %w", err)
	}

	importerRel, ok := module.RelativePath(pass.Pkg.Path(), info.Path)
	if !ok {
		return nil, fmt.Errorf("importcontrol: package %q is outside discovered module %q", pass.Pkg.Path(), info.Path)
	}

	checkImports(pass, info, importerRel, cfg)
	return nil, nil
}

func checkImports(pass *analysis.Pass, info module.Info, importerRel string, cfg []boundary.ImportPolicy) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.ImportSpec)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return
		}

		importedRel, ownership := module.ClassifyPath(importPath, info)
		if ownership != module.CurrentModule {
			return
		}

		if isSameScopeTooDeep(importerRel, importedRel) {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
			return
		}

		if isImmediateChildImport(importerRel, importedRel) {
			return
		}

		owner, found := boundary.FindImportPolicy(cfg, importerRel)
		if found {
			for _, selector := range owner.Imports {
				if selector.MatchesImport(importedRel) {
					return
				}
			}
		}

		pass.Reportf(imp.Pos(), "undeclared importcontrol import: %s", importPath)
	})
}

func isSameScopeTooDeep(importerRel, importedRel string) bool {
	importerScope := scopeFromRelPath(importerRel)
	importedScope := scopeFromRelPath(importedRel)
	if importerScope != importedScope {
		return false
	}

	importerDepth := depthWithinScope(importerRel, importerScope)
	importedDepth := depthWithinScope(importedRel, importedScope)
	return importedDepth > importerDepth+1
}

func scopeFromRelPath(relPath string) string {
	parts := strings.SplitN(relPath, "/", 2)
	return parts[0]
}

func depthWithinScope(relPath, scope string) int {
	suffix := strings.TrimPrefix(relPath, scope+"/")
	if suffix == relPath {
		return 0
	}

	return strings.Count(suffix, "/") + 1
}

func isImmediateChildImport(importerRel, importedRel string) bool {
	if importedRel == "" {
		return false
	}

	prefix := importerRel
	if prefix != "" {
		prefix += "/"
	}
	if !strings.HasPrefix(importedRel, prefix) {
		return false
	}

	return boundary.SegmentCount(strings.TrimPrefix(importedRel, prefix)) == 1
}
