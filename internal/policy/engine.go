package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
	policy   *Policy
	matcher  *PathMatcher
	filterer *ResponseFilterer
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

	return &PolicyEngine{
		policy:   &policy,
		matcher:  NewPathMatcher(),
		filterer: NewResponseFilterer(),
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

func (pe *PolicyEngine) ValidateToken(token string) bool {
	return token == pe.policy.User.Token
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

	var policy Policy
	if err := json.Unmarshal(policyRow.Content, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON from database: %w", err)
	}

	if err := validatePolicy(&policy); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	return &PolicyEngine{
		policy:   &policy,
		matcher:  NewPathMatcher(),
		filterer: NewResponseFilterer(),
	}, nil
}
