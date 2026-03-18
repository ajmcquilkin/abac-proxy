package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/abac/proxy/internal/api"
	"github.com/abac/proxy/internal/policy"
)

type mockApi struct {
	data *api.PolicyData
	err  error
}

func (m *mockApi) GetPolicyData(_ context.Context, _ string) (*api.PolicyData, error) {
	return m.data, m.err
}
func (m *mockApi) Invalidate(_ string)  {}
func (m *mockApi) InvalidateAll()       {}

type mockMatcher struct {
	results []bool
	idx     int
}

func (m *mockMatcher) Matches(_, _ string) bool { return false }
func (m *mockMatcher) MatchesWithMethod(_, _, _, _ string) bool {
	if m.idx >= len(m.results) {
		return false
	}
	r := m.results[m.idx]
	m.idx++
	return r
}

type mockFilterer struct {
	data any
	err  error
}

func (m *mockFilterer) Apply(data any, _ policy.ResponseFilter) (any, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.data != nil {
		return m.data, nil
	}
	return data, nil
}

func TestGetPolicyData(t *testing.T) {
	tests := []struct {
		name    string
		api     *mockApi
		wantErr bool
	}{
		{
			"api returns data",
			&mockApi{data: &api.PolicyData{
				Policy:        &policy.Policy{DefaultAction: "allow"},
				UpstreamToken: "tok",
			}},
			false,
		},
		{
			"api returns error",
			&mockApi{err: fmt.Errorf("db down")},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.api, &mockMatcher{}, &mockFilterer{})
			got, err := e.GetPolicyData(context.Background(), "token")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetPolicyData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestFindMatchingRule(t *testing.T) {
	rules := []policy.PolicyRule{
		{Route: "/api/users", Method: "GET", Action: "allow"},
		{Route: "/api/posts", Method: "POST", Action: "deny"},
	}

	tests := []struct {
		name       string
		results    []bool
		wantFound  bool
		wantAction string
	}{
		{"match found (first rule)", []bool{true, false}, true, "allow"},
		{"match found (second rule)", []bool{false, true}, true, "deny"},
		{"no match", []bool{false, false}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockMatcher{results: tt.results}
			e := New(&mockApi{}, m, &mockFilterer{})
			rule, found := e.FindMatchingRule(rules, "/path", "GET")
			if found != tt.wantFound {
				t.Fatalf("FindMatchingRule() found = %v, want %v", found, tt.wantFound)
			}
			if found && rule.Action != tt.wantAction {
				t.Errorf("got action %q, want %q", rule.Action, tt.wantAction)
			}
		})
	}
}

func TestGetDefaultAction(t *testing.T) {
	e := New(&mockApi{}, &mockMatcher{}, &mockFilterer{})

	p := &policy.Policy{DefaultAction: "deny"}
	if got := e.GetDefaultAction(p); got != "deny" {
		t.Errorf("GetDefaultAction() = %q, want %q", got, "deny")
	}

	p2 := &policy.Policy{DefaultAction: "allow"}
	if got := e.GetDefaultAction(p2); got != "allow" {
		t.Errorf("GetDefaultAction() = %q, want %q", got, "allow")
	}
}

func TestApplyFilter(t *testing.T) {
	data := map[string]any{"id": 1.0, "name": "alice"}
	f := policy.ResponseFilter{Type: policy.FilterTypeInclude, Fields: []string{"id"}}

	e := New(&mockApi{}, &mockMatcher{}, &mockFilterer{})
	got, err := e.ApplyFilter(data, f)
	if err != nil {
		t.Fatalf("ApplyFilter() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
}
