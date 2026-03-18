package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/abac/proxy/internal/api"
	dbstore "github.com/abac/proxy/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockQuerier struct {
	getDownstreamResult dbstore.GetDownstreamTokenByHashRow
	getDownstreamErr    error
	calls               int
}

func (m *mockQuerier) GetDownstreamTokenByHash(_ context.Context, _ string) (dbstore.GetDownstreamTokenByHashRow, error) {
	m.calls++
	return m.getDownstreamResult, m.getDownstreamErr
}

func (m *mockQuerier) ActivatePolicy(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) CreateDownstreamToken(_ context.Context, _ dbstore.CreateDownstreamTokenParams) (dbstore.DownstreamToken, error) {
	return dbstore.DownstreamToken{}, nil
}
func (m *mockQuerier) CreatePolicy(_ context.Context, _ dbstore.CreatePolicyParams) (dbstore.Policy, error) {
	return dbstore.Policy{}, nil
}
func (m *mockQuerier) CreateUpstreamCredential(_ context.Context, _ dbstore.CreateUpstreamCredentialParams) (dbstore.UpstreamCredential, error) {
	return dbstore.UpstreamCredential{}, nil
}
func (m *mockQuerier) CreateUser(_ context.Context, _ string) (dbstore.User, error) {
	return dbstore.User{}, nil
}
func (m *mockQuerier) DeactivateUserPolicies(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) DeleteDownstreamToken(_ context.Context, _ pgtype.UUID) error  { return nil }
func (m *mockQuerier) DeleteUpstreamCredential(_ context.Context, _ pgtype.UUID) error {
	return nil
}
func (m *mockQuerier) GetActivePolicyForUser(_ context.Context, _ pgtype.UUID) (dbstore.Policy, error) {
	return dbstore.Policy{}, nil
}
func (m *mockQuerier) GetUpstreamCredentialByID(_ context.Context, _ pgtype.UUID) (dbstore.UpstreamCredential, error) {
	return dbstore.UpstreamCredential{}, nil
}
func (m *mockQuerier) GetUserByEmail(_ context.Context, _ string) (dbstore.User, error) {
	return dbstore.User{}, nil
}
func (m *mockQuerier) GetUserByID(_ context.Context, _ pgtype.UUID) (dbstore.User, error) {
	return dbstore.User{}, nil
}
func (m *mockQuerier) ListActiveUpstreamCredentials(_ context.Context, _ pgtype.UUID) ([]dbstore.UpstreamCredential, error) {
	return nil, nil
}
func (m *mockQuerier) ListDownstreamTokensByPolicyID(_ context.Context, _ pgtype.UUID) ([]dbstore.DownstreamToken, error) {
	return nil, nil
}
func (m *mockQuerier) ListPolicyVersions(_ context.Context, _ pgtype.UUID) ([]dbstore.Policy, error) {
	return nil, nil
}
func (m *mockQuerier) ListUpstreamCredentialsByUserID(_ context.Context, _ pgtype.UUID) ([]dbstore.UpstreamCredential, error) {
	return nil, nil
}
func (m *mockQuerier) RevokeDownstreamToken(_ context.Context, _ pgtype.UUID) error { return nil }
func (m *mockQuerier) UpdateDownstreamTokenLastUsed(_ context.Context, _ pgtype.UUID) error {
	return nil
}
func (m *mockQuerier) UpdateUpstreamCredential(_ context.Context, _ dbstore.UpdateUpstreamCredentialParams) (dbstore.UpstreamCredential, error) {
	return dbstore.UpstreamCredential{}, nil
}

func plainHasher(token string) (string, error) { return token, nil }
func plainValidator(token, hash string) bool    { return token == hash }

func TestDBApi_CacheMiss(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := New(mq, 15*time.Second, plainHasher, plainValidator)
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

	d := New(mq, 15*time.Second, plainHasher, plainValidator)

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

	d := New(mq, 1*time.Millisecond, plainHasher, plainValidator)

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

	d := New(mq, 15*time.Second, plainHasher, plainValidator)
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
	d := New(mq, 15*time.Second, plainHasher, alwaysFalse)

	_, err := d.GetPolicyData(context.Background(), "test-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestDBApi_Invalidate(t *testing.T) {
	mq := &mockQuerier{
		getDownstreamResult: makeTestDBResult(),
	}

	d := New(mq, 15*time.Second, plainHasher, plainValidator)

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

	d := New(mq, 15*time.Second, plainHasher, plainValidator)

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

func makeTestDBResult() dbstore.GetDownstreamTokenByHashRow {
	rules, _ := json.Marshal([]any{})
	userID := pgtype.UUID{Valid: true}

	return dbstore.GetDownstreamTokenByHashRow{
		ID:        pgtype.UUID{Valid: true},
		TokenHash: "test-token",
		Policy: dbstore.Policy{
			Version:       "1.0",
			BaseUrl:       "https://api.example.com",
			DefaultAction: "allow",
			Rules:         rules,
			UserID:        userID,
		},
		UpstreamCredential: dbstore.UpstreamCredential{
			Token: "upstream-tok",
		},
	}
}

// ensure api.PolicyGroup is used by the return type
var _ *api.PolicyGroup
