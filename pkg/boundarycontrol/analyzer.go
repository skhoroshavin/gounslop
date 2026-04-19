package boundarycontrol

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "boundarycontrol",
	Doc:      "enforce package import boundaries within a configured module root",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type Config struct {
	ModuleRoot string           `json:"module-root"`
	Selectors  []SelectorPolicy `json:"selectors"`
}

type SelectorPolicy struct {
	Selector string   `json:"selector"`
	Imports  []string `json:"imports"`
}

func init() {
	Analyzer.Flags.StringVar(&moduleRootFlag, "module-root", "", "Go module path prefix (e.g. github.com/org/repo)")
	Analyzer.Flags.StringVar(&selectorsFlag, "selectors", "[]", "JSON-encoded boundarycontrol selectors")
}

func ValidateConfig(cfg Config) error {
	cfg = normalizeConfig(cfg)
	if cfg.ModuleRoot == "" {
		return fmt.Errorf("boundarycontrol: module-root is required")
	}

	for i, policy := range cfg.Selectors {
		if _, err := parseKeySelector(policy.Selector); err != nil {
			return fmt.Errorf("boundarycontrol: selectors[%d].selector: %w", i, err)
		}

		for j, selector := range policy.Imports {
			if _, err := parseImportSelector(selector); err != nil {
				return fmt.Errorf("boundarycontrol: selectors[%d].imports[%d]: %w", i, j, err)
			}
		}
	}

	return nil
}

func run(pass *analysis.Pass) (any, error) {
	cfg, err := configFromFlags()
	if err != nil {
		return nil, err
	}

	importerRel, ok := relativeModulePath(pass.Pkg.Path(), cfg.ModuleRoot)
	if !ok {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.ImportSpec)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return
		}

		importedRel, ok := relativeModulePath(importPath, cfg.ModuleRoot)
		if !ok {
			return
		}

		if isSameScopeTooDeep(importerRel, importedRel) {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
			return
		}

		if isImmediateChildImport(importerRel, importedRel) {
			return
		}

		owner, found := resolveOwner(cfg.Selectors, importerRel)
		if found && matchesImportSelectors(owner.Imports, importedRel) {
			return
		}

		pass.Reportf(imp.Pos(), "undeclared boundarycontrol import: %s", importPath)
	})

	return nil, nil
}

func configFromFlags() (Config, error) {
	cfg := Config{ModuleRoot: moduleRootFlag}
	if selectorsFlag != "" {
		if err := json.Unmarshal([]byte(selectorsFlag), &cfg.Selectors); err != nil {
			return Config{}, fmt.Errorf("boundarycontrol: invalid selectors setting: %w", err)
		}
	}

	cfg = normalizeConfig(cfg)
	if err := ValidateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

var (
	moduleRootFlag string
	selectorsFlag  string
)

func normalizeConfig(cfg Config) Config {
	cfg.ModuleRoot = strings.TrimRight(strings.TrimSpace(cfg.ModuleRoot), "/")
	if cfg.Selectors == nil {
		cfg.Selectors = []SelectorPolicy{}
	}

	for i := range cfg.Selectors {
		cfg.Selectors[i].Selector = strings.TrimSpace(cfg.Selectors[i].Selector)
		if cfg.Selectors[i].Imports == nil {
			cfg.Selectors[i].Imports = []string{}
		}

		for j := range cfg.Selectors[i].Imports {
			cfg.Selectors[i].Imports[j] = strings.TrimSpace(cfg.Selectors[i].Imports[j])
		}
	}

	return cfg
}

func relativeModulePath(pkgPath, moduleRoot string) (string, bool) {
	if pkgPath == moduleRoot {
		return "", true
	}

	prefix := moduleRoot + "/"
	if !strings.HasPrefix(pkgPath, prefix) {
		return "", false
	}

	return strings.TrimPrefix(pkgPath, prefix), true
}

func isSameScopeTooDeep(importerRel, importedRel string) bool {
	importerScope := scopeFromRelPath(importerRel)
	importedScope := scopeFromRelPath(importedRel)
	if importerScope != importedScope {
		return false
	}

	importerDepth := depthWithinScope(importerRel, importerScope)
	importedDepth := depthWithinScope(importedRel, importedScope)
	return importedDepth > importerDepth+1
}

func isImmediateChildImport(importerRel, importedRel string) bool {
	if importedRel == "" {
		return false
	}

	if importerRel == "" {
		return segmentCount(importedRel) == 1
	}

	prefix := importerRel + "/"
	if !strings.HasPrefix(importedRel, prefix) {
		return false
	}

	return segmentCount(strings.TrimPrefix(importedRel, prefix)) == 1
}

func resolveOwner(policies []SelectorPolicy, importerRel string) (SelectorPolicy, bool) {
	var best SelectorPolicy
	var bestCandidate ownerCandidate
	found := false

	for i, policy := range policies {
		candidate, ok := ownerCandidateFor(policy.Selector, importerRel)
		if !ok {
			continue
		}

		candidate.index = i
		if !found || betterOwner(candidate, bestCandidate) {
			best = policy
			bestCandidate = candidate
			found = true
		}
	}

	return best, found
}

func ownerCandidateFor(selector, importerRel string) (ownerCandidate, bool) {
	parsed, err := parseKeySelector(selector)
	if err != nil {
		return ownerCandidate{}, false
	}

	switch parsed.kind {
	case selectorKindRoot:
		if importerRel != "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{ownerDepth: 0}, true
	case selectorKindExact:
		if importerRel != parsed.base && !strings.HasPrefix(importerRel, parsed.base+"/") {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			ownerDepth:      parsed.depth,
			selectorPathLen: parsed.depth,
		}, true
	case selectorKindChildWildcard:
		prefix := parsed.base + "/"
		if !strings.HasPrefix(importerRel, prefix) {
			return ownerCandidate{}, false
		}

		remainder := strings.TrimPrefix(importerRel, prefix)
		if remainder == "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			ownerDepth:      parsed.depth + 1,
			wildcard:        true,
			selectorPathLen: parsed.depth,
		}, true
	default:
		return ownerCandidate{}, false
	}
}

func betterOwner(candidate, current ownerCandidate) bool {
	if candidate.ownerDepth != current.ownerDepth {
		return candidate.ownerDepth > current.ownerDepth
	}

	if candidate.wildcard != current.wildcard {
		return !candidate.wildcard
	}

	if candidate.selectorPathLen != current.selectorPathLen {
		return candidate.selectorPathLen > current.selectorPathLen
	}

	return candidate.index < current.index
}

type ownerCandidate struct {
	ownerDepth      int
	wildcard        bool
	selectorPathLen int
	index           int
}

func matchesImportSelectors(selectors []string, importedRel string) bool {
	for _, selector := range selectors {
		if matchesImportSelector(selector, importedRel) {
			return true
		}
	}

	return false
}

func matchesImportSelector(selector, importedRel string) bool {
	parsed, err := parseImportSelector(selector)
	if err != nil {
		return false
	}

	switch parsed.kind {
	case selectorKindRoot:
		return importedRel == ""
	case selectorKindExact:
		return importedRel == parsed.base
	case selectorKindChildWildcard:
		prefix := parsed.base + "/"
		if !strings.HasPrefix(importedRel, prefix) {
			return false
		}

		remainder := strings.TrimPrefix(importedRel, prefix)
		return remainder != "" && !strings.Contains(remainder, "/")
	case selectorKindSelfOrChild:
		if importedRel == parsed.base {
			return true
		}

		prefix := parsed.base + "/"
		if !strings.HasPrefix(importedRel, prefix) {
			return false
		}

		remainder := strings.TrimPrefix(importedRel, prefix)
		return remainder != "" && !strings.Contains(remainder, "/")
	default:
		return false
	}
}

func parseKeySelector(raw string) (parsedSelector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return parsedSelector{kind: selectorKindRoot}, nil
	}

	if strings.HasSuffix(raw, "/*") {
		base := strings.TrimSuffix(raw, "/*")
		if !isValidRelPath(base) {
			return parsedSelector{}, fmt.Errorf("unsupported key selector %q", raw)
		}

		return parsedSelector{
			base:  base,
			depth: segmentCount(base),
			kind:  selectorKindChildWildcard,
		}, nil
	}

	if !isValidRelPath(raw) {
		return parsedSelector{}, fmt.Errorf("unsupported key selector %q", raw)
	}

	return parsedSelector{
		base:  raw,
		depth: segmentCount(raw),
		kind:  selectorKindExact,
	}, nil
}

func parseImportSelector(raw string) (parsedSelector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return parsedSelector{kind: selectorKindRoot}, nil
	}

	if strings.HasSuffix(raw, "/*") {
		base := strings.TrimSuffix(raw, "/*")
		if !isValidRelPath(base) {
			return parsedSelector{}, fmt.Errorf("unsupported import selector %q", raw)
		}

		return parsedSelector{
			base:  base,
			depth: segmentCount(base),
			kind:  selectorKindChildWildcard,
		}, nil
	}

	if strings.HasSuffix(raw, "/+") {
		base := strings.TrimSuffix(raw, "/+")
		if !isValidRelPath(base) {
			return parsedSelector{}, fmt.Errorf("unsupported import selector %q", raw)
		}

		return parsedSelector{
			base:  base,
			depth: segmentCount(base),
			kind:  selectorKindSelfOrChild,
		}, nil
	}

	if !isValidRelPath(raw) {
		return parsedSelector{}, fmt.Errorf("unsupported import selector %q", raw)
	}

	return parsedSelector{
		base:  raw,
		depth: segmentCount(raw),
		kind:  selectorKindExact,
	}, nil
}

type selectorKind int

const (
	selectorKindRoot selectorKind = iota
	selectorKindExact
	selectorKindChildWildcard
	selectorKindSelfOrChild
)

type parsedSelector struct {
	base  string
	depth int
	kind  selectorKind
}

func isValidRelPath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") || strings.Contains(path, "//") {
		return false
	}

	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "*+") {
			return false
		}
	}

	return true
}

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

	return strings.Count(suffix, "/") + 1
}

func segmentCount(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}
