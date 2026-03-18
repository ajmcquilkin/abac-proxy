package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abac/proxy/internal/db"
	"github.com/abac/proxy/internal/policy"
	"github.com/google/uuid"
)

type TokenHasher func(token string) (string, error)
type TokenValidator func(token, hash string) bool

type DBApi struct {
	querier   db.Querier
	hasher    TokenHasher
	validator TokenValidator
	ttl       time.Duration

	mu    sync.RWMutex
	cache map[string]*cachedEntry
}

type cachedEntry struct {
	data     *PolicyData
	loadedAt time.Time
}

var _ Api = (*DBApi)(nil)

func NewDBApi(querier db.Querier, ttl time.Duration, hasher TokenHasher, validator TokenValidator) *DBApi {
	return &DBApi{
		querier:   querier,
		hasher:    hasher,
		validator: validator,
		ttl:       ttl,
		cache:     make(map[string]*cachedEntry),
	}
}

func (d *DBApi) GetPolicyData(ctx context.Context, token string) (*PolicyData, error) {
	tokenHash, err := d.hasher(token)
	if err != nil {
		return nil, fmt.Errorf("failed to hash token: %w", err)
	}

	d.mu.RLock()
	if cached, exists := d.cache[tokenHash]; exists && time.Since(cached.loadedAt) < d.ttl {
		data := cached.data
		d.mu.RUnlock()
		return data, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	if cached, exists := d.cache[tokenHash]; exists && time.Since(cached.loadedAt) < d.ttl {
		return cached.data, nil
	}

	data, err := d.loadFromDB(ctx, token, tokenHash)
	if err != nil {
		return nil, err
	}

	d.cache[tokenHash] = &cachedEntry{
		data:     data,
		loadedAt: time.Now(),
	}

	return data, nil
}

func (d *DBApi) Invalidate(token string) {
	tokenHash, err := d.hasher(token)
	if err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.cache, tokenHash)
}

func (d *DBApi) InvalidateAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache = make(map[string]*cachedEntry)
}

func (d *DBApi) loadFromDB(ctx context.Context, token, tokenHash string) (*PolicyData, error) {
	result, err := d.querier.GetDownstreamTokenByHash(ctx, tokenHash)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for token")
		}
		return nil, fmt.Errorf("failed to get policy by token: %w", err)
	}

	if !d.validator(token, result.TokenHash) {
		return nil, fmt.Errorf("invalid token")
	}

	var rules []policy.PolicyRule
	if err := json.Unmarshal(result.Policy.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules: %w", err)
	}

	userID := uuid.UUID(result.Policy.UserID.Bytes).String()

	p := &policy.Policy{
		Version:       result.Policy.Version,
		User:          policy.PolicyUser{Token: result.UpstreamCredential.Token, ID: userID},
		BaseURL:       result.Policy.BaseUrl,
		Rules:         rules,
		DefaultAction: result.Policy.DefaultAction,
	}

	if err := policy.ValidatePolicy(p); err != nil {
		return nil, fmt.Errorf("invalid policy from database: %w", err)
	}

	go func() {
		_ = d.querier.UpdateDownstreamTokenLastUsed(context.Background(), result.ID)
	}()

	return &PolicyData{
		Policy:               p,
		UpstreamToken:        result.UpstreamCredential.Token,
		UpstreamTokenType:    result.UpstreamCredential.TokenType,
		UpstreamHeaderString: result.UpstreamCredential.HeaderString,
	}, nil
}
