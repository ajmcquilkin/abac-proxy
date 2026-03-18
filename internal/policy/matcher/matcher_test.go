package matcher

import "testing"

func TestMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact match", "/api/users", "/api/users", true},
		{"wildcard segment", "/api/*/profile", "/api/123/profile", true},
		{"no match wrong segment", "/api/users", "/api/posts", false},
		{"no match wrong length", "/api/users/list", "/api/users", false},
		{"trailing slash normalized", "/api/users/", "/api/users", true},
		{"empty path becomes root", "", "/", true},
		{"no leading slash added", "api/users", "/api/users", true},
		{"root exact", "/", "/", true},
		{"wildcard at end", "/api/*", "/api/anything", true},
		{"wildcard mismatch length", "/api/*", "/api/users/123", false},
	}

	pm := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.Matches(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("Matches(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestMatchesWithMethod(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		method    string
		path      string
		reqMethod string
		want      bool
	}{
		{"method match", "/api/users", "GET", "/api/users", "GET", true},
		{"method mismatch", "/api/users", "POST", "/api/users", "GET", false},
		{"empty method matches all", "/api/users", "", "/api/users", "DELETE", true},
		{"case insensitive method", "/api/users", "get", "/api/users", "GET", true},
		{"path mismatch with method match", "/api/users", "GET", "/api/posts", "GET", false},
	}

	pm := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.MatchesWithMethod(tt.pattern, tt.method, tt.path, tt.reqMethod)
			if got != tt.want {
				t.Errorf("MatchesWithMethod(%q, %q, %q, %q) = %v, want %v",
					tt.pattern, tt.method, tt.path, tt.reqMethod, got, tt.want)
			}
		})
	}
}
