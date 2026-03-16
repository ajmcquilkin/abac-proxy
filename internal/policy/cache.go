package policy

import (
	"context"
	"sync"
	"time"

	"github.com/abac/proxy/internal/storage"
)

type PolicyCache struct {
	store  *storage.Store
	userID string
	ttl    time.Duration

	mu       sync.RWMutex
	engine   *PolicyEngine
	loadedAt time.Time
}

func NewPolicyCache(store *storage.Store, userID string, ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		store:  store,
		userID: userID,
		ttl:    ttl,
	}
}

func (pc *PolicyCache) Get(ctx context.Context) (*PolicyEngine, error) {
	pc.mu.RLock()
	if pc.engine != nil && time.Since(pc.loadedAt) < pc.ttl {
		engine := pc.engine
		pc.mu.RUnlock()
		return engine, nil
	}
	pc.mu.RUnlock()

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Double-check after acquiring write lock
	if pc.engine != nil && time.Since(pc.loadedAt) < pc.ttl {
		return pc.engine, nil
	}

	// Load fresh policy from database
	engine, err := NewPolicyEngineFromDB(ctx, pc.store, pc.userID)
	if err != nil {
		return nil, err
	}

	pc.engine = engine
	pc.loadedAt = time.Now()

	return engine, nil
}

func (pc *PolicyCache) Invalidate() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.engine = nil
}
