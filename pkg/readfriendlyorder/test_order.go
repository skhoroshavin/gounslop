package readfriendlyorder

import (
	"go/ast"
	"strings"

	"github.com/skhoroshavin/gounslop/pkg/analyzer"
	"golang.org/x/tools/go/analysis"
)

func reportTestOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	filename := pass.Fset.File(file.Pos()).Name()
	if !strings.HasSuffix(filename, "_test.go") {
		return
	}

	// Collect top-level test functions
	type testEntry struct {
		name  string
		node  *ast.FuncDecl
		index int
		kind  string // "testmain", "test", "benchmark", "helper"
	}

	var entries []testEntry
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		kind := classifyTestFunc(fn.Name.Name)
		entries = append(entries, testEntry{
			name:  fn.Name.Name,
			node:  fn,
			index: len(entries),
			kind:  kind,
		})
	}

	if len(entries) == 0 {
		return
	}

	// Check TestMain is first
	hasTests := false
	for _, e := range entries {
		if e.kind == "test" || e.kind == "benchmark" {
			hasTests = true
			break
		}
	}

	if hasTests {
		for _, e := range entries {
			if e.kind != "testmain" || e.index == 0 {
				continue
			}
			for _, other := range entries {
				if other.index < e.index && (other.kind == "test" || other.kind == "benchmark") {
					diag := analysis.Diagnostic{
						Pos:     e.node.Pos(),
						Message: "Place TestMain first in test file.",
					}
					fix := analyzer.BuildSwapFix(pass.Fset, file, src, other.node, e.node)
					if fix != nil {
						diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
					}
					pass.Report(diag)
					break
				}
			}
			break
		}
	}
}

func classifyTestFunc(name string) string {
	if name == "TestMain" {
		return "testmain"
	}
	if strings.HasPrefix(name, "Test") {
		return "test"
	}
	if strings.HasPrefix(name, "Benchmark") {
		return "benchmark"
	}
	return "helper"
}
