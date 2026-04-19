package boundarycontrol

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestDiscoverModuleContextMissingGoModFailsClearly(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "boundarycontrol-module-context-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	filePath := filepath.Join(tempDir, "feature", "consumer.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("creating feature dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("package feature\n\nfunc Use() {}\n"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parsing test file: %v", err)
	}

	_, err = discoverModuleContext(&analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Pkg:   types.NewPackage("example.com/missing/feature", "feature"),
	})
	if err == nil {
		t.Fatal("expected missing go.mod error")
	}
	if got := err.Error(); got != "boundarycontrol: module scope could not be discovered from go.mod for \"example.com/missing/feature\"" {
		t.Fatalf("unexpected error: %s", got)
	}
}
