package nofalsesharing

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
)

// Analyzer wraps the false-sharing check as an analysis.Analyzer for golangci-lint integration.
// It loads all packages once (via sync.Once) and reports diagnostics on shared packages
// that have fewer than 2 consumers.
var Analyzer = &analysis.Analyzer{
	Name: "nofalsesharing",
	Doc:  "checks that packages in shared directories are actually used by multiple consumers",
	Run:  runAnalyzer,
}

func init() {
	Analyzer.Flags.StringVar(&sharedDirsFlag, "shared-dirs", "", "comma-separated shared directory paths (e.g. pkg/shared,internal/common)")
	Analyzer.Flags.StringVar(&modeFlag, "mode", string(FileMode), "sharing mode: file or dir")
	Analyzer.Flags.StringVar(&moduleRootFlag, "module-root", "", "Go module path (auto-detected if not specified)")
}

func runAnalyzer(pass *analysis.Pass) (any, error) {
	if sharedDirsFlag == "" {
		return nil, nil
	}

	cacheOnce.Do(func() {
		var dirs []DirConfig
		for _, d := range strings.Split(sharedDirsFlag, ",") {
			d = strings.TrimSpace(d)
			if d == "" {
				continue
			}
			dirs = append(dirs, DirConfig{
				Path: d,
				Mode: SharingMode(modeFlag),
			})
		}

		dir := findProjectRoot(pass)
		diags, err := Run(dir, moduleRootFlag, dirs)
		if err != nil {
			cachedErr = err
			return
		}

		cachedDiags = make(map[string]string, len(diags))
		for _, d := range diags {
			cachedDiags[d.Package] = d.Message
		}
	})

	if cachedErr != nil {
		return nil, cachedErr
	}

	msg, ok := cachedDiags[pass.Pkg.Path()]
	if !ok {
		return nil, nil
	}

	if len(pass.Files) > 0 {
		pass.Reportf(pass.Files[0].Package, "%s", msg)
	}

	return nil, nil
}

var (
	sharedDirsFlag string
	modeFlag       string
	moduleRootFlag string

	cacheOnce   sync.Once
	cachedDiags map[string]string // pkgPath -> message
	cachedErr   error
)

func findProjectRoot(pass *analysis.Pass) string {
	if len(pass.Files) == 0 {
		return "."
	}
	pos := pass.Fset.Position(pass.Files[0].Package)
	dir := filepath.Dir(pos.Filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
