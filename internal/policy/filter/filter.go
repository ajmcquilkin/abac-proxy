package filter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abac/proxy/internal/policy"
)

type Filterer interface {
	Apply(data interface{}, filter policy.ResponseFilter) (interface{}, error)
}

type ResponseFilterer struct{}

var _ Filterer = (*ResponseFilterer)(nil)

func New() *ResponseFilterer {
	return &ResponseFilterer{}
}

func (rf *ResponseFilterer) Apply(data interface{}, f policy.ResponseFilter) (interface{}, error) {
	if f.Type == policy.FilterTypeInclude {
		return rf.applyInclude(data, f.Fields)
	}
	return rf.applyExclude(data, f.Fields)
}

func FilterJSON(jsonData []byte, f policy.ResponseFilter) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	filterer := New()
	filtered, err := filterer.Apply(data, f)
	if err != nil {
		return nil, fmt.Errorf("failed to apply filter: %w", err)
	}

	result, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered JSON: %w", err)
	}

	return result, nil
}

func (rf *ResponseFilterer) applyInclude(data interface{}, fields []string) (interface{}, error) {
	tree := buildIncludeTree(fields)
	return rf.filterWithTree(data, tree, "")
}

func (rf *ResponseFilterer) filterWithTree(data interface{}, node *pathNode, currentPath string) (interface{}, error) {
	if node == nil {
		return nil, nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})

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

	case []interface{}:
		if arrayChild, exists := node.children["[]"]; exists {
			result := make([]interface{}, 0)
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
			result := make([]interface{}, 0)
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

func (rf *ResponseFilterer) applyExclude(data interface{}, fields []string) (interface{}, error) {
	patterns := make([]string, len(fields))
	copy(patterns, fields)
	return rf.excludeRecursive(data, patterns, ""), nil
}

func (rf *ResponseFilterer) excludeRecursive(data interface{}, patterns []string, currentPath string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
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

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			itemPath := fmt.Sprintf("[%d]", i)
			if currentPath != "" {
				itemPath = currentPath + itemPath
			}
			result[i] = rf.excludeRecursive(item, patterns, itemPath)
		}
		return result

	default:
		return data
	}
}

func (rf *ResponseFilterer) shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if rf.matchesExcludePattern(path, pattern) {
			return true
		}
	}
	return false
}

func (rf *ResponseFilterer) matchesExcludePattern(path, pattern string) bool {
	pathParts := strings.Split(path, ".")
	patternParts := ParsePathPattern(pattern)
	return rf.matchPathSegments(pathParts, patternParts, 0, 0)
}

func (rf *ResponseFilterer) matchPathSegments(pathParts, patternParts []string, pathIdx, patternIdx int) bool {
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
		parts := ParsePathPattern(field)
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

func ParsePathPattern(pattern string) []string {
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
