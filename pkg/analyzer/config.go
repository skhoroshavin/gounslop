package analyzer

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type Policy struct {
	Imports []string
	Exports []string
	Shared  bool
}

func SortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func CompileConfig(architecture map[string]Policy) (CompiledConfig, error) {
	selectors := SortedMapKeys(architecture)

	policies := make([]CompiledPolicy, 0, len(selectors))
	hasSharedSelectors := false
	var cacheKeyParts []string
	for _, selector := range selectors {
		parsedKey, err := ParseKeySelector(selector)
		if err != nil {
			return CompiledConfig{}, fmt.Errorf("architecture[%q]: %w", selector, err)
		}

		policy := architecture[selector]
		imports := policy.Imports
		exports := policy.Exports
		parsedImports := make([]ParsedSelector, 0, len(imports))
		for i, importSelector := range imports {
			parsedImport, err := ParseImportSelector(importSelector)
			if err != nil {
				return CompiledConfig{}, fmt.Errorf("architecture[%q].imports[%d]: %w", selector, i, err)
			}

			parsedImports = append(parsedImports, parsedImport)
		}

		compiledExports := make([]*regexp.Regexp, 0, len(exports))
		for i, exportPattern := range exports {
			compiledPattern, err := regexp.Compile("^(?:" + exportPattern + ")$")
			if err != nil {
				return CompiledConfig{}, fmt.Errorf("architecture[%q].exports[%d]: invalid regex: %w", selector, i, err)
			}

			compiledExports = append(compiledExports, compiledPattern)
		}

		if policy.Shared {
			hasSharedSelectors = true
		}

		cacheKeyParts = append(cacheKeyParts, selector)
		if policy.Shared {
			cacheKeyParts = append(cacheKeyParts, "shared")
		}
		cacheKeyParts = append(cacheKeyParts, imports...)
		for _, exportPattern := range exports {
			cacheKeyParts = append(cacheKeyParts, "export", exportPattern)
		}

		policies = append(policies, CompiledPolicy{
			Selector: parsedKey,
			Imports:  parsedImports,
			Exports:  compiledExports,
			Shared:   policy.Shared,
		})
	}

	return CompiledConfig{
		Policies:           policies,
		HasSharedSelectors: hasSharedSelectors,
		CacheKey:           strings.Join(cacheKeyParts, "\x00"),
	}, nil
}

func NormalizeConfig(architecture map[string]Policy) map[string]Policy {
	if architecture == nil {
		return map[string]Policy{}
	}

	normalized := make(map[string]Policy, len(architecture))
	for selector, policy := range architecture {
		normalizedSelector := strings.TrimSpace(selector)
		imports := slices.Clone(policy.Imports)

		for i := range imports {
			imports[i] = strings.TrimSpace(imports[i])
		}

		exports := slices.Clone(policy.Exports)
		for i := range exports {
			exports[i] = strings.TrimSpace(exports[i])
		}

		normalized[normalizedSelector] = Policy{
			Imports: imports,
			Exports: exports,
			Shared:  policy.Shared,
		}
	}

	return normalized
}

func RelativeModulePath(pkgPath, modulePath string) (string, bool) {
	if pkgPath == modulePath {
		return "", true
	}

	prefix := modulePath + "/"
	if !strings.HasPrefix(pkgPath, prefix) {
		return "", false
	}

	return strings.TrimPrefix(pkgPath, prefix), true
}

func IsSameScopeTooDeep(importerRel, importedRel string) bool {
	importerScope := ScopeFromRelPath(importerRel)
	importedScope := ScopeFromRelPath(importedRel)
	if importerScope != importedScope {
		return false
	}

	importerDepth := DepthWithinScope(importerRel, importerScope)
	importedDepth := DepthWithinScope(importedRel, importedScope)
	return importedDepth > importerDepth+1
}

func IsImmediateChildImport(importerRel, importedRel string) bool {
	if importedRel == "" {
		return false
	}

	prefix := importerRel
	if prefix != "" {
		prefix += "/"
	}
	if !strings.HasPrefix(importedRel, prefix) {
		return false
	}

	return SegmentCount(strings.TrimPrefix(importedRel, prefix)) == 1
}

func ResolveOwner(policies []CompiledPolicy, importerRel string) (CompiledPolicy, bool) {
	var best CompiledPolicy
	var bestCandidate OwnerCandidate
	found := false

	for _, policy := range policies {
		candidate, ok := OwnerCandidateFor(policy.Selector, importerRel)
		if !ok {
			continue
		}

		if !found || BetterOwner(candidate, bestCandidate) {
			best = policy
			bestCandidate = candidate
			found = true
		}
	}

	return best, found
}

type CompiledConfig struct {
	Policies           []CompiledPolicy
	HasSharedSelectors bool
	CacheKey           string
}

type CompiledPolicy struct {
	Selector ParsedSelector
	Imports  []ParsedSelector
	Exports  []*regexp.Regexp
	Shared   bool
}

func OwnerCandidateFor(selector ParsedSelector, importerRel string) (OwnerCandidate, bool) {
	switch selector.Kind {
	case SelectorKindRoot:
		if importerRel != "" {
			return OwnerCandidate{}, false
		}

		return OwnerCandidate{OwnerDepth: 0}, true
	case SelectorKindExact:
		if importerRel != selector.Base && !strings.HasPrefix(importerRel, selector.Base+"/") {
			return OwnerCandidate{}, false
		}

		return OwnerCandidate{
			OwnerDepth:      selector.Depth,
			SelectorPathLen: selector.Depth,
		}, true
	case SelectorKindChildWildcard:
		prefix := selector.Base + "/"
		if !strings.HasPrefix(importerRel, prefix) {
			return OwnerCandidate{}, false
		}

		remainder := strings.TrimPrefix(importerRel, prefix)
		if remainder == "" {
			return OwnerCandidate{}, false
		}

		return OwnerCandidate{
			OwnerDepth:      selector.Depth + 1,
			Wildcard:        true,
			SelectorPathLen: selector.Depth,
		}, true
	default:
		return OwnerCandidate{}, false
	}
}

func BetterOwner(candidate, current OwnerCandidate) bool {
	if candidate.OwnerDepth != current.OwnerDepth {
		return candidate.OwnerDepth > current.OwnerDepth
	}

	if candidate.Wildcard != current.Wildcard {
		return !candidate.Wildcard
	}

	return candidate.SelectorPathLen > current.SelectorPathLen
}

type OwnerCandidate struct {
	OwnerDepth      int
	Wildcard        bool
	SelectorPathLen int
}

func MatchesImportSelectors(selectors []ParsedSelector, importedRel string) bool {
	for _, selector := range selectors {
		if MatchesImportSelector(selector, importedRel) {
			return true
		}
	}

	return false
}

func MatchesImportSelector(selector ParsedSelector, importedRel string) bool {
	switch selector.Kind {
	case SelectorKindRoot:
		return importedRel == ""
	case SelectorKindExact:
		return importedRel == selector.Base
	case SelectorKindChildWildcard:
		return isDirectChild(importedRel, selector.Base)
	case SelectorKindSelfOrChild:
		return importedRel == selector.Base || isDirectChild(importedRel, selector.Base)
	default:
		return false
	}
}

func isDirectChild(importedRel, base string) bool {
	prefix := base + "/"
	if !strings.HasPrefix(importedRel, prefix) {
		return false
	}

	remainder := strings.TrimPrefix(importedRel, prefix)
	return remainder != "" && !strings.Contains(remainder, "/")
}

func ParseKeySelector(raw string) (ParsedSelector, error) {
	return ParseSelector(raw, "key", false)
}

func ParseImportSelector(raw string) (ParsedSelector, error) {
	return ParseSelector(raw, "import", true)
}

func ParseSelector(raw, selectorType string, allowSelfOrChild bool) (ParsedSelector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return ParsedSelector{Kind: SelectorKindRoot}, nil
	}

	if strings.HasSuffix(raw, "/*") {
		base := strings.TrimSuffix(raw, "/*")
		if !isValidRelPath(base) {
			return ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
		}

		return ParsedSelector{
			Base:  base,
			Depth: SegmentCount(base),
			Kind:  SelectorKindChildWildcard,
		}, nil
	}

	if allowSelfOrChild && strings.HasSuffix(raw, "/+") {
		base := strings.TrimSuffix(raw, "/+")
		if !isValidRelPath(base) {
			return ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
		}

		return ParsedSelector{
			Base:  base,
			Depth: SegmentCount(base),
			Kind:  SelectorKindSelfOrChild,
		}, nil
	}

	if !isValidRelPath(raw) {
		return ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
	}

	return ParsedSelector{
		Base:  raw,
		Depth: SegmentCount(raw),
		Kind:  SelectorKindExact,
	}, nil
}

type SelectorKind int

const (
	SelectorKindRoot SelectorKind = iota
	SelectorKindExact
	SelectorKindChildWildcard
	SelectorKindSelfOrChild
)

type ParsedSelector struct {
	Base  string
	Depth int
	Kind  SelectorKind
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

func ScopeFromRelPath(relPath string) string {
	parts := strings.SplitN(relPath, "/", 2)
	return parts[0]
}

func DepthWithinScope(relPath, scope string) int {
	suffix := strings.TrimPrefix(relPath, scope+"/")
	if suffix == relPath {
		return 0
	}

	return strings.Count(suffix, "/") + 1
}

func SegmentCount(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}
