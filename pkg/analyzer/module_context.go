package analyzer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
)

type ModuleContext struct {
	Dir               string
	Path              string
	NestedModulePaths []string
}

type ModuleContextCache struct {
	cache sync.Map
}

func NewModuleContextCache() *ModuleContextCache {
	return &ModuleContextCache{}
}

func (c *ModuleContextCache) Discover(pass *analysis.Pass) (ModuleContext, error) {
	if len(pass.Files) == 0 {
		return ModuleContext{}, fmt.Errorf("no files available to discover module scope")
	}

	packageFile := pass.Fset.PositionFor(pass.Files[0].Package, false).Filename
	if packageFile == "" {
		return ModuleContext{}, fmt.Errorf("no file path available to discover module scope for %q", pass.Pkg.Path())
	}

	goModPath, err := nearestGoMod(filepath.Dir(packageFile))
	if err != nil {
		return ModuleContext{}, err
	}
	if goModPath == "" {
		return ModuleContext{}, fmt.Errorf("module scope could not be discovered from go.mod for %q", pass.Pkg.Path())
	}

	moduleDir := filepath.Dir(goModPath)
	return c.load(moduleDir, goModPath)
}

func (c *ModuleContextCache) load(moduleDir, goModPath string) (ModuleContext, error) {
	if cached, ok := c.cache.Load(moduleDir); ok {
		return cached.(ModuleContext), nil
	}

	modulePath, err := readModulePath(goModPath)
	if err != nil {
		return ModuleContext{}, err
	}

	nestedModulePaths, err := discoverNestedModulePaths(moduleDir)
	if err != nil {
		return ModuleContext{}, err
	}

	ctx := ModuleContext{
		Dir:               moduleDir,
		Path:              modulePath,
		NestedModulePaths: nestedModulePaths,
	}

	actual, _ := c.cache.LoadOrStore(moduleDir, ctx)
	return actual.(ModuleContext), nil
}

func nearestGoMod(startDir string) (string, error) {
	currentDir := startDir
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		_, err := os.Stat(goModPath)
		if err == nil {
			return goModPath, nil
		}
		if err != nil && !os.IsNotExist(err) {
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

	err := filepath.WalkDir(moduleDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}
		if d.Name() != "go.mod" || path == rootGoModPath {
			return nil
		}

		modulePath, err := readModulePath(path)
		if err != nil {
			return err
		}

		modulePaths[modulePath] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning nested modules under %s: %w", moduleDir, err)
	}

	nestedModulePaths := make([]string, 0, len(modulePaths))
	for modulePath := range modulePaths {
		nestedModulePaths = append(nestedModulePaths, modulePath)
	}

	sort.Slice(nestedModulePaths, func(i, j int) bool {
		if len(nestedModulePaths[i]) != len(nestedModulePaths[j]) {
			return len(nestedModulePaths[i]) > len(nestedModulePaths[j])
		}

		return nestedModulePaths[i] < nestedModulePaths[j]
	})

	return nestedModulePaths, nil
}

func ClassifyImportPath(importPath string, moduleCtx ModuleContext) (string, ImportOwnership) {
	importedRel, ok := RelativeModulePath(importPath, moduleCtx.Path)
	if !ok {
		return "", ImportOwnershipOutsideModule
	}

	for _, nestedModulePath := range moduleCtx.NestedModulePaths {
		if importPath == nestedModulePath || strings.HasPrefix(importPath, nestedModulePath+"/") {
			return "", ImportOwnershipNestedModule
		}
	}

	return importedRel, ImportOwnershipCurrentModule
}

func readModulePath(goModPath string) (string, error) {
	contents, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", goModPath, err)
	}

	modulePath := strings.TrimSpace(modfile.ModulePath(contents))
	if modulePath == "" {
		return "", fmt.Errorf("%s does not declare a module path", goModPath)
	}

	return modulePath, nil
}

type ImportOwnership int

const (
	ImportOwnershipOutsideModule ImportOwnership = iota
	ImportOwnershipCurrentModule
	ImportOwnershipNestedModule
)
