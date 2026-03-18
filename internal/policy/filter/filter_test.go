package filter

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/abac/proxy/internal/policy"
)

func TestApplyInclude(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		fields []string
		want   any
	}{
		{
			"simple fields",
			map[string]any{"id": 1.0, "name": "alice", "secret": "x"},
			[]string{"id", "name"},
			map[string]any{"id": 1.0, "name": "alice"},
		},
		{
			"nested field",
			map[string]any{
				"user": map[string]any{"name": "alice", "email": "a@b.com"},
				"other": "x",
			},
			[]string{"user.name"},
			map[string]any{
				"user": map[string]any{"name": "alice"},
			},
		},
		{
			"array field",
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
		},
		{
			"wildcard field",
			map[string]any{
				"a": map[string]any{"name": "x", "val": 1.0},
				"b": map[string]any{"name": "y", "val": 2.0},
			},
			[]string{"*.name"},
			map[string]any{
				"a": map[string]any{"name": "x"},
				"b": map[string]any{"name": "y"},
			},
		},
		{
			"empty fields returns nil",
			map[string]any{"id": 1.0},
			[]string{},
			nil,
		},
	}

	f := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Apply(tt.data, policy.ResponseFilter{
				Type:   policy.FilterTypeInclude,
				Fields: tt.fields,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyExclude(t *testing.T) {
	tests := []struct {
		name   string
		data   any
		fields []string
		want   any
	}{
		{
			"simple exclude",
			map[string]any{"id": 1.0, "secret": "x", "name": "alice"},
			[]string{"secret"},
			map[string]any{"id": 1.0, "name": "alice"},
		},
		{
			"nested exclude",
			map[string]any{
				"user": map[string]any{"name": "alice", "password": "hash"},
			},
			[]string{"user.password"},
			map[string]any{
				"user": map[string]any{"name": "alice"},
			},
		},
		{
			"wildcard exclude",
			map[string]any{
				"a": map[string]any{"name": "x", "secret": "s1"},
				"b": map[string]any{"name": "y", "secret": "s2"},
			},
			[]string{"*.secret"},
			map[string]any{
				"a": map[string]any{"name": "x"},
				"b": map[string]any{"name": "y"},
			},
		},
	}

	f := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Apply(tt.data, policy.ResponseFilter{
				Type:   policy.FilterTypeExclude,
				Fields: tt.fields,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		filter  policy.ResponseFilter
		wantErr bool
	}{
		{
			"valid JSON",
			`{"id":1,"name":"alice","secret":"x"}`,
			policy.ResponseFilter{Type: policy.FilterTypeInclude, Fields: []string{"id", "name"}},
			false,
		},
		{
			"invalid JSON",
			`{not json}`,
			policy.ResponseFilter{Type: policy.FilterTypeInclude, Fields: []string{"id"}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterJSON([]byte(tt.json), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FilterJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Fatal("expected non-nil result")
			}
			if !tt.wantErr {
				var parsed map[string]any
				if err := json.Unmarshal(result, &parsed); err != nil {
					t.Fatalf("result is not valid JSON: %v", err)
				}
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
		{"dots", "user.name", []string{"user", "name"}},
		{"array notation", "items[].id", []string{"items", "[]", "id"}},
		{"wildcard", "*.name", []string{"*", "name"}},
		{"empty", "", []string{}},
		{"nested array", "data[].items[].name", []string{"data", "[]", "items", "[]", "name"}},
		{"simple field", "id", []string{"id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePathPattern(tt.pattern)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePathPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}
