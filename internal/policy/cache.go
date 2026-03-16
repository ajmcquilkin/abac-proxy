package policy

import (
	"context"
	"sync"
	"time"

	"github.com/abac/proxy/internal/storage"
)

type PolicyCache struct {
	store *storage.Store
	ttl   time.Duration

	mu       sync.RWMutex
	engines  map[string]*cachedEngine
}

type cachedEngine struct {
	engine   *PolicyEngine
	loadedAt time.Time
}

func NewPolicyCache(store *storage.Store, ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		store:   store,
		ttl:     ttl,
		engines: make(map[string]*cachedEngine),
	}
}

func (pc *PolicyCache) GetByToken(ctx context.Context, token string) (*PolicyEngine, error) {
	pc.mu.RLock()
	if cached, exists := pc.engines[token]; exists && time.Since(cached.loadedAt) < pc.ttl {
		engine := cached.engine
		pc.mu.RUnlock()
		return engine, nil
	}
	pc.mu.RUnlock()

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := pc.engines[token]; exists && time.Since(cached.loadedAt) < pc.ttl {
		return cached.engine, nil
	}

	// Load fresh policy from database
	engine, err := NewPolicyEngineFromToken(ctx, pc.store, token)
	if err != nil {
		return nil, err
	}

	pc.engines[token] = &cachedEngine{
		engine:   engine,
		loadedAt: time.Now(),
	}

	return engine, nil
}

func (pc *PolicyCache) Invalidate(token string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.engines, token)
}

func (pc *PolicyCache) InvalidateAll() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.engines = make(map[string]*cachedEngine)
}
