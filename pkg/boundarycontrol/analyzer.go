package boundarycontrol

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "boundarycontrol",
	Doc:      "enforce package import boundaries and shared-package usage within the discovered Go module",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type Config struct {
	Architecture map[string]Policy `json:"architecture"`
}

type Policy struct {
	Imports []string `json:"imports"`
	Shared  bool     `json:"shared"`
}

func init() {
	Analyzer.Flags.StringVar(&architectureFlag, "architecture", "{}", "JSON-encoded boundarycontrol architecture")
}

func ValidateConfig(cfg Config) error {
	_, err := compileConfig(normalizeConfig(cfg))
	return err
}

func run(pass *analysis.Pass) (any, error) {
	cfg, err := configFromFlags()
	if err != nil {
		return nil, err
	}

	moduleCtx, err := discoverModuleContext(pass)
	if err != nil {
		return nil, err
	}

	importerRel, ok := relativeModulePath(pass.Pkg.Path(), moduleCtx.path)
	if !ok {
		return nil, fmt.Errorf("boundarycontrol: package %q is outside discovered module %q", pass.Pkg.Path(), moduleCtx.path)
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.ImportSpec)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		imp := n.(*ast.ImportSpec)
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return
		}

		importedRel, ownership := classifyImportPath(importPath, moduleCtx)
		if ownership != importOwnershipCurrentModule {
			return
		}

		if isSameScopeTooDeep(importerRel, importedRel) {
			pass.Reportf(imp.Pos(), "%s is too deep (max 1 level below importer within same scope).", importPath)
			return
		}

		if isImmediateChildImport(importerRel, importedRel) {
			return
		}

		owner, found := resolveOwner(cfg.policies, importerRel)
		if found && matchesImportSelectors(owner.imports, importedRel) {
			return
		}

		pass.Reportf(imp.Pos(), "undeclared boundarycontrol import: %s", importPath)
	})

	if err := reportFalseSharingDiagnostic(pass, moduleCtx, cfg); err != nil {
		return nil, err
	}

	return nil, nil
}

func configFromFlags() (compiledConfig, error) {
	var cfg Config
	if architectureFlag != "" {
		decoder := json.NewDecoder(strings.NewReader(architectureFlag))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cfg.Architecture); err != nil {
			return compiledConfig{}, fmt.Errorf("boundarycontrol: invalid architecture setting: %w", err)
		}
	}

	compiled, err := compileConfig(normalizeConfig(cfg))
	if err != nil {
		return compiledConfig{}, err
	}

	return compiled, nil

}

var architectureFlag string

func compileConfig(cfg Config) (compiledConfig, error) {
	selectors := make([]string, 0, len(cfg.Architecture))
	for selector := range cfg.Architecture {
		selectors = append(selectors, selector)
	}
	sort.Strings(selectors)

	policies := make([]compiledPolicy, 0, len(selectors))
	hasSharedSelectors := false
	var cacheKey strings.Builder
	for _, selector := range selectors {
		parsedKey, err := parseKeySelector(selector)
		if err != nil {
			return compiledConfig{}, fmt.Errorf("boundarycontrol: architecture[%q]: %w", selector, err)
		}

		policy := cfg.Architecture[selector]
		imports := policy.Imports
		parsedImports := make([]parsedSelector, 0, len(imports))
		for i, importSelector := range imports {
			parsedImport, err := parseImportSelector(importSelector)
			if err != nil {
				return compiledConfig{}, fmt.Errorf("boundarycontrol: architecture[%q].imports[%d]: %w", selector, i, err)
			}

			parsedImports = append(parsedImports, parsedImport)
		}

		if policy.Shared {
			hasSharedSelectors = true
		}

		cacheKey.WriteString(selector)
		cacheKey.WriteByte(0)
		if policy.Shared {
			cacheKey.WriteString("shared")
		}
		cacheKey.WriteByte(0)
		for _, importSelector := range imports {
			cacheKey.WriteString(importSelector)
			cacheKey.WriteByte(0)
		}

		policies = append(policies, compiledPolicy{
			selector: parsedKey,
			imports:  parsedImports,
			shared:   policy.Shared,
		})
	}

	return compiledConfig{
		policies:           policies,
		hasSharedSelectors: hasSharedSelectors,
		cacheKey:           cacheKey.String(),
	}, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.Architecture == nil {
		cfg.Architecture = map[string]Policy{}
		return cfg
	}

	normalized := make(map[string]Policy, len(cfg.Architecture))
	for selector, policy := range cfg.Architecture {
		normalizedSelector := strings.TrimSpace(selector)
		imports := append([]string(nil), policy.Imports...)
		if imports == nil {
			imports = []string{}
		}

		for i := range imports {
			imports[i] = strings.TrimSpace(imports[i])
		}

		normalized[normalizedSelector] = Policy{
			Imports: imports,
			Shared:  policy.Shared,
		}
	}

	cfg.Architecture = normalized
	return cfg
}

func relativeModulePath(pkgPath, modulePath string) (string, bool) {
	if pkgPath == modulePath {
		return "", true
	}

	prefix := modulePath + "/"
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

func resolveOwner(policies []compiledPolicy, importerRel string) (compiledPolicy, bool) {
	var best compiledPolicy
	var bestCandidate ownerCandidate
	found := false

	for _, policy := range policies {
		candidate, ok := ownerCandidateFor(policy.selector, importerRel)
		if !ok {
			continue
		}

		if !found || betterOwner(candidate, bestCandidate) {
			best = policy
			bestCandidate = candidate
			found = true
		}
	}

	return best, found
}

type compiledConfig struct {
	policies           []compiledPolicy
	hasSharedSelectors bool
	cacheKey           string
}

type compiledPolicy struct {
	selector parsedSelector
	imports  []parsedSelector
	shared   bool
}

func ownerCandidateFor(selector parsedSelector, importerRel string) (ownerCandidate, bool) {
	switch selector.kind {
	case selectorKindRoot:
		if importerRel != "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{ownerDepth: 0}, true
	case selectorKindExact:
		if importerRel != selector.base && !strings.HasPrefix(importerRel, selector.base+"/") {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			ownerDepth:      selector.depth,
			selectorPathLen: selector.depth,
		}, true
	case selectorKindChildWildcard:
		prefix := selector.base + "/"
		if !strings.HasPrefix(importerRel, prefix) {
			return ownerCandidate{}, false
		}

		remainder := strings.TrimPrefix(importerRel, prefix)
		if remainder == "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			ownerDepth:      selector.depth + 1,
			wildcard:        true,
			selectorPathLen: selector.depth,
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

	return candidate.selectorPathLen > current.selectorPathLen
}

type ownerCandidate struct {
	ownerDepth      int
	wildcard        bool
	selectorPathLen int
}

func matchesImportSelectors(selectors []parsedSelector, importedRel string) bool {
	for _, selector := range selectors {
		if matchesImportSelector(selector, importedRel) {
			return true
		}
	}

	return false
}

func matchesImportSelector(selector parsedSelector, importedRel string) bool {
	switch selector.kind {
	case selectorKindRoot:
		return importedRel == ""
	case selectorKindExact:
		return importedRel == selector.base
	case selectorKindChildWildcard:
		prefix := selector.base + "/"
		if !strings.HasPrefix(importedRel, prefix) {
			return false
		}

		remainder := strings.TrimPrefix(importedRel, prefix)
		return remainder != "" && !strings.Contains(remainder, "/")
	case selectorKindSelfOrChild:
		if importedRel == selector.base {
			return true
		}

		prefix := selector.base + "/"
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
