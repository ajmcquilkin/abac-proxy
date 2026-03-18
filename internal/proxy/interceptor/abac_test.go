package interceptor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/abac/proxy/internal/api"
	"github.com/abac/proxy/internal/policy"
	"github.com/abac/proxy/internal/policy/engine"
)

type mockEngine struct {
	policyData *api.PolicyData
	policyErr  error
	matchRule  *policy.PolicyRule
	matchFound bool
	filterData any
	filterErr  error
}

var _ engine.Engine = (*mockEngine)(nil)

func (m *mockEngine) GetPolicyData(_ context.Context, _, _ string) (*api.PolicyData, error) {
	return m.policyData, m.policyErr
}
func (m *mockEngine) FindMatchingRule(_ []policy.PolicyRule, _, _ string) (*policy.PolicyRule, bool) {
	return m.matchRule, m.matchFound
}
func (m *mockEngine) ApplyFilter(data any, _ policy.ResponseFilter) (any, error) {
	if m.filterErr != nil {
		return nil, m.filterErr
	}
	if m.filterData != nil {
		return m.filterData, nil
	}
	return data, nil
}

func newTestResponse(req *http.Request, statusCode int, body any) *http.Response {
	var bodyReader io.ReadCloser
	var contentType string
	var contentLength int64

	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = io.NopCloser(bytes.NewReader(b))
		contentType = "application/json"
		contentLength = int64(len(b))
	} else {
		bodyReader = io.NopCloser(bytes.NewReader(nil))
	}

	return &http.Response{
		StatusCode:    statusCode,
		Header:        http.Header{"Content-Type": []string{contentType}},
		Body:          bodyReader,
		ContentLength: contentLength,
		Request:       req,
	}
}

func readResponseBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	return result
}

func TestInterceptRequestResponse(t *testing.T) {
	upstreamType := "bearer"
	upstreamHeader := ""

	tests := []struct {
		name                   string
		engine                 *mockEngine
		passthroughUnspecified bool
		authHeader             string
		respBody               any
		wantStatus             int
		wantError              string
	}{
		{
			"valid token, allow, no filter",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy:               &policy.Policy{Rules: []policy.PolicyRule{{Route: "/api", Action: "allow"}}},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchRule:  &policy.PolicyRule{Action: "allow"},
				matchFound: true,
			},
			false,
			"Bearer valid-token",
			map[string]any{"id": 1.0, "name": "alice"},
			200,
			"",
		},
		{
			"invalid token returns 403",
			&mockEngine{
				policyErr: fmt.Errorf("invalid token"),
			},
			false,
			"Bearer bad-token",
			map[string]any{"id": 1.0},
			403,
			"invalid or missing token",
		},
		{
			"missing auth header returns 403",
			&mockEngine{
				policyErr: fmt.Errorf("empty token"),
			},
			false,
			"",
			map[string]any{"id": 1.0},
			403,
			"invalid or missing token",
		},
		{
			"deny action returns 403",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy:               &policy.Policy{Rules: []policy.PolicyRule{{Route: "/api", Action: "deny"}}},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchRule:  &policy.PolicyRule{Action: "deny"},
				matchFound: true,
			},
			false,
			"Bearer valid-token",
			map[string]any{"id": 1.0},
			403,
			"access denied by policy",
		},
		{
			"no match, no passthrough → deny",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy:               &policy.Policy{},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchFound: false,
			},
			false,
			"Bearer valid-token",
			map[string]any{"id": 1.0},
			403,
			"access denied by policy",
		},
		{
			"no match, passthrough → allow",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy:               &policy.Policy{},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchFound: false,
			},
			true,
			"Bearer valid-token",
			map[string]any{"id": 1.0},
			200,
			"",
		},
		{
			"no match, DefaultAction override from DB",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy:               &policy.Policy{},
					DefaultAction:        "deny",
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchFound: false,
			},
			true,
			"Bearer valid-token",
			map[string]any{"id": 1.0},
			403,
			"access denied by policy",
		},
		{
			"filter applied successfully",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy: &policy.Policy{Rules: []policy.PolicyRule{{
						Route:  "/api",
						Action: "allow",
						ResponseFilter: &policy.ResponseFilter{
							Type:   policy.FilterTypeInclude,
							Fields: []string{"id"},
						},
					}}},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchRule: &policy.PolicyRule{
					Action: "allow",
					ResponseFilter: &policy.ResponseFilter{
						Type:   policy.FilterTypeInclude,
						Fields: []string{"id"},
					},
				},
				matchFound: true,
				filterData: map[string]any{"id": 1.0},
			},
			false,
			"Bearer valid-token",
			map[string]any{"id": 1.0, "name": "alice", "secret": "hidden"},
			200,
			"",
		},
		{
			"filter error returns 500",
			&mockEngine{
				policyData: &api.PolicyData{
					Policy: &policy.Policy{Rules: []policy.PolicyRule{{
						Route:  "/api",
						Action: "allow",
						ResponseFilter: &policy.ResponseFilter{
							Type:   policy.FilterTypeInclude,
							Fields: []string{"id"},
						},
					}}},
					UpstreamToken:        "upstream-token-value-xxxxx",
					UpstreamTokenType:    &upstreamType,
					UpstreamHeaderString: &upstreamHeader,
				},
				matchRule: &policy.PolicyRule{
					Action: "allow",
					ResponseFilter: &policy.ResponseFilter{
						Type:   policy.FilterTypeInclude,
						Fields: []string{"id"},
					},
				},
				matchFound: true,
				filterErr:  fmt.Errorf("filter broke"),
			},
			false,
			"Bearer valid-token",
			map[string]any{"id": 1.0},
			500,
			"failed to filter response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := New(tt.engine, tt.passthroughUnspecified)

			req, _ := http.NewRequest("GET", "http://api.example.com/api/users", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			req = i.InterceptRequest(req)

			resp := newTestResponse(req, 200, tt.respBody)
			err := i.InterceptResponse(resp)
			if err != nil {
				t.Fatalf("InterceptResponse() returned error: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantError != "" {
				body := readResponseBody(t, resp)
				if errMsg, ok := body["error"].(string); !ok || errMsg != tt.wantError {
					t.Errorf("error = %q, want %q", errMsg, tt.wantError)
				}
			}
		})
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       string
	}{
		{"valid bearer", "Bearer my-token", "my-token"},
		{"case insensitive", "bearer my-token", "my-token"},
		{"empty header", "", ""},
		{"missing bearer prefix", "Basic abc123", ""},
		{"no space", "Bearertoken", ""},
		{"trims whitespace", "Bearer  my-token ", "my-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			got := extractToken(req)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReplaceWithError(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	replaceWithError(resp, 403, "access denied")

	if resp.StatusCode != 403 {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if body["error"] != "access denied" {
		t.Errorf("error = %q, want %q", body["error"], "access denied")
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", resp.Header.Get("Content-Type"))
	}
}

func TestReadAndReplaceBody(t *testing.T) {
	t.Run("transforms body", func(t *testing.T) {
		body := []byte(`{"original":true}`)
		resp := &http.Response{
			Header:        http.Header{},
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: int64(len(body)),
		}

		err := readAndReplaceBody(resp, func(b []byte) ([]byte, error) {
			return []byte(`{"modified":true}`), nil
		})
		if err != nil {
			t.Fatalf("readAndReplaceBody() error = %v", err)
		}

		got, _ := io.ReadAll(resp.Body)
		if string(got) != `{"modified":true}` {
			t.Errorf("body = %s, want %s", got, `{"modified":true}`)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		err := readAndReplaceBody(resp, func(b []byte) ([]byte, error) {
			return nil, fmt.Errorf("should not be called")
		})
		if err != nil {
			t.Fatalf("readAndReplaceBody(nil body) error = %v", err)
		}
	})

	t.Run("modifier error", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{},
			Body:   io.NopCloser(bytes.NewReader([]byte("data"))),
		}
		err := readAndReplaceBody(resp, func(b []byte) ([]byte, error) {
			return nil, fmt.Errorf("modifier failed")
		})
		if err == nil {
			t.Fatal("expected error from modifier")
		}
	})
}

func TestContextKeyPassing(t *testing.T) {
	upstreamType := "bearer"
	upstreamHeader := ""

	e := &mockEngine{
		policyData: &api.PolicyData{
			Policy:               &policy.Policy{},
			UpstreamToken:        "upstream-token-value-xxxxx",
			UpstreamTokenType:    &upstreamType,
			UpstreamHeaderString: &upstreamHeader,
		},
		matchFound: false,
	}

	i := New(e, true)
	req, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	req = i.InterceptRequest(req)

	ctx := req.Context()
	if v, _ := ctx.Value(ContextKeyTokenValid).(bool); !v {
		t.Error("expected ContextKeyTokenValid = true")
	}
	if v, _ := ctx.Value(ContextKeyRequestPath).(string); v != "/api/users" {
		t.Errorf("ContextKeyRequestPath = %q, want /api/users", v)
	}
	if v, _ := ctx.Value(ContextKeyRequestMethod).(string); v != "GET" {
		t.Errorf("ContextKeyRequestMethod = %q, want GET", v)
	}
	if v, _ := ctx.Value(ContextKeyUpstreamToken).(string); v != "upstream-token-value-xxxxx" {
		t.Errorf("ContextKeyUpstreamToken = %q, want upstream-token-value-xxxxx", v)
	}
}
