package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abac/proxy/internal/proxy/allowlist"
	"github.com/abac/proxy/internal/proxy/interceptor"
)

type mockChecker struct {
	scheme string
	found  bool
	hosts  []string
}

var _ allowlist.Allowlist = (*mockChecker)(nil)

func (m *mockChecker) FindHost(_ string) (string, bool) { return m.scheme, m.found }
func (m *mockChecker) IsAllowed(_ string) bool          { return m.found }
func (m *mockChecker) GetHostList() []string            { return m.hosts }

type mockInterceptor struct {
	reqFn  func(*http.Request) *http.Request
	respFn func(*http.Response) error
}

var _ interceptor.Interceptor = (*mockInterceptor)(nil)

func (m *mockInterceptor) InterceptRequest(req *http.Request) *http.Request {
	if m.reqFn != nil {
		return m.reqFn(req)
	}
	return req
}

func (m *mockInterceptor) InterceptResponse(resp *http.Response) error {
	if m.respFn != nil {
		return m.respFn(resp)
	}
	return nil
}

func TestServeHTTP_HostNotAllowed(t *testing.T) {
	srv := New(
		&mockChecker{found: false},
		&mockInterceptor{},
	)

	req := httptest.NewRequest("GET", "http://evil.com/api", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServeHTTP_HostAllowed(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	intercepted := false
	srv := New(
		&mockChecker{scheme: "http", found: true},
		&mockInterceptor{
			reqFn: func(req *http.Request) *http.Request {
				intercepted = true
				return req
			},
		},
	)

	req := httptest.NewRequest("GET", backend.URL+"/api", nil)
	req.Host = "127.0.0.1"
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if !intercepted {
		t.Error("expected interceptor to be called")
	}
}

func TestNew_NilInterceptorPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil interceptor")
		}
	}()
	New(&mockChecker{found: true}, nil)
}

func TestNew_NilHostsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil hosts")
		}
	}()
	New(nil, &mockInterceptor{})
}
