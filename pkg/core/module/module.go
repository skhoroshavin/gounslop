package module

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
)

type Info struct {
	Dir               string
	Path              string
	NestedModulePaths []string
}

type Cache struct {
	cache sync.Map
}

func (c *Cache) Discover(pass *analysis.Pass) (Info, error) {
	if len(pass.Files) == 0 {
		return Info{}, fmt.Errorf("no files available to discover module scope")
	}

	packageFile := pass.Fset.PositionFor(pass.Files[0].Package, false).Filename
	if packageFile == "" {
		return Info{}, fmt.Errorf("no file path available to discover module scope for %q", pass.Pkg.Path())
	}

	goModPath, err := nearestGoMod(filepath.Dir(packageFile))
	if err != nil {
		return Info{}, err
	}
	if goModPath == "" {
		return Info{}, fmt.Errorf("module scope could not be discovered from go.mod for %q", pass.Pkg.Path())
	}

	moduleDir := filepath.Dir(goModPath)
	return c.load(moduleDir, goModPath)
}

func (c *Cache) load(moduleDir, goModPath string) (Info, error) {
	if cached, ok := c.cache.Load(moduleDir); ok {
		return cached.(Info), nil
	}

	modulePath, err := readModulePath(goModPath)
	if err != nil {
		return Info{}, err
	}

	nestedModulePaths, err := discoverNestedModulePaths(moduleDir)
	if err != nil {
		return Info{}, err
	}

	ctx := Info{
		Dir:               moduleDir,
		Path:              modulePath,
		NestedModulePaths: nestedModulePaths,
	}

	actual, _ := c.cache.LoadOrStore(moduleDir, ctx)
	return actual.(Info), nil
}

func nearestGoMod(startDir string) (string, error) {
	currentDir := startDir
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		_, err := os.Stat(goModPath)
		if err == nil {
			return goModPath, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("checking %s: %w", goModPath, err)
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", nil
		}

		currentDir = parentDir
	}
}

func discoverNestedModulePaths(moduleDir string) ([]string, error) {
	modulePaths := make(map[string]struct{})
	rootGoModPath := filepath.Join(moduleDir, "go.mod")

	collector := &modulePathCollector{rootGoModPath: rootGoModPath, modulePaths: modulePaths}
	err := filepath.WalkDir(moduleDir, collector.collect)
	if err != nil {
		return nil, fmt.Errorf("scanning nested modules under %s: %w", moduleDir, err)
	}

	result := make([]string, 0, len(modulePaths))
	for p := range modulePaths {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b string) int {
		if len(a) != len(b) {
			return cmp.Compare(len(b), len(a))
		}
		return cmp.Compare(a, b)
	})
	return result, nil
}

type modulePathCollector struct {
	rootGoModPath string
	modulePaths   map[string]struct{}
}

func (c *modulePathCollector) collect(path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if d.IsDir() || d.Name() != "go.mod" || path == c.rootGoModPath {
		return nil
	}
	modulePath, err := readModulePath(path)
	if err != nil {
		return err
	}
	c.modulePaths[modulePath] = struct{}{}
	return nil
}

func RelativePath(pkgPath, modulePath string) (string, bool) {
	if pkgPath == modulePath {
		return "", true
	}

	prefix := modulePath + "/"
	if !strings.HasPrefix(pkgPath, prefix) {
		return "", false
	}

	return strings.TrimPrefix(pkgPath, prefix), true
}

func ClassifyPath(importPath string, info Info) (string, Ownership) {
	importedRel, ok := RelativePath(importPath, info.Path)
	if !ok {
		return "", OutsideModule
	}

	for _, nestedModulePath := range info.NestedModulePaths {
		if importPath == nestedModulePath || strings.HasPrefix(importPath, nestedModulePath+"/") {
			return "", NestedModule
		}
	}

	return importedRel, CurrentModule
}

func readModulePath(goModPath string) (string, error) {
	contents, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", goModPath, err)
	}

	modulePath := modfile.ModulePath(contents)
	if modulePath == "" {
		return "", fmt.Errorf("%s does not declare a module path", goModPath)
	}

	return modulePath, nil
}

type Ownership int

const (
	OutsideModule Ownership = iota
	CurrentModule
	NestedModule
)
