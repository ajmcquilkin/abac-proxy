package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abac/proxy/internal/db"
	"github.com/abac/proxy/internal/policy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockQuerier struct {
	getDownstreamResult db.GetDownstreamTokenByHashRow
	getDownstreamErr    error
	calls               int
}

func (m *mockQuerier) GetDownstreamTokenByHash(_ context.Context, _ string) (db.GetDownstreamTokenByHashRow, error) {
	m.calls++
	return m.getDownstreamResult, m.getDownstreamErr
}

func (m *mockQuerier) ActivatePolicy(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) CreateDownstreamToken(_ context.Context, _ db.CreateDownstreamTokenParams) (db.DownstreamToken, error) {
	return db.DownstreamToken{}, nil
}
func (m *mockQuerier) CreatePolicy(_ context.Context, _ db.CreatePolicyParams) (db.Policy, error) {
	return db.Policy{}, nil
}
func (m *mockQuerier) CreateUpstreamCredential(_ context.Context, _ db.CreateUpstreamCredentialParams) (db.UpstreamCredential, error) {
	return db.UpstreamCredential{}, nil
}
func (m *mockQuerier) CreateUser(_ context.Context, _ string) (db.User, error) {
	return db.User{}, nil
}
func (m *mockQuerier) DeactivateUserPolicies(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) DeleteDownstreamToken(_ context.Context, _ pgtype.UUID) error  { return nil }
func (m *mockQuerier) DeleteUpstreamCredential(_ context.Context, _ pgtype.UUID) error {
	return nil
}
func (m *mockQuerier) GetActivePolicyForUser(_ context.Context, _ pgtype.UUID) (db.Policy, error) {
	return db.Policy{}, nil
}
func (m *mockQuerier) GetUpstreamCredentialByID(_ context.Context, _ pgtype.UUID) (db.UpstreamCredential, error) {
	return db.UpstreamCredential{}, nil
}
func (m *mockQuerier) GetUserByEmail(_ context.Context, _ string) (db.User, error) {
	return db.User{}, nil
}
func (m *mockQuerier) GetUserByID(_ context.Context, _ pgtype.UUID) (db.User, error) {
	return db.User{}, nil
}
func (m *mockQuerier) ListActiveUpstreamCredentials(_ context.Context, _ pgtype.UUID) ([]db.UpstreamCredential, error) {
	return nil, nil
}
func (m *mockQuerier) ListDownstreamTokensByPolicyID(_ context.Context, _ pgtype.UUID) ([]db.DownstreamToken, error) {
	return nil, nil
}
func (m *mockQuerier) ListPolicyVersions(_ context.Context, _ pgtype.UUID) ([]db.Policy, error) {
	return nil, nil
}
func (m *mockQuerier) ListUpstreamCredentialsByUserID(_ context.Context, _ pgtype.UUID) ([]db.UpstreamCredential, error) {
	return nil, nil
}
func (m *mockQuerier) RevokeDownstreamToken(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) UpdateDownstreamTokenLastUsed(_ context.Context, _ pgtype.UUID) error {
	return nil
}
func (m *mockQuerier) UpdateUpstreamCredential(_ context.Context, _ db.UpdateUpstreamCredentialParams) (db.UpstreamCredential, error) {
	return db.UpstreamCredential{}, nil
}

func plainHasher(token string) (string, error) { return token, nil }
func plainValidator(token, hash string) bool    { return token == hash }

func TestFileApi_GetPolicyData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	writeTestPolicyGroup(t, path)

	fa, err := NewFileApi([]string{path})
	if err != nil {
		t.Fatalf("NewFileApi() error = %v", err)
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
}

func TestFileApi_TokenNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.policygroup.json")
	writeTestPolicyGroup(t, path)

	fa, err := NewFileApi([]string{path})
	if err != nil {
		t.Fatalf("NewFileApi() error = %v", err)
	}

	_, err = fa.GetPolicyData(context.Background(), "unknown-token")
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}

func TestFileApi_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	pg1 := policy.PolicyGroup{
		Version:    "1.0",
		LocalToken: "token-a",
		Policies: []policy.Policy{{
			BaseURL: "https://host-a.com",
			Rules:   []policy.PolicyRule{{Route: "/a", Method: "GET", Action: "allow"}},
		}},
	}
	pg2 := policy.PolicyGroup{
		Version:    "1.0",
		LocalToken: "token-b",
		Policies: []policy.Policy{{
			BaseURL: "https://host-b.com",
			Rules:   []policy.PolicyRule{{Route: "/b", Method: "GET", Action: "allow"}},
		}},
	}

	path1 := filepath.Join(dir, "a.policygroup.json")
	path2 := filepath.Join(dir, "b.policygroup.json")
	writePolicyGroupFile(t, path1, pg1)
	writePolicyGroupFile(t, path2, pg2)

	fa, err := NewFileApi([]string{path1, path2})
	if err != nil {
		t.Fatalf("NewFileApi() error = %v", err)
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
	pg := policy.PolicyGroup{
		Version:    "1.0",
		LocalToken: "same-token",
		Policies: []policy.Policy{{
			BaseURL: "https://host.com",
			Rules:   []policy.PolicyRule{{Route: "/a", Method: "GET", Action: "allow"}},
		}},
	}

	path1 := filepath.Join(dir, "a.policygroup.json")
	path2 := filepath.Join(dir, "b.policygroup.json")
	writePolicyGroupFile(t, path1, pg)
	writePolicyGroupFile(t, path2, pg)

	_, err := NewFileApi([]string{path1, path2})
	if err == nil {
		t.Fatal("expected error for duplicate localToken")
	}
}

func TestNewFileApi_Errors(t *testing.T) {
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
				data, _ := json.Marshal(map[string]any{
					"localToken": "tok",
					"policies": []any{
						map[string]any{"baseUrl": "https://example.com", "rules": []any{}},
					},
				})
				os.WriteFile(p, data, 0644)
				return p
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(dir)
			_, err := NewFileApi([]string{path})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileApi() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDBApi_CacheMiss(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := NewDBApi(mq, 15*time.Second, plainHasher, plainValidator)
	data, err := d.GetPolicyData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetPolicyData() error = %v", err)
	}
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	if mq.calls != 1 {
		t.Errorf("expected 1 DB call, got %d", mq.calls)
	}
}

func TestDBApi_CacheHit(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := NewDBApi(mq, 15*time.Second, plainHasher, plainValidator)

	_, err := d.GetPolicyData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	_, err = d.GetPolicyData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("second call error = %v", err)
	}

	if mq.calls != 1 {
		t.Errorf("expected 1 DB call (cached), got %d", mq.calls)
	}
}

func TestDBApi_CacheExpired(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := NewDBApi(mq, 1*time.Millisecond, plainHasher, plainValidator)

	_, err := d.GetPolicyData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	_, err = d.GetPolicyData(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("second call error = %v", err)
	}

	if mq.calls != 2 {
		t.Errorf("expected 2 DB calls (expired), got %d", mq.calls)
	}
}

func TestDBApi_QueryError(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamErr: pgx.ErrNoRows,
	}

	d := NewDBApi(mq, 15*time.Second, plainHasher, plainValidator)
	_, err := d.GetPolicyData(context.Background(), "test-token")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBApi_InvalidToken(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	alwaysFalse := func(_, _ string) bool { return false }
	d := NewDBApi(mq, 15*time.Second, plainHasher, alwaysFalse)

	_, err := d.GetPolicyData(context.Background(), "test-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestDBApi_Invalidate(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := NewDBApi(mq, 15*time.Second, plainHasher, plainValidator)

	_, _ = d.GetPolicyData(context.Background(), "test-token")
	if mq.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mq.calls)
	}

	d.Invalidate("test-token")

	_, _ = d.GetPolicyData(context.Background(), "test-token")
	if mq.calls != 2 {
		t.Errorf("expected 2 calls after invalidate, got %d", mq.calls)
	}
}

func TestDBApi_InvalidateAll(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := NewDBApi(mq, 15*time.Second, plainHasher, plainValidator)

	_, _ = d.GetPolicyData(context.Background(), "token-a")
	_, _ = d.GetPolicyData(context.Background(), "token-b")
	initialCalls := mq.calls

	d.InvalidateAll()

	_, _ = d.GetPolicyData(context.Background(), "token-a")
	_, _ = d.GetPolicyData(context.Background(), "token-b")

	if mq.calls != initialCalls+2 {
		t.Errorf("expected %d calls after InvalidateAll, got %d", initialCalls+2, mq.calls)
	}
}

func writeTestPolicyGroup(t *testing.T, path string) {
	t.Helper()
	pg := policy.PolicyGroup{
		Version:    "1.0",
		LocalToken: "test-local-token",
		Policies: []policy.Policy{{
			BaseURL:       "https://api.example.com",
			UpstreamToken: "upstream-token",
			Rules:         []policy.PolicyRule{},
		}},
	}
	writePolicyGroupFile(t, path, pg)
}

func writePolicyGroupFile(t *testing.T, path string, pg policy.PolicyGroup) {
	t.Helper()
	data, _ := json.Marshal(pg)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test policy group: %v", err)
	}
}

func makeTestDBResult() db.GetDownstreamTokenByHashRow {
	rules, _ := json.Marshal([]any{})
	userID := pgtype.UUID{Valid: true}

	return db.GetDownstreamTokenByHashRow{
		ID:        pgtype.UUID{Valid: true},
		TokenHash: "test-token",
		Policy: db.Policy{
			Version:       "1.0",
			BaseUrl:       "https://api.example.com",
			DefaultAction: "allow",
			Rules:         rules,
			UserID:        userID,
		},
		UpstreamCredential: db.UpstreamCredential{
			Token: "upstream-tok",
		},
	}
}
