package filter

import (
	"reflect"
	"testing"

	"github.com/abac/proxy/internal/api"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name        string
		data        any
		fields      []string
		wantInclude any
		wantExclude any
	}{
		{
			"single root field",
			map[string]any{"name": "alice", "age": 30.0, "secret": "x"},
			[]string{"name"},
			map[string]any{"name": "alice"},
			map[string]any{"age": 30.0, "secret": "x"},
		},
		{
			"multiple root fields",
			map[string]any{"id": 1.0, "name": "alice", "secret": "x"},
			[]string{"id", "name"},
			map[string]any{"id": 1.0, "name": "alice"},
			map[string]any{"secret": "x"},
		},
		{
			"nested object field",
			map[string]any{
				"address": map[string]any{"city": "NYC", "zip": "10001"},
				"name":    "alice",
			},
			[]string{"address.city"},
			map[string]any{
				"address": map[string]any{"city": "NYC"},
			},
			map[string]any{
				"address": map[string]any{"zip": "10001"},
				"name":    "alice",
			},
		},
		{
			"nested wildcard",
			map[string]any{
				"address": map[string]any{
					"city": map[string]any{"name": "NYC", "pop": 8e6},
				},
				"id": 1.0,
			},
			[]string{"address.city.*"},
			map[string]any{
				"address": map[string]any{
					"city": map[string]any{"name": "NYC", "pop": 8e6},
				},
			},
			map[string]any{
				"address": map[string]any{
					"city": map[string]any{},
				},
				"id": 1.0,
			},
		},
		{
			"root array pick field",
			[]any{
				map[string]any{"name": "alice", "age": 30.0},
				map[string]any{"name": "bob", "age": 25.0},
			},
			[]string{"[].name"},
			[]any{
				map[string]any{"name": "alice"},
				map[string]any{"name": "bob"},
			},
			[]any{
				map[string]any{"age": 30.0},
				map[string]any{"age": 25.0},
			},
		},
		{
			"array within object",
			map[string]any{
				"items": []any{
					map[string]any{"id": 1.0, "name": "a"},
					map[string]any{"id": 2.0, "name": "b"},
				},
			},
			[]string{"items[].id"},
			map[string]any{
				"items": []any{
					map[string]any{"id": 1.0},
					map[string]any{"id": 2.0},
				},
			},
			map[string]any{
				"items": []any{
					map[string]any{"name": "a"},
					map[string]any{"name": "b"},
				},
			},
		},
		{
			"root array nested array",
			[]any{
				map[string]any{
					"id": 1.0,
					"members": []any{
						map[string]any{"email": "a@b.com", "role": "admin"},
						map[string]any{"email": "c@d.com", "role": "user"},
					},
				},
				map[string]any{
					"id": 2.0,
					"members": []any{
						map[string]any{"email": "e@f.com", "role": "owner"},
					},
				},
			},
			[]string{"[].members[].email"},
			[]any{
				map[string]any{
					"members": []any{
						map[string]any{"email": "a@b.com"},
						map[string]any{"email": "c@d.com"},
					},
				},
				map[string]any{
					"members": []any{
						map[string]any{"email": "e@f.com"},
					},
				},
			},
			[]any{
				map[string]any{
					"id": 1.0,
					"members": []any{
						map[string]any{"role": "admin"},
						map[string]any{"role": "user"},
					},
				},
				map[string]any{
					"id": 2.0,
					"members": []any{
						map[string]any{"role": "owner"},
					},
				},
			},
		},
		{
			"wildcard at root",
			map[string]any{
				"a": map[string]any{"name": "x", "val": 1.0},
				"b": map[string]any{"name": "y", "val": 2.0},
			},
			[]string{"*.name"},
			map[string]any{
				"a": map[string]any{"name": "x"},
				"b": map[string]any{"name": "y"},
			},
			map[string]any{
				"a": map[string]any{"val": 1.0},
				"b": map[string]any{"val": 2.0},
			},
		},
		{
			"empty fields",
			map[string]any{"id": 1.0},
			[]string{},
			nil,
			map[string]any{"id": 1.0},
		},
		{
			"no matching include fields",
			map[string]any{"id": 1.0, "name": "alice"},
			[]string{"missing"},
			nil,
			map[string]any{"id": 1.0, "name": "alice"},
		},
		{
			"exclude all fields",
			map[string]any{"id": 1.0},
			[]string{"id"},
			map[string]any{"id": 1.0},
			map[string]any{},
		},
	}

	f := New()
	for _, tt := range tests {
		t.Run(tt.name+"/include", func(t *testing.T) {
			got, err := f.Apply(tt.data, api.ResponseFilter{
				Type:   api.FilterTypeInclude,
				Fields: tt.fields,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantInclude) {
				t.Errorf("got %v, want %v", got, tt.wantInclude)
			}
		})
		t.Run(tt.name+"/exclude", func(t *testing.T) {
			got, err := f.Apply(tt.data, api.ResponseFilter{
				Type:   api.FilterTypeExclude,
				Fields: tt.fields,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantExclude) {
				t.Errorf("got %v, want %v", got, tt.wantExclude)
			}
		})
	}
}

func TestParsePathPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{"simple field", "id", []string{"id"}},
		{"dots", "user.name", []string{"user", "name"}},
		{"deep nested", "address.city.name", []string{"address", "city", "name"}},
		{"array notation", "items[].id", []string{"items", "[]", "id"}},
		{"wildcard", "*.name", []string{"*", "name"}},
		{"nested wildcard", "address.city.*", []string{"address", "city", "*"}},
		{"root array field", "[].name", []string{"[]", "name"}},
		{"nested array", "data[].items[].name", []string{"data", "[]", "items", "[]", "name"}},
		{"root array nested array", "[].members[].email", []string{"[]", "members", "[]", "email"}},
		{"empty", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePathPattern(tt.pattern)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePathPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}
