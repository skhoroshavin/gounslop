package boundarycontrol

import (
	"fmt"
	"sort"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

func reportFalseSharingDiagnostic(pass *analysis.Pass, moduleCtx moduleContext, cfg compiledConfig) error {
	if !cfg.hasSharedSelectors {
		return nil
	}

	diagnostics, err := loadFalseSharingDiagnostics(moduleCtx, cfg)
	if err != nil {
		return err
	}

	msg, ok := diagnostics[pass.Pkg.Path()]
	if !ok || len(pass.Files) == 0 {
		return nil
	}

	pass.Reportf(pass.Files[0].Package, "%s", msg)
	return nil
}

func loadFalseSharingDiagnostics(moduleCtx moduleContext, cfg compiledConfig) (map[string]string, error) {
	cacheKey := falseSharingCacheKey{
		moduleDir: moduleCtx.dir,
		configKey: cfg.cacheKey,
	}

	entryValue, _ := falseSharingCache.LoadOrStore(cacheKey, &falseSharingCacheEntry{})
	entry := entryValue.(*falseSharingCacheEntry)
	entry.once.Do(func() {
		entry.diagnostics, entry.err = analyzeFalseSharing(moduleCtx, cfg)
	})

	return entry.diagnostics, entry.err
}

var falseSharingCache sync.Map

func analyzeFalseSharing(moduleCtx moduleContext, cfg compiledConfig) (map[string]string, error) {
	packagesByPath, err := loadModulePackages(moduleCtx.dir)
	if err != nil {
		return nil, err
	}

	sharedPackages := collectSharedPackages(packagesByPath, moduleCtx, cfg.policies)
	if len(sharedPackages) == 0 {
		return nil, nil
	}

	countSharedPackageConsumers(packagesByPath, sharedPackages, moduleCtx)
	return falseSharingDiagnostics(sharedPackages), nil
}

type falseSharingCacheKey struct {
	moduleDir string
	configKey string
}

type falseSharingCacheEntry struct {
	once        sync.Once
	diagnostics map[string]string
	err         error
}

func loadModulePackages(moduleDir string) (map[string]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedCompiledGoFiles | packages.NeedDeps,
		Dir:  moduleDir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("boundarycontrol: loading packages for shared-package analysis: %w", err)
	}

	packagesByPath := make(map[string]*packages.Package)
	var collect func(pkg *packages.Package)
	collect = func(pkg *packages.Package) {
		if pkg == nil || pkg.PkgPath == "" {
			return
		}

		if _, ok := packagesByPath[pkg.PkgPath]; ok {
			return
		}

		packagesByPath[pkg.PkgPath] = pkg
		for _, importedPkg := range pkg.Imports {
			collect(importedPkg)
		}
	}

	for _, pkg := range pkgs {
		collect(pkg)
	}

	return packagesByPath, nil
}

func collectSharedPackages(
	packagesByPath map[string]*packages.Package,
	moduleCtx moduleContext,
	policies []compiledPolicy,
) map[string]*sharedPackageEntry {
	sharedPackages := make(map[string]*sharedPackageEntry)
	for pkgPath := range packagesByPath {
		relPath, ownership := classifyImportPath(pkgPath, moduleCtx)
		if ownership != importOwnershipCurrentModule {
			continue
		}

		owner, found := resolveOwner(policies, relPath)
		if !found || !owner.shared {
			continue
		}

		sharedPackages[pkgPath] = &sharedPackageEntry{
			consumers: make(map[string]struct{}),
		}
	}

	return sharedPackages
}

func countSharedPackageConsumers(
	packagesByPath map[string]*packages.Package,
	sharedPackages map[string]*sharedPackageEntry,
	moduleCtx moduleContext,
) {
	for pkgPath, pkg := range packagesByPath {
		if _, isSharedPackage := sharedPackages[pkgPath]; isSharedPackage {
			continue
		}

		consumerRelPath, ownership := classifyImportPath(pkgPath, moduleCtx)
		if ownership != importOwnershipCurrentModule || !hasNonTestFiles(pkg.CompiledGoFiles) {
			continue
		}

		consumer := consumerLabel(consumerRelPath)
		for importedPkgPath := range pkg.Imports {
			entry, ok := sharedPackages[importedPkgPath]
			if !ok {
				continue
			}

			entry.consumers[consumer] = struct{}{}
		}
	}
}

func falseSharingDiagnostics(sharedPackages map[string]*sharedPackageEntry) map[string]string {
	packagePaths := make([]string, 0, len(sharedPackages))
	for pkgPath := range sharedPackages {
		packagePaths = append(packagePaths, pkgPath)
	}
	sort.Strings(packagePaths)

	diagnostics := make(map[string]string)
	for _, pkgPath := range packagePaths {
		entry := sharedPackages[pkgPath]
		if len(entry.consumers) >= 2 {
			continue
		}

		consumers := sortedConsumers(entry.consumers)
		if len(consumers) == 0 {
			diagnostics[pkgPath] = "not imported by any entity -> Must be used by 2+ entities"
			continue
		}

		diagnostics[pkgPath] = "only used by: " + joinConsumers(consumers) + " -> Must be used by 2+ entities"
	}

	return diagnostics
}

type sharedPackageEntry struct {
	consumers map[string]struct{}
}

func consumerLabel(relPath string) string {
	if relPath == "" {
		return "."
	}

	return relPath
}

func hasNonTestFiles(filePaths []string) bool {
	for _, filePath := range filePaths {
		if len(filePath) >= len("_test.go") && filePath[len(filePath)-len("_test.go"):] == "_test.go" {
			continue
		}

		return true
	}

	return false
}

func joinConsumers(consumers []string) string {
	if len(consumers) == 0 {
		return ""
	}

	result := consumers[0]
	for i := 1; i < len(consumers); i++ {
		result += ", " + consumers[i]
	}

	return result
}

func sortedConsumers(consumers map[string]struct{}) []string {
	keys := make([]string, 0, len(consumers))
	for consumer := range consumers {
		keys = append(keys, consumer)
	}
	sort.Strings(keys)
	return keys
}
