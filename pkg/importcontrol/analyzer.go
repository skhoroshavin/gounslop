package importcontrol

import (
	"fmt"
	"go/ast"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/skhoroshavin/gounslop/pkg/analyzer"
)

// Run performs import boundary checking for a single package.
func Run(pass *analysis.Pass, modCache *analyzer.ModuleContextCache, cfg analyzer.CompiledConfig) (any, error) {
	moduleCtx, err := modCache.Discover(pass)
	if err != nil {
		return nil, fmt.Errorf("importcontrol: %w", err)
	}

	importerRel, ok := analyzer.RelativeModulePath(pass.Pkg.Path(), moduleCtx.Path)
	if !ok {
		return nil, fmt.Errorf("importcontrol: package %q is outside discovered module %q", pass.Pkg.Path(), moduleCtx.Path)
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.ImportSpec)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return
		}

		importedRel, ownership := analyzer.ClassifyImportPath(importPath, moduleCtx)
		if ownership != analyzer.ImportOwnershipCurrentModule {
			return
		}

		if analyzer.IsSameScopeTooDeep(importerRel, importedRel) {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
			return
		}

		if analyzer.IsImmediateChildImport(importerRel, importedRel) {
			return
		}

		owner, found := analyzer.ResolveOwner(cfg.Policies, importerRel)
		if found && analyzer.MatchesImportSelectors(owner.Imports, importedRel) {
			return
		}

		pass.Reportf(imp.Pos(), "undeclared importcontrol import: %s", importPath)
	})

	return nil, nil
}
