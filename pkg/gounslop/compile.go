package gounslop

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/skhoroshavin/gounslop/pkg/core/boundary"
)

func compileConfig(cfg Config) (boundary.Rules, error) {
	if cfg.Architecture == nil {
		return boundary.Rules{}, nil
	}

	selectors := make([]string, 0, len(cfg.Architecture))
	for selector := range cfg.Architecture {
		selectors = append(selectors, selector)
	}
	sort.Strings(selectors)

	var rules boundary.Rules
	var cacheKeyParts []string

	for _, rawSelector := range selectors {
		selector := strings.TrimSpace(rawSelector)
		policy := cfg.Architecture[rawSelector]

		parsedKey, parts, importPol, exportPol, err := compileSelector(selector, policy)
		if err != nil {
			return boundary.Rules{}, err
		}

		if importPol != nil {
			rules.Import = append(rules.Import, *importPol)
		}
		if exportPol != nil {
			rules.Export = append(rules.Export, *exportPol)
		}
		if policy.Shared {
			rules.Shared.Selectors = append(rules.Shared.Selectors, parsedKey)
			rules.Shared.HasSharedSelectors = true
		}
		cacheKeyParts = append(cacheKeyParts, parts...)
	}

	rules.Shared.CacheKey = strings.Join(cacheKeyParts, "\x00")
	return rules, nil
}

func compileSelector(selector string, policy PolicyConfig) (boundary.Selector, []string, *boundary.ImportPolicy, *boundary.ExportPolicy, error) {
	parsedKey, err := boundary.ParseSelector(selector, false, "key")
	if err != nil {
		return boundary.Selector{}, nil, nil, nil, fmt.Errorf("architecture[%q]: %w", selector, err)
	}

	parts := buildCacheKeyParts(selector, policy)
	importPol, err := buildImportPolicy(selector, policy, parsedKey)
	if err != nil {
		return boundary.Selector{}, nil, nil, nil, err
	}
	exportPol, err := buildExportPolicy(selector, policy, parsedKey)
	if err != nil {
		return boundary.Selector{}, nil, nil, nil, err
	}
	return parsedKey, parts, importPol, exportPol, nil
}

func buildCacheKeyParts(selector string, policy PolicyConfig) []string {
	parts := []string{selector}
	if policy.Shared {
		parts = append(parts, "shared")
	}
	parts = append(parts, policy.Imports...)
	for _, ep := range policy.Exports {
		parts = append(parts, "export", ep)
	}
	return parts
}

func buildImportPolicy(selector string, policy PolicyConfig, parsedKey boundary.Selector) (*boundary.ImportPolicy, error) {
	if len(policy.Imports) == 0 {
		return nil, nil
	}
	parsedImports := make([]boundary.Selector, 0, len(policy.Imports))
	for i, importSelector := range policy.Imports {
		trimmed := strings.TrimSpace(importSelector)
		parsedImport, err := boundary.ParseSelector(trimmed, true, "import")
		if err != nil {
			return nil, fmt.Errorf("architecture[%q].imports[%d]: %w", selector, i, err)
		}
		parsedImports = append(parsedImports, parsedImport)
	}
	return &boundary.ImportPolicy{Selector: parsedKey, Imports: parsedImports}, nil
}

func buildExportPolicy(selector string, policy PolicyConfig, parsedKey boundary.Selector) (*boundary.ExportPolicy, error) {
	if len(policy.Exports) == 0 {
		return nil, nil
	}
	compiledExports := make([]*regexp.Regexp, 0, len(policy.Exports))
	for i, exportPattern := range policy.Exports {
		trimmed := strings.TrimSpace(exportPattern)
		compiledPattern, err := regexp.Compile("^(?:" + trimmed + ")$")
		if err != nil {
			return nil, fmt.Errorf("architecture[%q].exports[%d]: invalid regex: %w", selector, i, err)
		}
		compiledExports = append(compiledExports, compiledPattern)
	}
	return &boundary.ExportPolicy{Selector: parsedKey, Exports: compiledExports}, nil
}
