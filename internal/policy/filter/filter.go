package filter

import (
	"fmt"
	"strings"

	"github.com/abac/proxy/internal/policy"
)

type Filterer interface {
	Apply(data any, filter policy.ResponseFilter) (any, error)
}

type responseFilterer struct{}

// compile-time interface check
var _ Filterer = (*responseFilterer)(nil)

func New() Filterer {
	return &responseFilterer{}
}

func (rf *responseFilterer) Apply(data any, f policy.ResponseFilter) (any, error) {
	if f.Type == policy.FilterTypeInclude {
		return rf.applyInclude(data, f.Fields)
	}
	return rf.applyExclude(data, f.Fields)
}

func (rf *responseFilterer) applyInclude(data any, fields []string) (any, error) {
	tree := buildIncludeTree(fields)
	return rf.filterWithTree(data, tree, "")
}

func (rf *responseFilterer) filterWithTree(data any, node *pathNode, currentPath string) (any, error) {
	if node == nil {
		return nil, nil
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)

		for key, value := range v {
			childPath := key
			if currentPath != "" {
				childPath = currentPath + "." + key
			}

			if child, exists := node.children[key]; exists {
				if child.isTerminal {
					result[key] = value
				} else {
					filtered, err := rf.filterWithTree(value, child, childPath)
					if err != nil {
						return nil, err
					}
					if filtered != nil {
						result[key] = filtered
					}
				}
			} else if wildcard, exists := node.children["*"]; exists {
				if wildcard.isTerminal {
					result[key] = value
				} else {
					filtered, err := rf.filterWithTree(value, wildcard, childPath)
					if err != nil {
						return nil, err
					}
					if filtered != nil {
						result[key] = filtered
					}
				}
			}
		}

		if len(result) == 0 {
			return nil, nil
		}
		return result, nil

	case []any:
		if arrayChild, exists := node.children["[]"]; exists {
			result := make([]any, 0)
			for i, item := range v {
				itemPath := fmt.Sprintf("[%d]", i)
				if currentPath != "" {
					itemPath = currentPath + itemPath
				}
				filtered, err := rf.filterWithTree(item, arrayChild, itemPath)
				if err != nil {
					return nil, err
				}
				if filtered != nil {
					result = append(result, filtered)
				}
			}
			return result, nil
		}

		if wildcard, exists := node.children["*"]; exists {
			result := make([]any, 0)
			for i, item := range v {
				itemPath := fmt.Sprintf("[%d]", i)
				if currentPath != "" {
					itemPath = currentPath + itemPath
				}
				filtered, err := rf.filterWithTree(item, wildcard, itemPath)
				if err != nil {
					return nil, err
				}
				if filtered != nil {
					result = append(result, filtered)
				}
			}
			return result, nil
		}

		return nil, nil

	default:
		if node.isTerminal {
			return data, nil
		}
		return nil, nil
	}
}

func (rf *responseFilterer) applyExclude(data any, fields []string) (any, error) {
	patterns := make([]string, len(fields))
	copy(patterns, fields)
	return rf.excludeRecursive(data, patterns, ""), nil
}

func (rf *responseFilterer) excludeRecursive(data any, patterns []string, currentPath string) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			fieldPath := key
			if currentPath != "" {
				fieldPath = currentPath + "." + key
			}
			if !rf.shouldExclude(fieldPath, patterns) {
				result[key] = rf.excludeRecursive(value, patterns, fieldPath)
			}
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = rf.excludeRecursive(item, patterns, currentPath)
		}
		return result

	default:
		return data
	}
}

func (rf *responseFilterer) shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if rf.matchesExcludePattern(path, pattern) {
			return true
		}
	}
	return false
}

func (rf *responseFilterer) matchesExcludePattern(path, pattern string) bool {
	pathParts := strings.Split(path, ".")
	patternParts := parsePathPattern(pattern)
	return rf.matchPathSegments(pathParts, patternParts, 0, 0)
}

func (rf *responseFilterer) matchPathSegments(pathParts, patternParts []string, pathIdx, patternIdx int) bool {
	if patternIdx >= len(patternParts) {
		return pathIdx >= len(pathParts)
	}

	if pathIdx >= len(pathParts) {
		return false
	}

	patternPart := patternParts[patternIdx]

	if patternPart == "[]" {
		return rf.matchPathSegments(pathParts, patternParts, pathIdx, patternIdx+1)
	}

	if patternPart == "*" {
		if patternIdx == len(patternParts)-1 {
			return true
		}
		return rf.matchPathSegments(pathParts, patternParts, pathIdx+1, patternIdx+1)
	}

	if pathParts[pathIdx] == patternPart {
		return rf.matchPathSegments(pathParts, patternParts, pathIdx+1, patternIdx+1)
	}

	return false
}

type pathNode struct {
	children   map[string]*pathNode
	isTerminal bool
}

func newPathNode() *pathNode {
	return &pathNode{
		children: make(map[string]*pathNode),
	}
}

func buildIncludeTree(fields []string) *pathNode {
	root := newPathNode()

	for _, field := range fields {
		parts := parsePathPattern(field)
		current := root

		for i, part := range parts {
			if _, exists := current.children[part]; !exists {
				current.children[part] = newPathNode()
			}
			current = current.children[part]

			if i == len(parts)-1 {
				current.isTerminal = true
			}
		}
	}

	return root
}

func parsePathPattern(pattern string) []string {
	if pattern == "" {
		return []string{}
	}

	var parts []string
	var current strings.Builder

	i := 0
	for i < len(pattern) {
		if i+1 < len(pattern) && pattern[i:i+2] == "[]" {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			parts = append(parts, "[]")
			i += 2
			if i < len(pattern) && pattern[i] == '.' {
				i++
			}
		} else if pattern[i] == '.' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			i++
		} else {
			current.WriteByte(pattern[i])
			i++
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
