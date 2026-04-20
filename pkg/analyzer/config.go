package analyzer

import (
	"regexp"
	"strings"
)

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

func ResolveOwner(policies []CompiledPolicy, importerRel string) (CompiledPolicy, bool) {
	var best CompiledPolicy
	var bestCandidate ownerCandidate
	found := false

	for _, policy := range policies {
		candidate, ok := ownerCandidateFor(policy.Selector, importerRel)
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

type ParsedSelector struct {
	Base  string
	Depth int
	Kind  selectorKind
}

type selectorKind int

const (
	SelectorKindRoot selectorKind = iota
	SelectorKindExact
	SelectorKindChildWildcard
	SelectorKindSelfOrChild
)

func ownerCandidateFor(selector ParsedSelector, importerRel string) (ownerCandidate, bool) {
	switch selector.Kind {
	case SelectorKindRoot:
		if importerRel != "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{OwnerDepth: 0}, true
	case SelectorKindExact:
		if importerRel != selector.Base && !strings.HasPrefix(importerRel, selector.Base+"/") {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			OwnerDepth:      selector.Depth,
			SelectorPathLen: selector.Depth,
		}, true
	case SelectorKindChildWildcard:
		prefix := selector.Base + "/"
		if !strings.HasPrefix(importerRel, prefix) {
			return ownerCandidate{}, false
		}

		remainder := strings.TrimPrefix(importerRel, prefix)
		if remainder == "" {
			return ownerCandidate{}, false
		}

		return ownerCandidate{
			OwnerDepth:      selector.Depth + 1,
			Wildcard:        true,
			SelectorPathLen: selector.Depth,
		}, true
	default:
		return ownerCandidate{}, false
	}
}

func betterOwner(candidate, current ownerCandidate) bool {
	if candidate.OwnerDepth != current.OwnerDepth {
		return candidate.OwnerDepth > current.OwnerDepth
	}

	if candidate.Wildcard != current.Wildcard {
		return !candidate.Wildcard
	}

	return candidate.SelectorPathLen > current.SelectorPathLen
}

type ownerCandidate struct {
	OwnerDepth      int
	Wildcard        bool
	SelectorPathLen int
}
