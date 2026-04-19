package boundarycontrol

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

func discoverModuleContext(pass *analysis.Pass) (moduleContext, error) {
	if len(pass.Files) == 0 {
		return moduleContext{}, fmt.Errorf("boundarycontrol: no files available to discover module scope")
	}

	packageFile := pass.Fset.PositionFor(pass.Files[0].Package, false).Filename
	if packageFile == "" {
		return moduleContext{}, fmt.Errorf("boundarycontrol: no file path available to discover module scope for %q", pass.Pkg.Path())
	}

	goModPath, err := nearestGoMod(filepath.Dir(packageFile))
	if err != nil {
		return moduleContext{}, err
	}
	if goModPath == "" {
		return moduleContext{}, fmt.Errorf("boundarycontrol: module scope could not be discovered from go.mod for %q", pass.Pkg.Path())
	}

	moduleDir := filepath.Dir(goModPath)
	return loadModuleContext(moduleDir, goModPath)
}

func loadModuleContext(moduleDir, goModPath string) (moduleContext, error) {
	if cached, ok := moduleContextCache.Load(moduleDir); ok {
		return cached.(moduleContext), nil
	}

	modulePath, err := readModulePath(goModPath)
	if err != nil {
		return moduleContext{}, err
	}

	nestedModulePaths, err := discoverNestedModulePaths(moduleDir)
	if err != nil {
		return moduleContext{}, err
	}

	ctx := moduleContext{
		dir:               moduleDir,
		path:              modulePath,
		nestedModulePaths: nestedModulePaths,
	}

	actual, _ := moduleContextCache.LoadOrStore(moduleDir, ctx)
	return actual.(moduleContext), nil
}

var moduleContextCache sync.Map

func nearestGoMod(startDir string) (string, error) {
	currentDir := startDir
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		_, err := os.Stat(goModPath)
		if err == nil {
			return goModPath, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("boundarycontrol: checking %s: %w", goModPath, err)
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
		return nil, fmt.Errorf("boundarycontrol: scanning nested modules under %s: %w", moduleDir, err)
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

func classifyImportPath(importPath string, moduleCtx moduleContext) (string, importOwnership) {
	importedRel, ok := relativeModulePath(importPath, moduleCtx.path)
	if !ok {
		return "", importOwnershipOutsideModule
	}

	for _, nestedModulePath := range moduleCtx.nestedModulePaths {
		if importPath == nestedModulePath || strings.HasPrefix(importPath, nestedModulePath+"/") {
			return "", importOwnershipNestedModule
		}
	}

	return importedRel, importOwnershipCurrentModule
}

type moduleContext struct {
	dir               string
	path              string
	nestedModulePaths []string
}

func readModulePath(goModPath string) (string, error) {
	contents, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("boundarycontrol: reading %s: %w", goModPath, err)
	}

	modulePath := strings.TrimSpace(modfile.ModulePath(contents))
	if modulePath == "" {
		return "", fmt.Errorf("boundarycontrol: %s does not declare a module path", goModPath)
	}

	return modulePath, nil
}

type importOwnership int

const (
	importOwnershipOutsideModule importOwnership = iota
	importOwnershipCurrentModule
	importOwnershipNestedModule
)
