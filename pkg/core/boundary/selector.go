package boundary

import (
	"fmt"
	"strings"
)

func ParseSelector(raw string, allowSelfOrChildren bool, name string) (Selector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return Selector{Kind: KindRoot}, nil
	}

	if strings.HasSuffix(raw, "/*") {
		base := strings.TrimSuffix(raw, "/*")
		if !isValidRelPath(base) {
			return Selector{}, fmt.Errorf("unsupported %s selector %q", name, raw)
		}

		return Selector{
			Base:  base,
			Depth: SegmentCount(base),
			Kind:  KindChildren,
		}, nil
	}

	if allowSelfOrChildren && strings.HasSuffix(raw, "/+") {
		base := strings.TrimSuffix(raw, "/+")
		if !isValidRelPath(base) {
			return Selector{}, fmt.Errorf("unsupported %s selector %q", name, raw)
		}

		return Selector{
			Base:  base,
			Depth: SegmentCount(base),
			Kind:  KindSelfOrChildren,
		}, nil
	}

	if !isValidRelPath(raw) {
		return Selector{}, fmt.Errorf("unsupported %s selector %q", name, raw)
	}

	return Selector{
		Base:  raw,
		Depth: SegmentCount(raw),
		Kind:  KindExact,
	}, nil
}

func isValidRelPath(path string) bool {
	if !isValidBasePath(path) {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if !isValidPathSegment(segment) {
			return false
		}
	}
	return true
}

func isValidBasePath(path string) bool {
	return path != "" && !strings.HasPrefix(path, "/") && !strings.HasSuffix(path, "/") && !strings.Contains(path, "//")
}

func isValidPathSegment(segment string) bool {
	return segment != "" && segment != "." && segment != ".." && !strings.ContainsAny(segment, "*+")
}

func SegmentCount(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}
