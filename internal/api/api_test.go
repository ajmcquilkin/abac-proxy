package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abac/proxy/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockQuerier implements db.Querier for testing
type mockQuerier struct {
	getDownstreamResult db.GetDownstreamTokenByHashRow
	getDownstreamErr    error
	calls               int
}

func (m *mockQuerier) GetDownstreamTokenByHash(_ context.Context, _ string) (db.GetDownstreamTokenByHashRow, error) {
	m.calls++
	return m.getDownstreamResult, m.getDownstreamErr
}

// Stub out all other Querier methods
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
	path := filepath.Join(dir, "policy.json")
	writeTestPolicy(t, path)

	fa, err := NewFileApi(path)
	if err != nil {
		t.Fatalf("NewFileApi() error = %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"returns pre-loaded data", "any-token"},
		{"token ignored", "different-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := fa.GetPolicyData(context.Background(), tt.token)
			if err != nil {
				t.Fatalf("GetPolicyData() error = %v", err)
			}
			if data == nil || data.Policy == nil {
				t.Fatal("expected non-nil policy data")
			}
			if data.Policy.DefaultAction != "deny" {
				t.Errorf("got default_action %q, want %q", data.Policy.DefaultAction, "deny")
			}
		})
	}
}

func TestNewFileApi(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) string
		wantErr bool
	}{
		{
			"valid file",
			func(dir string) string {
				p := filepath.Join(dir, "good.json")
				writeTestPolicy(t, p)
				return p
			},
			false,
		},
		{
			"not found",
			func(_ string) string { return "/nonexistent/policy.json" },
			true,
		},
		{
			"invalid JSON",
			func(dir string) string {
				p := filepath.Join(dir, "bad.json")
				os.WriteFile(p, []byte("{not json}"), 0644)
				return p
			},
			true,
		},
		{
			"invalid policy (missing version)",
			func(dir string) string {
				p := filepath.Join(dir, "invalid.json")
				data, _ := json.Marshal(map[string]interface{}{
					"user":           map[string]string{"token": "t", "id": "1"},
					"baseUrl":        "http://example.com",
					"policies":       []interface{}{},
					"default_action": "allow",
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
			_, err := NewFileApi(path)
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

func writeTestPolicy(t *testing.T, path string) {
	t.Helper()
	p := map[string]interface{}{
		"version":        "1.0",
		"user":           map[string]string{"token": "upstream-token", "id": "user-1"},
		"baseUrl":        "https://api.example.com",
		"policies":       []interface{}{},
		"default_action": "deny",
	}
	data, _ := json.Marshal(p)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test policy: %v", err)
	}
}

func makeTestDBResult() db.GetDownstreamTokenByHashRow {
	rules, _ := json.Marshal([]interface{}{})
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
