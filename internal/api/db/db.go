package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abac/proxy/internal/api"
	dbstore "github.com/abac/proxy/internal/db"
	"github.com/google/uuid"
)

type TokenHasher func(token string) (string, error)
type TokenValidator func(token, hash string) bool

type dbApi struct {
	querier   dbstore.Querier
	hasher    TokenHasher
	validator TokenValidator
	ttl       time.Duration

	mu    sync.RWMutex
	cache map[string]*cachedEntry
}

type cachedEntry struct {
	data     *api.PolicyGroup
	loadedAt time.Time
}

var _ api.Api = (*dbApi)(nil)

func New(querier dbstore.Querier, ttl time.Duration, hasher TokenHasher, validator TokenValidator) api.Api {
	return &dbApi{
		querier:   querier,
		hasher:    hasher,
		validator: validator,
		ttl:       ttl,
		cache:     make(map[string]*cachedEntry),
	}
}

func (d *dbApi) GetPolicyData(ctx context.Context, token string) (*api.PolicyGroup, error) {
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

func (d *dbApi) GetAllowedHosts() []api.HostEntry {
	return nil
}

func (d *dbApi) Invalidate(token string) {
	tokenHash, err := d.hasher(token)
	if err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.cache, tokenHash)
}

func (d *dbApi) InvalidateAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache = make(map[string]*cachedEntry)
}

func (d *dbApi) loadFromDB(ctx context.Context, token, tokenHash string) (*api.PolicyGroup, error) {
	result, err := d.querier.GetDownstreamTokenByHash(ctx, tokenHash)
	if err != nil {
		if dbstore.IsNotFound(err) {
			return nil, fmt.Errorf("no active policy found for token")
		}
		return nil, fmt.Errorf("failed to get policy by token: %w", err)
	}

	if !d.validator(token, result.TokenHash) {
		return nil, fmt.Errorf("invalid token")
	}

	var rules []api.PolicyRule
	if err := json.Unmarshal(result.Policy.Rules, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse policy rules: %w", err)
	}

	_ = uuid.UUID(result.Policy.UserID.Bytes).String()

	p := api.Policy{
		BaseURL:       result.Policy.BaseUrl,
		UpstreamToken: result.UpstreamCredential.Token,
		Rules:         rules,
	}

	if result.UpstreamCredential.TokenType != nil {
		p.UpstreamTokenType = *result.UpstreamCredential.TokenType
	}
	if result.UpstreamCredential.HeaderString != nil {
		p.UpstreamHeaderString = *result.UpstreamCredential.HeaderString
	}

	go func() {
		_ = d.querier.UpdateDownstreamTokenLastUsed(context.Background(), result.ID)
	}()

	return &api.PolicyGroup{
		Version:       result.Policy.Version,
		Policies:      []api.Policy{p},
		DefaultAction: result.Policy.DefaultAction,
	}, nil
}
