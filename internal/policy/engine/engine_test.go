package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/abac/proxy/internal/api"
)

type mockApi struct {
	data *api.PolicyGroup
	err  error
}

func (m *mockApi) GetPolicyData(_ context.Context, _ string) (*api.PolicyGroup, error) {
	return m.data, m.err
}
func (m *mockApi) GetAllowedHosts() []api.HostEntry { return nil }
func (m *mockApi) Invalidate(_ string)               {}
func (m *mockApi) InvalidateAll()                     {}

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

func (m *mockFilterer) Apply(data any, _ api.ResponseFilter) (any, error) {
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
		host    string
		wantErr bool
	}{
		{
			"resolves policy by host",
			&mockApi{data: &api.PolicyGroup{
				Policies: []api.Policy{
					{BaseURL: "https://api.example.com", UpstreamToken: "tok"},
				},
			}},
			"api.example.com",
			false,
		},
		{
			"api returns error",
			&mockApi{err: fmt.Errorf("db down")},
			"api.example.com",
			true,
		},
		{
			"host not found in group",
			&mockApi{data: &api.PolicyGroup{
				Policies: []api.Policy{
					{BaseURL: "https://other.com"},
				},
			}},
			"api.example.com",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.api, &mockMatcher{}, &mockFilterer{})
			got, err := e.GetPolicyData(context.Background(), "token", tt.host)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetPolicyData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}

func TestGetPolicyData_ResolvesUpstreamToken(t *testing.T) {
	a := &mockApi{data: &api.PolicyGroup{
		Policies: []api.Policy{
			{BaseURL: "https://host-a.com", UpstreamToken: "token-a"},
			{BaseURL: "https://host-b.com", UpstreamToken: "token-b"},
		},
	}}

	e := New(a, &mockMatcher{}, &mockFilterer{})

	got, err := e.GetPolicyData(context.Background(), "tok", "host-b.com")
	if err != nil {
		t.Fatalf("GetPolicyData() error = %v", err)
	}
	if got.Policy.UpstreamToken != "token-b" {
		t.Errorf("UpstreamToken = %q, want %q", got.Policy.UpstreamToken, "token-b")
	}
	if got.Policy.BaseURL != "https://host-b.com" {
		t.Errorf("BaseURL = %q, want %q", got.Policy.BaseURL, "https://host-b.com")
	}
}

func TestGetPolicyData_DefaultAction(t *testing.T) {
	a := &mockApi{data: &api.PolicyGroup{
		Policies: []api.Policy{
			{BaseURL: "https://api.example.com"},
		},
		DefaultAction: "allow",
	}}

	e := New(a, &mockMatcher{}, &mockFilterer{})
	got, err := e.GetPolicyData(context.Background(), "tok", "api.example.com")
	if err != nil {
		t.Fatalf("GetPolicyData() error = %v", err)
	}
	if got.DefaultAction != "allow" {
		t.Errorf("DefaultAction = %q, want %q", got.DefaultAction, "allow")
	}
}

func TestFindMatchingRule(t *testing.T) {
	rules := []api.PolicyRule{
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

func TestApplyFilter(t *testing.T) {
	data := map[string]any{"id": 1.0, "name": "alice"}
	f := api.ResponseFilter{Type: api.FilterTypeInclude, Fields: []string{"id"}}

	e := New(&mockApi{}, &mockMatcher{}, &mockFilterer{})
	got, err := e.ApplyFilter(data, f)
	if err != nil {
		t.Fatalf("ApplyFilter() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
}
