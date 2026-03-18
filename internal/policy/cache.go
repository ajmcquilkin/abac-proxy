package policy

import (
	"context"
	"sync"
	"time"

	"github.com/abac/proxy/internal/auth"
	"github.com/abac/proxy/internal/db"
)

type PolicyCache struct {
	store *db.Store
	ttl   time.Duration

	mu       sync.RWMutex
	engines  map[string]*cachedEngine
}

type cachedEngine struct {
	engine   *PolicyEngine
	loadedAt time.Time
}

func NewPolicyCache(store *db.Store, ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		store:   store,
		ttl:     ttl,
		engines: make(map[string]*cachedEngine),
	}
}

func (pc *PolicyCache) GetByToken(ctx context.Context, token string) (*PolicyEngine, error) {
	// Hash the token for cache key
	tokenHash, err := auth.HashToken(token)
	if err != nil {
		return nil, err
	}

	pc.mu.RLock()
	if cached, exists := pc.engines[tokenHash]; exists && time.Since(cached.loadedAt) < pc.ttl {
		engine := cached.engine
		pc.mu.RUnlock()
		return engine, nil
	}
	pc.mu.RUnlock()

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := pc.engines[tokenHash]; exists && time.Since(cached.loadedAt) < pc.ttl {
		return cached.engine, nil
	}

	// Load fresh policy from database
	engine, err := NewPolicyEngineFromDownstreamToken(ctx, pc.store, token)
	if err != nil {
		return nil, err
	}

	pc.engines[tokenHash] = &cachedEngine{
		engine:   engine,
		loadedAt: time.Now(),
	}

	return engine, nil
}

func (pc *PolicyCache) Invalidate(token string) {
	tokenHash, err := auth.HashToken(token)
	if err != nil {
		return
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.engines, tokenHash)
}

func (pc *PolicyCache) InvalidateByTokenHash(tokenHash string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.engines, tokenHash)
}

func (pc *PolicyCache) InvalidateAll() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.engines = make(map[string]*cachedEngine)
}
