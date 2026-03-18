package matcher

import "strings"

type Matcher interface {
	Matches(pattern, path string) bool
	MatchesWithMethod(pattern, method, path, reqMethod string) bool
}

type pathMatcher struct{}

// compile-time interface check
var _ Matcher = (*pathMatcher)(nil)

func New() *pathMatcher {
	return &pathMatcher{}
}

func (pm *pathMatcher) Matches(pattern, path string) bool {
	pattern = normalizePath(pattern)
	path = normalizePath(path)

	if pattern == path {
		return true
	}

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}

func (pm *pathMatcher) MatchesWithMethod(pattern, method, path, reqMethod string) bool {
	if method != "" && strings.ToUpper(method) != strings.ToUpper(reqMethod) {
		return false
	}
	return pm.Matches(pattern, path)
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}
