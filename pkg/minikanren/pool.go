package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// pooledStoreCounter provides unique IDs for pooled stores
var pooledStoreCounter int64

// ConstraintStorePool provides zero-copy buffer pooling for ConstraintStore instances.
// This enables efficient reuse of constraint stores in high-throughput streaming scenarios,
// reducing garbage collection pressure and memory allocations.
//
// The pool maintains separate pools for stores with and without global constraint bus
// integration, as these have different lifecycle requirements.
type ConstraintStorePool struct {
	// localPool pools stores without global bus integration
	localPool sync.Pool

	// globalPool pools stores with global bus integration
	globalPool sync.Pool

	// stats tracks pool performance metrics
	stats PoolStats

	// maxSize limits the number of stores that can be pooled
	maxSize int

	// mu protects stats updates
	mu sync.RWMutex
}

// PoolStats tracks performance metrics for the constraint store pool.
// These metrics help monitor pool efficiency and identify optimization opportunities.
type PoolStats struct {
	// Hits is the number of times a store was successfully retrieved from the pool
	Hits int64

	// Misses is the number of times a new store had to be created
	Misses int64

	// Returns is the number of times a store was returned to the pool
	Returns int64

	// Evictions is the number of times a store was evicted due to pool size limits
	Evictions int64

	// TotalAllocations is the total number of stores ever created by this pool
	TotalAllocations int64

	// CurrentSize is the current number of stores in the pool
	CurrentSize int64

	// MaxSizeReached indicates if the pool ever reached its maximum size
	MaxSizeReached bool

	// LastReset tracks when the pool was last reset for monitoring
	LastReset time.Time
}

// NewConstraintStorePool creates a new constraint store pool with the specified maximum size.
// A maxSize of 0 indicates no limit on pool size.
func NewConstraintStorePool(maxSize int) *ConstraintStorePool {
	pool := &ConstraintStorePool{
		maxSize: maxSize,
		stats: PoolStats{
			LastReset: time.Now(),
		},
	}

	// Initialize local pool (stores without global bus)
	pool.localPool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.stats.TotalAllocations, 1)
			atomic.AddInt64(&pool.stats.Misses, 1)
			return NewLocalConstraintStore(nil)
		},
	}

	// Initialize global pool (stores with global bus)
	pool.globalPool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.stats.TotalAllocations, 1)
			atomic.AddInt64(&pool.stats.Misses, 1)
			return NewLocalConstraintStore(NewGlobalConstraintBus())
		},
	}

	return pool
}

// GetLocal retrieves a constraint store from the local pool.
// The returned store is ready for use and has no global bus integration.
func (csp *ConstraintStorePool) GetLocal() ConstraintStore {
	store := csp.localPool.Get().(*LocalConstraintStoreImpl)
	atomic.AddInt64(&csp.stats.Hits, 1)
	atomic.AddInt64(&csp.stats.CurrentSize, -1) // Will be incremented when returned
	return store
}

// GetGlobal retrieves a constraint store from the global pool.
// The returned store is ready for use and has global bus integration.
func (csp *ConstraintStorePool) GetGlobal() ConstraintStore {
	store := csp.globalPool.Get().(*LocalConstraintStoreImpl)
	atomic.AddInt64(&csp.stats.Hits, 1)
	atomic.AddInt64(&csp.stats.CurrentSize, -1) // Will be incremented when returned
	return store
}

// PutLocal returns a constraint store to the local pool for reuse.
// The store is reset to a clean state before being pooled.
// Returns true if the store was successfully pooled, false if it was evicted.
func (csp *ConstraintStorePool) PutLocal(store ConstraintStore) bool {
	localStore, ok := store.(*LocalConstraintStoreImpl)
	if !ok {
		// Wrong type, cannot pool
		return false
	}

	// Check if we should evict due to size limits
	if csp.maxSize > 0 {
		currentSize := atomic.LoadInt64(&csp.stats.CurrentSize)
		if currentSize >= int64(csp.maxSize) {
			atomic.AddInt64(&csp.stats.Evictions, 1)
			csp.stats.MaxSizeReached = true
			localStore.Shutdown() // Clean shutdown
			return false
		}
	}

	// Reset the store to clean state
	localStore.Reset()

	// Return to pool
	csp.localPool.Put(localStore)
	atomic.AddInt64(&csp.stats.Returns, 1)
	atomic.AddInt64(&csp.stats.CurrentSize, 1)

	return true
}

// PutGlobal returns a constraint store to the global pool for reuse.
// The store is reset to a clean state before being pooled.
// Returns true if the store was successfully pooled, false if it was evicted.
func (csp *ConstraintStorePool) PutGlobal(store ConstraintStore) bool {
	localStore, ok := store.(*LocalConstraintStoreImpl)
	if !ok {
		// Wrong type, cannot pool
		return false
	}

	// Check if we should evict due to size limits
	if csp.maxSize > 0 {
		currentSize := atomic.LoadInt64(&csp.stats.CurrentSize)
		if currentSize >= int64(csp.maxSize) {
			atomic.AddInt64(&csp.stats.Evictions, 1)
			csp.stats.MaxSizeReached = true
			localStore.Shutdown() // Clean shutdown
			return false
		}
	}

	// Reset the store to clean state
	localStore.Reset()

	// Return to pool
	csp.globalPool.Put(localStore)
	atomic.AddInt64(&csp.stats.Returns, 1)
	atomic.AddInt64(&csp.stats.CurrentSize, 1)

	return true
}

// GetStats returns a snapshot of the current pool statistics.
// This is safe to call concurrently with pool operations.
func (csp *ConstraintStorePool) GetStats() PoolStats {
	csp.mu.RLock()
	defer csp.mu.RUnlock()

	return PoolStats{
		Hits:             atomic.LoadInt64(&csp.stats.Hits),
		Misses:           atomic.LoadInt64(&csp.stats.Misses),
		Returns:          atomic.LoadInt64(&csp.stats.Returns),
		Evictions:        atomic.LoadInt64(&csp.stats.Evictions),
		TotalAllocations: atomic.LoadInt64(&csp.stats.TotalAllocations),
		CurrentSize:      atomic.LoadInt64(&csp.stats.CurrentSize),
		MaxSizeReached:   csp.stats.MaxSizeReached,
		LastReset:        csp.stats.LastReset,
	}
}

// Reset resets the pool statistics and clears all pooled stores.
// This is primarily used for testing and benchmarking.
func (csp *ConstraintStorePool) Reset() {
	csp.mu.Lock()
	defer csp.mu.Unlock()

	// Reset statistics
	atomic.StoreInt64(&csp.stats.Hits, 0)
	atomic.StoreInt64(&csp.stats.Misses, 0)
	atomic.StoreInt64(&csp.stats.Returns, 0)
	atomic.StoreInt64(&csp.stats.Evictions, 0)
	atomic.StoreInt64(&csp.stats.TotalAllocations, 0)
	atomic.StoreInt64(&csp.stats.CurrentSize, 0)
	csp.stats.MaxSizeReached = false
	csp.stats.LastReset = time.Now()

	// Note: We don't actually clear the sync.Pool contents as that's not possible
	// with the sync.Pool API. The pools will naturally clear over time.
}

// HitRate returns the pool hit rate as a percentage (0.0 to 1.0).
// A higher hit rate indicates better pool efficiency.
func (csp *ConstraintStorePool) HitRate() float64 {
	hits := atomic.LoadInt64(&csp.stats.Hits)
	misses := atomic.LoadInt64(&csp.stats.Misses)

	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total)
}

// PooledResultStream extends ChannelResultStream with buffer pool integration
// for zero-copy streaming. This implementation uses StoreRefs to pass lightweight
// references through channels, enabling true zero-copy streaming where stores
// are only copied when they actually diverge.
type PooledResultStream struct {
	refCh     chan StoreRef // Channel for StoreRefs instead of full stores
	pool      *ConstraintStorePool
	useGlobal bool                 // Whether to use global bus stores
	resolver  *PooledStoreResolver // Resolver for StoreRefs
	count     int64                // Atomic counter for results
	closed    int32                // Atomic flag for closed state
	mu        sync.Mutex
}

// PooledStoreResolver resolves StoreRefs for pooled streaming.
// It maintains a registry of stores and provides lazy resolution.
type PooledStoreResolver struct {
	stores    map[string]ConstraintStore
	mu        sync.RWMutex
	pool      *ConstraintStorePool
	useGlobal bool
}

// NewPooledStoreResolver creates a new resolver for pooled stores.
func NewPooledStoreResolver(pool *ConstraintStorePool, useGlobal bool) *PooledStoreResolver {
	return &PooledStoreResolver{
		stores:    make(map[string]ConstraintStore),
		pool:      pool,
		useGlobal: useGlobal,
	}
}

// RegisterStore registers a store with the resolver for later retrieval.
func (psr *PooledStoreResolver) RegisterStore(store ConstraintStore) StoreRef {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	// Generate a unique ID for this store
	storeID := fmt.Sprintf("pooled-%d", atomic.AddInt64(&pooledStoreCounter, 1))
	psr.stores[storeID] = store

	return StoreRef{
		storeID:  storeID,
		resolver: psr,
	}
}

// ResolveStore resolves a StoreRef back to a ConstraintStore.
// Implements the StoreResolver interface.
func (psr *PooledStoreResolver) ResolveStore(ref StoreRef) ConstraintStore {
	psr.mu.RLock()
	store, exists := psr.stores[ref.storeID]
	psr.mu.RUnlock()

	if exists {
		return store
	}

	// Store not found - this shouldn't happen in normal operation
	// Return a fresh store as fallback
	if psr.pool != nil {
		if psr.useGlobal {
			return psr.pool.GetGlobal()
		} else {
			return psr.pool.GetLocal()
		}
	}

	if psr.useGlobal {
		return NewLocalConstraintStore(NewGlobalConstraintBus())
	}
	return NewLocalConstraintStore(nil)
}

// Cleanup removes a store from the resolver when it's no longer needed.
func (psr *PooledStoreResolver) Cleanup(storeID string) {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	if store, exists := psr.stores[storeID]; exists {
		delete(psr.stores, storeID)
		// Return store to pool
		if psr.pool != nil {
			if psr.useGlobal {
				psr.pool.PutGlobal(store)
			} else {
				psr.pool.PutLocal(store)
			}
		}
	}
}

// NewPooledResultStream creates a new pooled result stream with the specified buffer size.
// The pool parameter controls store reuse, and useGlobal determines whether
// stores should have global constraint bus integration.
func NewPooledResultStream(pool *ConstraintStorePool, bufferSize int, useGlobal bool) ResultStream {
	resolver := NewPooledStoreResolver(pool, useGlobal)

	return &PooledResultStream{
		refCh:     make(chan StoreRef, bufferSize),
		pool:      pool,
		useGlobal: useGlobal,
		resolver:  resolver,
	}
}

// Put implements ResultStream.Put with zero-copy StoreRef streaming.
func (prs *PooledResultStream) Put(ctx context.Context, store ConstraintStore) error {
	// Check if stream is closed
	if atomic.LoadInt32(&prs.closed) == 1 {
		return nil // Silently ignore puts to closed stream
	}

	// Register the store with the resolver and get a reference
	storeRef := prs.resolver.RegisterStore(store)

	select {
	case prs.refCh <- storeRef:
		atomic.AddInt64(&prs.count, 1)
		return nil
	case <-ctx.Done():
		// Context cancelled, clean up the registered store
		prs.resolver.Cleanup(storeRef.storeID)
		return ctx.Err()
	}
}

// Take implements ResultStream.Take with lazy store resolution.
func (prs *PooledResultStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	var results []ConstraintStore

	for i := 0; i < n; i++ {
		select {
		case storeRef, ok := <-prs.refCh:
			if !ok {
				// Channel is closed, no more items
				return results, false, nil
			}

			// Resolve the StoreRef to get the actual store
			store := storeRef.Resolve()

			// Clean up the reference from resolver after use
			defer prs.resolver.Cleanup(storeRef.storeID)

			results = append(results, store)

		case <-ctx.Done():
			// Context cancelled, return what we have so far
			return results, len(results) > 0, ctx.Err()
		}
	}

	// We took exactly n items, check if there are more or if channel is closed
	select {
	case _, ok := <-prs.refCh:
		if ok {
			// There are more items, but we can't put it back
			return results, true, nil
		}
		// Channel is closed
		return results, false, nil
	default:
		// No more items immediately available, but channel might not be closed
		return results, true, nil
	}
}

// Close implements ResultStream.Close.
func (prs *PooledResultStream) Close() error {
	prs.mu.Lock()
	defer prs.mu.Unlock()

	if atomic.LoadInt32(&prs.closed) == 1 {
		return nil // Already closed
	}

	atomic.StoreInt32(&prs.closed, 1)
	close(prs.refCh)
	return nil
}

// Count implements ResultStream.Count.
func (prs *PooledResultStream) Count() int64 {
	return atomic.LoadInt64(&prs.count)
}

func (prs *PooledResultStream) Drain(ctx context.Context) {
	for {
		select {
		case _, ok := <-prs.refCh:
			if !ok {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// String returns a string representation of the pooled stream for debugging.
func (prs *PooledResultStream) String() string {
	if prs.pool != nil {
		stats := prs.pool.GetStats()
		return fmt.Sprintf("PooledResultStream{useGlobal: %v, poolStats: %+v}",
			prs.useGlobal, stats)
	}
	return fmt.Sprintf("PooledResultStream{useGlobal: %v, no pool}", prs.useGlobal)
}
