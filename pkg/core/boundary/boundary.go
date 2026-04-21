package boundary

import (
	"regexp"
	"strings"
)

type ImportPolicy struct {
	Selector Selector
	Imports  []Selector
}

type ExportPolicy struct {
	Selector Selector
	Exports  []*regexp.Regexp
}

type SharedRules struct {
	Selectors          []Selector
	HasSharedSelectors bool
	CacheKey           string
}

type Rules struct {
	Import []ImportPolicy
	Export []ExportPolicy
	Shared SharedRules
}

type Selector struct {
	Base  string
	Depth int
	Kind  Kind
}

type Kind int

const (
	KindRoot Kind = iota
	KindExact
	KindChildren
	KindSelfOrChildren
)

func FindImportPolicy(policies []ImportPolicy, relPath string) (ImportPolicy, bool) {
	return findBest(policies, relPath, func(p ImportPolicy) Selector { return p.Selector })
}

func FindExportPolicy(policies []ExportPolicy, relPath string) (ExportPolicy, bool) {
	return findBest(policies, relPath, func(p ExportPolicy) Selector { return p.Selector })
}

func FindSharedSelector(selectors []Selector, relPath string) (Selector, bool) {
	return findBest(selectors, relPath, func(s Selector) Selector { return s })
}

func findBest[T any](policies []T, relPath string, selector func(T) Selector) (T, bool) {
	var best T
	var bestMatch policyMatch
	found := false

	for _, p := range policies {
		match, ok := selector(p).match(relPath)
		if !ok {
			continue
		}

		if !found || match.isMoreSpecificThan(bestMatch) {
			best = p
			bestMatch = match
			found = true
		}
	}

	return best, found
}

func (candidate policyMatch) isMoreSpecificThan(other policyMatch) bool {
	if candidate.depth != other.depth {
		return candidate.depth > other.depth
	}

	if candidate.isWildcard != other.isWildcard {
		return !candidate.isWildcard
	}

	return candidate.selectorDepth > other.selectorDepth
}

func (s Selector) Covers(relPath string) bool {
	_, ok := s.match(relPath)
	return ok
}

func (s Selector) match(relPath string) (policyMatch, bool) {
	switch s.Kind {
	case KindRoot:
		return policyMatch{}, relPath == ""
	case KindExact:
		return matchExact(s, relPath)
	case KindSelfOrChildren:
		if relPath == s.Base {
			return policyMatch{depth: s.Depth, selectorDepth: s.Depth}, true
		}
		return matchChild(s, relPath)
	case KindChildren:
		return matchChild(s, relPath)
	default:
		return policyMatch{}, false
	}
}

func matchExact(s Selector, relPath string) (policyMatch, bool) {
	if relPath != s.Base && !strings.HasPrefix(relPath, s.Base+"/") {
		return policyMatch{}, false
	}
	return policyMatch{depth: s.Depth, selectorDepth: s.Depth}, true
}

func matchChild(s Selector, relPath string) (policyMatch, bool) {
	prefix := s.Base + "/"
	if !strings.HasPrefix(relPath, prefix) || strings.TrimPrefix(relPath, prefix) == "" {
		return policyMatch{}, false
	}
	return policyMatch{depth: s.Depth + 1, isWildcard: true, selectorDepth: s.Depth}, true
}

type policyMatch struct {
	depth         int
	isWildcard    bool
	selectorDepth int
}

func (s Selector) MatchesImport(importRel string) bool {
	switch s.Kind {
	case KindRoot:
		return importRel == ""
	case KindExact:
		return importRel == s.Base
	case KindChildren:
		return isDirectChild(importRel, s.Base)
	case KindSelfOrChildren:
		return importRel == s.Base || isDirectChild(importRel, s.Base)
	default:
		return false
	}
}

func isDirectChild(importRel, base string) bool {
	prefix := base + "/"
	if !strings.HasPrefix(importRel, prefix) {
		return false
	}

	remainder := strings.TrimPrefix(importRel, prefix)
	return remainder != "" && !strings.Contains(remainder, "/")
}
