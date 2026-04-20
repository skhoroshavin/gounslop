package importcontrol

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"

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

		if isSameScopeTooDeep(importerRel, importedRel) {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
			return
		}

		if isImmediateChildImport(importerRel, importedRel) {
			return
		}

		owner, found := analyzer.ResolveOwner(cfg.Policies, importerRel)
		if found && matchesImportSelectors(owner.Imports, importedRel) {
			return
		}

		pass.Reportf(imp.Pos(), "undeclared importcontrol import: %s", importPath)
	})

	return nil, nil
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

	return segmentCount(strings.TrimPrefix(importedRel, prefix)) == 1
}

func segmentCount(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}

func matchesImportSelectors(selectors []analyzer.ParsedSelector, importedRel string) bool {
	for _, selector := range selectors {
		if matchesImportSelector(selector, importedRel) {
			return true
		}
	}

	return false
}

func matchesImportSelector(selector analyzer.ParsedSelector, importedRel string) bool {
	switch selector.Kind {
	case analyzer.SelectorKindRoot:
		return importedRel == ""
	case analyzer.SelectorKindExact:
		return importedRel == selector.Base
	case analyzer.SelectorKindChildWildcard:
		return isDirectChild(importedRel, selector.Base)
	case analyzer.SelectorKindSelfOrChild:
		return importedRel == selector.Base || isDirectChild(importedRel, selector.Base)
	default:
		return false
	}
}

func isDirectChild(importedRel, base string) bool {
	prefix := base + "/"
	if !strings.HasPrefix(importedRel, prefix) {
		return false
	}

	remainder := strings.TrimPrefix(importedRel, prefix)
	return remainder != "" && !strings.Contains(remainder, "/")
}
