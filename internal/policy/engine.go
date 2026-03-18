package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/abac/proxy/internal/auth"
	"github.com/abac/proxy/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PolicyEngine struct {
	policy               *Policy
	matcher              *PathMatcher
	filterer             *ResponseFilterer
	upstreamToken        string
	upstreamTokenType    *string
	upstreamHeaderString *string
}

func NewPolicyEngine(policyPath string) (*PolicyEngine, error) {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
	}

	if err := ValidatePolicy(&p); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	bearerType := "bearer"
	return &PolicyEngine{
		policy:               &p,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        p.User.Token,
		upstreamTokenType:    &bearerType,
		upstreamHeaderString: nil,
	}, nil
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
	for i := range pe.policy.Rules {
		rule := &pe.policy.Rules[i]
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

func NewPolicyEngineFromDB(ctx context.Context, store *db.Store, userID string) (*PolicyEngine, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var uidBytes [16]byte
	copy(uidBytes[:], uid[:])
	pgUUID := pgtype.UUID{
		Bytes: uidBytes,
		Valid: true,
	}

	policyRow, err := store.GetActivePolicyForUser(ctx, pgUUID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to get active policy: %w", err)
	}

	upstreamCred, err := store.GetUpstreamCredentialByID(ctx, policyRow.UpstreamCredentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream credential: %w", err)
	}

	var rules []PolicyRule
	if err := json.Unmarshal(policyRow.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules from database: %w", err)
	}

	p := Policy{
		Version:       policyRow.Version,
		User:          PolicyUser{Token: upstreamCred.Token, ID: userID},
		BaseURL:       policyRow.BaseUrl,
		Rules:         rules,
		DefaultAction: policyRow.DefaultAction,
	}

	if err := ValidatePolicy(&p); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	return &PolicyEngine{
		policy:               &p,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        upstreamCred.Token,
		upstreamTokenType:    upstreamCred.TokenType,
		upstreamHeaderString: upstreamCred.HeaderString,
	}, nil
}

func NewPolicyEngineFromDownstreamToken(ctx context.Context, store *db.Store, token string) (*PolicyEngine, error) {
	tokenHash, err := auth.HashToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	result, err := store.GetDownstreamTokenByHash(ctx, tokenHash)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for token")
		}
		return nil, fmt.Errorf("failed to get policy by token: %w", err)
	}

	if !auth.ValidateToken(token, result.TokenHash) {
		return nil, fmt.Errorf("invalid token")
	}

	var rules []PolicyRule
	if err := json.Unmarshal(result.Policy.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules from database: %w", err)
	}

	userID := uuid.UUID(result.Policy.UserID.Bytes).String()

	p := Policy{
		Version:       result.Policy.Version,
		User:          PolicyUser{Token: result.UpstreamCredential.Token, ID: userID},
		BaseURL:       result.Policy.BaseUrl,
		Rules:         rules,
		DefaultAction: result.Policy.DefaultAction,
	}

	if err := ValidatePolicy(&p); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	go func() {
		_ = store.UpdateDownstreamTokenLastUsed(context.Background(), result.ID)
	}()

	return &PolicyEngine{
		policy:               &p,
		matcher:              NewPathMatcher(),
		filterer:             NewResponseFilterer(),
		upstreamToken:        result.UpstreamCredential.Token,
		upstreamTokenType:    result.UpstreamCredential.TokenType,
		upstreamHeaderString: result.UpstreamCredential.HeaderString,
	}, nil
}
