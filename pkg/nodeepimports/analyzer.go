package nodeepimports

import (
	"go/ast"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "nodeepimports",
	Doc:      "forbid deep imports inside the same top-level folder",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.StringVar(&moduleRoot, "module-root", "", "Go module path prefix (e.g. github.com/org/repo)")
}

func run(pass *analysis.Pass) (any, error) {
	root := moduleRoot
	if root == "" {
		return nil, nil
	}

	pkgPath := pass.Pkg.Path()
	if !strings.HasPrefix(pkgPath, root+"/") && pkgPath != root {
		return nil, nil
	}

	importerRel := strings.TrimPrefix(pkgPath, root+"/")
	if importerRel == pkgPath {
		importerRel = ""
	}

	importerScope := scopeFromRelPath(importerRel)
	importerDepth := depthWithinScope(importerRel, importerScope)

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return
		}

		if !strings.HasPrefix(importPath, root+"/") {
			return
		}

		targetRel := strings.TrimPrefix(importPath, root+"/")
		targetScope := scopeFromRelPath(targetRel)

		if targetScope != importerScope {
			return
		}

		targetDepth := depthWithinScope(targetRel, targetScope)
		if targetDepth > importerDepth+1 {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
		}
	})

	return nil, nil
}

var moduleRoot string

func scopeFromRelPath(relPath string) string {
	parts := strings.SplitN(relPath, "/", 2)
	return parts[0]
}

func depthWithinScope(relPath, scope string) int {
	if relPath == scope {
		return 0
	}
	suffix := strings.TrimPrefix(relPath, scope+"/")
	if suffix == relPath {
		return 0
	}
	// Count path segments: "child/deep" = 2 segments = depth 2
	return strings.Count(suffix, "/") + 1
}
