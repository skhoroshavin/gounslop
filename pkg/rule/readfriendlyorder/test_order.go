package readfriendlyorder

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func reportTestOrdering(pass *analysis.Pass, file *ast.File, src []byte) {
	if !isTestFile(pass, file) {
		return
	}

	entries := collectTestEntries(file)
	if len(entries) == 0 {
		return
	}

	testMain, blocker := findTestMainOrderingViolation(entries)
	if testMain == nil {
		return
	}

	fixCtx := newFixContext(pass.Fset, file, src)
	diag := analysis.Diagnostic{
		Pos:     testMain.node.Pos(),
		Message: "Place TestMain first in test file.",
	}
	fix := fixCtx.buildSwapFix(blocker.node, testMain.node)
	if fix != nil {
		diag.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	pass.Report(diag)
}

func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	filename := pass.Fset.File(file.Pos()).Name()
	return strings.HasSuffix(filename, "_test.go")
}

func collectTestEntries(file *ast.File) []testEntry {
	var entries []testEntry
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		entries = append(entries, testEntry{
			node: fn,
			kind: classifyTestFunc(fn.Name.Name),
		})
	}
	return entries
}

func findTestMainOrderingViolation(entries []testEntry) (*testEntry, *testEntry) {
	var mainIdx int
	foundMain := false
	for i, e := range entries {
		if e.kind == testKindMain {
			mainIdx = i
			foundMain = true
			break
		}
	}
	if !foundMain || mainIdx == 0 {
		return nil, nil
	}

	for i := 0; i < mainIdx; i++ {
		if entries[i].kind == testKindTest || entries[i].kind == testKindBenchmark {
			return &entries[mainIdx], &entries[i]
		}
	}
	return nil, nil
}

func classifyTestFunc(name string) testKind {
	if name == "TestMain" {
		return testKindMain
	}
	if strings.HasPrefix(name, "Test") {
		return testKindTest
	}
	if strings.HasPrefix(name, "Benchmark") {
		return testKindBenchmark
	}
	return testKindHelper
}

type testEntry struct {
	node *ast.FuncDecl
	kind testKind
}

type testKind string

const (
	testKindMain      testKind = "testmain"
	testKindTest      testKind = "test"
	testKindBenchmark testKind = "benchmark"
	testKindHelper    testKind = "helper"
)
