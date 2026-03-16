package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/abac/proxy/internal/auth"
	"github.com/abac/proxy/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Policy struct {
	Version       string       `json:"version"`
	User          PolicyUser   `json:"user"`
	BaseURL       string       `json:"baseUrl"`
	Policies      []PolicyRule `json:"policies"`
	DefaultAction string       `json:"default_action"`
}

type PolicyUser struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

type PolicyRule struct {
	Route          string          `json:"route"`
	Method         string          `json:"method"`
	Action         string          `json:"action"`
	ResponseFilter *ResponseFilter `json:"response_filter,omitempty"`
}

type PolicyEngine struct {
	policy             *Policy
	matcher            *PathMatcher
	filterer           *ResponseFilterer
	upstreamToken      string
	upstreamTokenType  *string
	upstreamHeaderString *string
}

func NewPolicyEngine(policyPath string) (*PolicyEngine, error) {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
	}

	if err := validatePolicy(&policy); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	// File-based policies use bearer auth by default
	bearerType := "bearer"
	return &PolicyEngine{
		policy:               &policy,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        policy.User.Token,
		upstreamTokenType:    &bearerType,
		upstreamHeaderString: nil,
	}, nil
}

func validatePolicy(p *Policy) error {
	if p.Version == "" {
		return fmt.Errorf("version is required")
	}
	if p.User.Token == "" {
		return fmt.Errorf("user token is required")
	}
	if p.DefaultAction == "" {
		return fmt.Errorf("default_action is required")
	}
	if p.DefaultAction != "allow" && p.DefaultAction != "deny" {
		return fmt.Errorf("default_action must be 'allow' or 'deny'")
	}
	return nil
}

func (pe *PolicyEngine) GetUpstreamToken() string {
	return pe.upstreamToken
}

func (pe *PolicyEngine) GetUpstreamTokenType() *string {
	return pe.upstreamTokenType
}

func (pe *PolicyEngine) GetUpstreamHeaderString() *string {
	return pe.upstreamHeaderString
}

func (pe *PolicyEngine) FindMatchingRule(path, method string) (*PolicyRule, bool) {
	for i := range pe.policy.Policies {
		rule := &pe.policy.Policies[i]
		if pe.matcher.MatchesWithMethod(rule.Route, rule.Method, path, method) {
			return rule, true
		}
	}
	return nil, false
}

func (pe *PolicyEngine) GetDefaultAction() string {
	return pe.policy.DefaultAction
}

func (pe *PolicyEngine) GetFilterer() *ResponseFilterer {
	return pe.filterer
}

// NewPolicyEngineFromDB creates a PolicyEngine from database storage
func NewPolicyEngineFromDB(ctx context.Context, store *storage.Store, userID string) (*PolicyEngine, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert uuid.UUID to pgtype.UUID
	var uidBytes [16]byte
	copy(uidBytes[:], uid[:])
	pgUUID := pgtype.UUID{
		Bytes: uidBytes,
		Valid: true,
	}

	policyRow, err := store.GetActivePolicyForUser(ctx, pgUUID)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get active policy: %w", err)
	}

	// Get upstream credential
	upstreamCred, err := store.GetUpstreamCredentialByID(ctx, policyRow.UpstreamCredentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream credential: %w", err)
	}

	// Parse rules from JSONB
	var rules []PolicyRule
	if err := json.Unmarshal(policyRow.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules from database: %w", err)
	}

	// Construct policy from normalized columns
	policy := Policy{
		Version:       policyRow.Version,
		User:          PolicyUser{Token: upstreamCred.Token, ID: userID},
		BaseURL:       policyRow.BaseUrl,
		Policies:      rules,
		DefaultAction: policyRow.DefaultAction,
	}

	if err := validatePolicy(&policy); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	return &PolicyEngine{
		policy:               &policy,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        upstreamCred.Token,
		upstreamTokenType:    upstreamCred.TokenType,
		upstreamHeaderString: upstreamCred.HeaderString,
	}, nil
}

// NewPolicyEngineFromDownstreamToken creates a PolicyEngine by looking up the downstream token
func NewPolicyEngineFromDownstreamToken(ctx context.Context, store *storage.Store, token string) (*PolicyEngine, error) {
	// Hash the client token
	tokenHash, err := auth.HashToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	// Query with 3-way JOIN
	result, err := store.GetDownstreamTokenByHash(ctx, tokenHash)
	if err != nil {
		if storage.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for token")
		}
		return nil, fmt.Errorf("failed to get policy by token: %w", err)
	}

	// Validate token hash
	if !auth.ValidateToken(token, result.TokenHash) {
		return nil, fmt.Errorf("invalid token")
	}

	// Parse rules from JSONB
	var rules []PolicyRule
	if err := json.Unmarshal(result.Policy.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules from database: %w", err)
	}

	// Convert UUID to string for user ID
	userID := uuid.UUID(result.Policy.UserID.Bytes).String()

	// Construct policy from normalized columns
	policy := Policy{
		Version:       result.Policy.Version,
		User:          PolicyUser{Token: result.UpstreamCredential.Token, ID: userID},
		BaseURL:       result.Policy.BaseUrl,
		Policies:      rules,
		DefaultAction: result.Policy.DefaultAction,
	}

	if err := validatePolicy(&policy); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	// Update last_used_at asynchronously
	go func() {
		_ = store.UpdateDownstreamTokenLastUsed(context.Background(), result.ID)
	}()

	return &PolicyEngine{
		policy:               &policy,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        result.UpstreamCredential.Token,
		upstreamTokenType:    result.UpstreamCredential.TokenType,
		upstreamHeaderString: result.UpstreamCredential.HeaderString,
	}, nil
}
