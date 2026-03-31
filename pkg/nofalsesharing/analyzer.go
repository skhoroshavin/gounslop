package nofalsesharing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// SharingMode determines how consumers are counted.
type SharingMode string

const (
	// FileMode counts each importing file as a separate consumer.
	FileMode SharingMode = "file"
	// DirMode counts each importing package's directory as a consumer.
	DirMode SharingMode = "dir"
)

// DirConfig specifies a shared directory and its sharing mode.
type DirConfig struct {
	Path string
	Mode SharingMode
}

// Diagnostic represents a single false-sharing violation.
type Diagnostic struct {
	Package string
	Message string
}

// Run performs the false-sharing analysis on all packages in the given directory.
func Run(dir string, moduleRoot string, sharedDirs []DirConfig) ([]Diagnostic, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedFiles |
			packages.NeedCompiledGoFiles | packages.NeedDeps,
		Dir: dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	if moduleRoot == "" {
		// Infer module root from first package
		for _, pkg := range pkgs {
			if pkg.PkgPath != "" {
				moduleRoot = inferModuleRoot(pkg.PkgPath, dir)
				break
			}
		}
	}

	// Build package map
	allPkgs := make(map[string]*packages.Package)
	var collectPkgs func(pkg *packages.Package)
	collectPkgs = func(pkg *packages.Package) {
		if _, ok := allPkgs[pkg.PkgPath]; ok {
			return
		}
		allPkgs[pkg.PkgPath] = pkg
		for _, imp := range pkg.Imports {
			collectPkgs(imp)
		}
	}
	for _, pkg := range pkgs {
		collectPkgs(pkg)
	}

	// Index shared packages
	type moduleEntry struct {
		pkgPath   string
		mode      SharingMode
		consumers map[string]bool
	}
	modules := make(map[string]*moduleEntry)

	for _, dirCfg := range sharedDirs {
		sharedPrefix := moduleRoot + "/" + dirCfg.Path
		for pkgPath := range allPkgs {
			if pkgPath == sharedPrefix || strings.HasPrefix(pkgPath, sharedPrefix+"/") {
				mode := dirCfg.Mode
				if mode == "" {
					mode = FileMode
				}
				modules[pkgPath] = &moduleEntry{
					pkgPath:   pkgPath,
					mode:      mode,
					consumers: make(map[string]bool),
				}
			}
		}
	}

	// Count consumers
	for _, pkg := range allPkgs {
		// Skip shared packages themselves
		if _, isShared := modules[pkg.PkgPath]; isShared {
			continue
		}

		for importPath := range pkg.Imports {
			entry, ok := modules[importPath]
			if !ok {
				continue
			}

			// Count consumers from non-test files
			for _, file := range pkg.CompiledGoFiles {
				if isTestFile(file) {
					continue
				}

				consumer := deriveEntity(pkg.PkgPath, moduleRoot, file, entry.mode)
				entry.consumers[consumer] = true
			}
		}
	}

	// Collect diagnostics
	var diags []Diagnostic
	for _, entry := range modules {
		if len(entry.consumers) >= 2 {
			continue
		}

		consumers := sortedKeys(entry.consumers)
		var description string
		if len(consumers) == 0 {
			description = "not imported by any entity"
		} else {
			description = "only used by: " + strings.Join(consumers, ", ")
		}

		diags = append(diags, Diagnostic{
			Package: entry.pkgPath,
			Message: description + " -> Must be used by 2+ entities",
		})
	}

	sort.Slice(diags, func(i, j int) bool {
		return diags[i].Package < diags[j].Package
	})

	return diags, nil
}

func deriveEntity(pkgPath, moduleRoot, filePath string, mode SharingMode) string {
	relPkg := strings.TrimPrefix(pkgPath, moduleRoot+"/")
	if mode == FileMode {
		return relPkg + "/" + filepath.Base(filePath)
	}
	// DirMode: use the package path segments up to maxDirDepth
	parts := strings.Split(relPkg, "/")
	if len(parts) > maxDirDepth {
		parts = parts[:maxDirDepth]
	}
	return strings.Join(parts, "/")
}

const maxDirDepth = 3

func isTestFile(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

func inferModuleRoot(pkgPath, dir string) string {
	// Parse go.mod to get the module path
	absDir, _ := filepath.Abs(dir)
	data, err := os.ReadFile(filepath.Join(absDir, "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "module"))
			}
		}
	}
	return pkgPath
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
