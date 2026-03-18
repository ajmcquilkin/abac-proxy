package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileApi_GetPolicyData(t *testing.T) {
	t.Setenv("TEST_UPSTREAM_TOKEN", "upstream-token")

	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	writeTestPolicyGroup(t, path)

	fa, err := New([]string{path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, err := fa.GetPolicyData(context.Background(), "test-local-token")
	if err != nil {
		t.Fatalf("GetPolicyData() error = %v", err)
	}
	if data == nil || len(data.Policies) == 0 {
		t.Fatal("expected non-empty policies")
	}
	if data.Policies[0].BaseURL != "https://api.example.com" {
		t.Errorf("got baseURL %q, want %q", data.Policies[0].BaseURL, "https://api.example.com")
	}
	if data.Policies[0].UpstreamToken != "upstream-token" {
		t.Errorf("got UpstreamToken %q, want %q", data.Policies[0].UpstreamToken, "upstream-token")
	}
}

func TestFileApi_TokenNotFound(t *testing.T) {
	t.Setenv("TEST_UPSTREAM_TOKEN", "upstream-token")

	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	writeTestPolicyGroup(t, path)

	fa, err := New([]string{path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = fa.GetPolicyData(context.Background(), "unknown-token")
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestFileApi_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	path1 := filepath.Join(dir, "a.policygroup.json")
	os.WriteFile(path1, []byte(`{
		"version": "1.0",
		"localToken": "token-a",
		"policies": [{"baseUrl": "https://host-a.com", "rules": [{"route": "/a", "method": "GET", "action": "allow"}]}]
	}`), 0644)

	path2 := filepath.Join(dir, "b.policygroup.json")
	os.WriteFile(path2, []byte(`{
		"version": "1.0",
		"localToken": "token-b",
		"policies": [{"baseUrl": "https://host-b.com", "rules": [{"route": "/b", "method": "GET", "action": "allow"}]}]
	}`), 0644)

	fa, err := New([]string{path1, path2})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, err := fa.GetPolicyData(context.Background(), "token-a")
	if err != nil {
		t.Fatalf("GetPolicyData(token-a) error = %v", err)
	}
	if data.Policies[0].BaseURL != "https://host-a.com" {
		t.Errorf("got %q, want %q", data.Policies[0].BaseURL, "https://host-a.com")
	}

	data, err = fa.GetPolicyData(context.Background(), "token-b")
	if err != nil {
		t.Fatalf("GetPolicyData(token-b) error = %v", err)
	}
	if data.Policies[0].BaseURL != "https://host-b.com" {
		t.Errorf("got %q, want %q", data.Policies[0].BaseURL, "https://host-b.com")
	}

	hosts := fa.GetAllowedHosts()
	if len(hosts) != 2 {
		t.Errorf("expected 2 allowed hosts, got %d", len(hosts))
	}
}

func TestFileApi_DuplicateToken(t *testing.T) {
	dir := t.TempDir()

	data := []byte(`{
		"version": "1.0",
		"localToken": "same-token",
		"policies": [{"baseUrl": "https://host.com", "rules": [{"route": "/a", "method": "GET", "action": "allow"}]}]
	}`)

	path1 := filepath.Join(dir, "a.policygroup.json")
	path2 := filepath.Join(dir, "b.policygroup.json")
	os.WriteFile(path1, data, 0644)
	os.WriteFile(path2, data, 0644)

	_, err := New([]string{path1, path2})
	if err == nil {
		t.Fatal("expected error for duplicate localToken")
	}
}

func TestNew_Errors(t *testing.T) {
	t.Setenv("TEST_UPSTREAM_TOKEN", "upstream-token")

	tests := []struct {
		name    string
		setup   func(string) string
		wantErr bool
	}{
		{
			"valid file",
			func(dir string) string {
				p := filepath.Join(dir, "good.policygroup.json")
				writeTestPolicyGroup(t, p)
				return p
			},
			false,
		},
		{
			"not found",
			func(_ string) string { return "/nonexistent/policy.policygroup.json" },
			true,
		},
		{
			"invalid JSON",
			func(dir string) string {
				p := filepath.Join(dir, "bad.policygroup.json")
				os.WriteFile(p, []byte("{not json}"), 0644)
				return p
			},
			true,
		},
		{
			"invalid policy group (missing version)",
			func(dir string) string {
				p := filepath.Join(dir, "invalid.policygroup.json")
				os.WriteFile(p, []byte(`{
					"localToken": "tok",
					"policies": [{"baseUrl": "https://example.com", "rules": []}]
				}`), 0644)
				return p
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(dir)
			_, err := New([]string{path})
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileApi_EnvVarResolved(t *testing.T) {
	t.Setenv("MY_TEST_TOKEN", "resolved-secret-value")

	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	os.WriteFile(path, []byte(`{
		"version": "1.0",
		"localToken": "tok",
		"policies": [{
			"baseUrl": "https://api.example.com",
			"localUpstreamTokenKey": "MY_TEST_TOKEN",
			"rules": []
		}]
	}`), 0644)

	fa, err := New([]string{path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, _ := fa.GetPolicyData(context.Background(), "tok")
	if data.Policies[0].UpstreamToken != "resolved-secret-value" {
		t.Errorf("got %q, want %q", data.Policies[0].UpstreamToken, "resolved-secret-value")
	}
}

func TestFileApi_MissingEnvVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	os.WriteFile(path, []byte(`{
		"version": "1.0",
		"localToken": "tok",
		"policies": [{
			"baseUrl": "https://api.example.com",
			"localUpstreamTokenKey": "NONEXISTENT_VAR_12345",
			"rules": []
		}]
	}`), 0644)

	_, err := New([]string{path})
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestFileApi_NoUpstreamTokenKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	os.WriteFile(path, []byte(`{
		"version": "1.0",
		"localToken": "tok",
		"policies": [{
			"baseUrl": "https://api.example.com",
			"rules": []
		}]
	}`), 0644)

	fa, err := New([]string{path})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data, _ := fa.GetPolicyData(context.Background(), "tok")
	if data.Policies[0].UpstreamToken != "" {
		t.Errorf("expected empty UpstreamToken, got %q", data.Policies[0].UpstreamToken)
	}
}

func writeTestPolicyGroup(t *testing.T, path string) {
	t.Helper()
	data := []byte(`{
		"version": "1.0",
		"localToken": "test-local-token",
		"policies": [{
			"baseUrl": "https://api.example.com",
			"localUpstreamTokenKey": "TEST_UPSTREAM_TOKEN",
			"rules": []
		}]
	}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test policy group: %v", err)
	}
}
