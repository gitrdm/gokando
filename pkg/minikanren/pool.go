package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

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
// for zero-copy streaming. This implementation reuses constraint stores
// from a pool to minimize allocations in high-throughput scenarios.
type PooledResultStream struct {
	*ChannelResultStream
	pool      *ConstraintStorePool
	useGlobal bool // Whether to use global bus stores
}

// NewPooledResultStream creates a new pooled result stream with the specified buffer size.
// The pool parameter controls store reuse, and useGlobal determines whether
// stores should have global constraint bus integration.
func NewPooledResultStream(pool *ConstraintStorePool, bufferSize int, useGlobal bool) ResultStream {
	return &PooledResultStream{
		ChannelResultStream: &ChannelResultStream{
			ch: make(chan ConstraintStore, bufferSize),
		},
		pool:      pool,
		useGlobal: useGlobal,
	}
}

// Put implements ResultStream.Put with pooled store allocation.
func (prs *PooledResultStream) Put(ctx context.Context, store ConstraintStore) error {
	// First put the incoming store back to the pool if it's poolable
	if prs.pool != nil {
		if prs.useGlobal {
			prs.pool.PutGlobal(store)
		} else {
			prs.pool.PutLocal(store)
		}
	}

	// Get a fresh store from the pool for the channel
	var pooledStore ConstraintStore
	if prs.pool != nil {
		if prs.useGlobal {
			pooledStore = prs.pool.GetGlobal()
		} else {
			pooledStore = prs.pool.GetLocal()
		}
	} else {
		// Fallback to new store creation if no pool
		if prs.useGlobal {
			pooledStore = NewLocalConstraintStore(NewGlobalConstraintBus())
		} else {
			pooledStore = NewLocalConstraintStore(nil)
		}
	}

	// Copy the original store's state to the pooled store
	if err := prs.copyStoreState(store, pooledStore); err != nil {
		// Return pooled store back to pool on error
		if prs.pool != nil {
			if prs.useGlobal {
				prs.pool.PutGlobal(pooledStore)
			} else {
				prs.pool.PutLocal(pooledStore)
			}
		}
		return fmt.Errorf("failed to copy store state: %w", err)
	}

	// Put the pooled store into the channel
	return prs.ChannelResultStream.Put(ctx, pooledStore)
}

// copyStoreState copies the state from source to destination store.
// This enables zero-copy streaming by reusing store instances.
func (prs *PooledResultStream) copyStoreState(src, dst ConstraintStore) error {
	// Copy bindings
	srcBindings := src.GetSubstitution()
	if srcBindings != nil {
		for varID, term := range srcBindings.bindings {
			if err := dst.AddBinding(varID, term); err != nil {
				return err
			}
		}
	}

	// Copy constraints
	srcConstraints := src.GetConstraints()
	for _, constraint := range srcConstraints {
		if err := dst.AddConstraint(constraint); err != nil {
			return err
		}
	}

	return nil
}

// Close implements ResultStream.Close with proper pool cleanup.
func (prs *PooledResultStream) Close() error {
	err := prs.ChannelResultStream.Close()

	// Note: Stores in the channel will be returned to pool by consumers
	// or cleaned up by garbage collection if the stream is abandoned

	return err
}

// String returns a string representation of the pooled stream for debugging.
func (prs *PooledResultStream) String() string {
	stats := prs.pool.GetStats()
	return fmt.Sprintf("PooledResultStream{useGlobal: %v, poolStats: %+v}",
		prs.useGlobal, stats)
}
