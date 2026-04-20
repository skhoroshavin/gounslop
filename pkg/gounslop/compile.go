package gounslop

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/skhoroshavin/gounslop/pkg/analyzer"
)

func compileConfig(cfg Config) (analyzer.CompiledConfig, error) {
	if cfg.Architecture == nil {
		return analyzer.CompiledConfig{}, nil
	}

	for selector, policy := range cfg.Architecture {
		if policy.Mode != nil {
			return analyzer.CompiledConfig{}, fmt.Errorf(
				"gounslop: architecture[%q].mode is unsupported; migrated false-sharing counts consumers by importing package path only",
				selector,
			)
		}
	}

	arch := normalizeConfig(cfg.Architecture)

	selectors := make([]string, 0, len(arch))
	for k := range arch {
		selectors = append(selectors, k)
	}
	sort.Strings(selectors)

	policies := make([]analyzer.CompiledPolicy, 0, len(selectors))
	hasSharedSelectors := false
	var cacheKeyParts []string
	for _, selector := range selectors {
		parsedKey, err := parseKeySelector(selector)
		if err != nil {
			return analyzer.CompiledConfig{}, fmt.Errorf("architecture[%q]: %w", selector, err)
		}

		policy := arch[selector]
		imports := policy.Imports
		exports := policy.Exports
		parsedImports := make([]analyzer.ParsedSelector, 0, len(imports))
		for i, importSelector := range imports {
			parsedImport, err := parseImportSelector(importSelector)
			if err != nil {
				return analyzer.CompiledConfig{}, fmt.Errorf("architecture[%q].imports[%d]: %w", selector, i, err)
			}

			parsedImports = append(parsedImports, parsedImport)
		}

		compiledExports := make([]*regexp.Regexp, 0, len(exports))
		for i, exportPattern := range exports {
			compiledPattern, err := regexp.Compile("^(?:" + exportPattern + ")$")
			if err != nil {
				return analyzer.CompiledConfig{}, fmt.Errorf("architecture[%q].exports[%d]: invalid regex: %w", selector, i, err)
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

		policies = append(policies, analyzer.CompiledPolicy{
			Selector: parsedKey,
			Imports:  parsedImports,
			Exports:  compiledExports,
			Shared:   policy.Shared,
		})
	}

	return analyzer.CompiledConfig{
		Policies:           policies,
		HasSharedSelectors: hasSharedSelectors,
		CacheKey:           strings.Join(cacheKeyParts, "\x00"),
	}, nil
}

func normalizeConfig(arch Architecture) Architecture {
	if arch == nil {
		return Architecture{}
	}

	normalized := make(Architecture, len(arch))
	for selector, policy := range arch {
		normalizedSelector := strings.TrimSpace(selector)
		imports := slices.Clone(policy.Imports)

		for i := range imports {
			imports[i] = strings.TrimSpace(imports[i])
		}

		exports := slices.Clone(policy.Exports)
		for i := range exports {
			exports[i] = strings.TrimSpace(exports[i])
		}

		normalized[normalizedSelector] = PolicyConfig{
			Imports: imports,
			Exports: exports,
			Shared:  policy.Shared,
		}
	}

	return normalized
}

func parseKeySelector(raw string) (analyzer.ParsedSelector, error) {
	return parseSelector(raw, "key", false)
}

func parseImportSelector(raw string) (analyzer.ParsedSelector, error) {
	return parseSelector(raw, "import", true)
}

func parseSelector(raw, selectorType string, allowSelfOrChild bool) (analyzer.ParsedSelector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return analyzer.ParsedSelector{Kind: analyzer.SelectorKindRoot}, nil
	}

	if strings.HasSuffix(raw, "/*") {
		base := strings.TrimSuffix(raw, "/*")
		if !isValidRelPath(base) {
			return analyzer.ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
		}

		return analyzer.ParsedSelector{
			Base:  base,
			Depth: segmentCount(base),
			Kind:  analyzer.SelectorKindChildWildcard,
		}, nil
	}

	if allowSelfOrChild && strings.HasSuffix(raw, "/+") {
		base := strings.TrimSuffix(raw, "/+")
		if !isValidRelPath(base) {
			return analyzer.ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
		}

		return analyzer.ParsedSelector{
			Base:  base,
			Depth: segmentCount(base),
			Kind:  analyzer.SelectorKindSelfOrChild,
		}, nil
	}

	if !isValidRelPath(raw) {
		return analyzer.ParsedSelector{}, fmt.Errorf("unsupported %s selector %q", selectorType, raw)
	}

	return analyzer.ParsedSelector{
		Base:  raw,
		Depth: segmentCount(raw),
		Kind:  analyzer.SelectorKindExact,
	}, nil
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

func segmentCount(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}
