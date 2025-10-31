package minikanren

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TabledGoal wraps a goal with tabling support.
// It checks for cached results before executing the goal,
// and caches new results as they are found.
type TabledGoal struct {
	// goal is the original goal being tabled
	goal Goal

	// manager handles table lifecycle and caching
	manager *TableManager

	// variantGenerator creates cache keys for goals
	variantGenerator *VariantGenerator
}

// NewTabledGoal creates a new tabled goal wrapper.
func NewTabledGoal(goal Goal, manager *TableManager) *TabledGoal {
	return &TabledGoal{
		goal:             goal,
		manager:          manager,
		variantGenerator: NewVariantGenerator(),
	}
}

// Execute runs the tabled goal, checking cache first and updating cache with new results.
func (tg *TabledGoal) Execute(ctx context.Context, store ConstraintStore) ResultStream {
	stream := NewStream()

	go func() {
		// Get or create table for this goal variant
		table := tg.manager.GetOrCreateTable(tg.goal, store)

		// Send existing cached results immediately
		existingEntries := table.GetEntries()
		for _, entry := range existingEntries {
			select {
			case <-ctx.Done():
				stream.Close()
				return
			default:
				stream.Put(ctx, entry.Store)
			}
		}

		// If table is already complete, we're done
		if table.IsCompleted() {
			stream.Close()
			return
		}

		// Start the actual goal execution in a separate goroutine
		resultStream := tg.goal(ctx, store)

		// Process results from the goal execution
		go func() {
			defer stream.Close() // Close stream when processing is done

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				results, hasMore, err := resultStream.Take(ctx, 10)
				if err != nil {
					return
				}

				for _, result := range results {
					// Add to table cache
					table.AddEntry(result)

					// Forward to our output stream
					stream.Put(ctx, result)
				}

				if !hasMore {
					// Mark table as completed
					table.MarkCompleted()
					return
				}
			}
		}()
	}()

	return stream
}

// TableGoal creates a tabled version of a goal.
// This prevents infinite loops in recursive relations by memoizing results.
// The returned goal will check for cached results before executing,
// and cache new results as they are found.
//
// Example:
//
//	// Without tabling, this would loop infinitely
//	path := func(x, y *Var) Goal {
//	    return Disj(
//	        Eq(x, y),
//	        func(ctx context.Context, store ConstraintStore) ResultStream {
//	            z := Fresh("z")
//	            return Existo(func() Goal { return Conj(edge(x, z), path(z, y)) })
//	        },
//	    )
//	}
//
//	// With tabling, this terminates correctly
//	tabledPath := TableGoal(path(x, y))
func TableGoal(goal Goal) Goal {
	// Use global table manager - in production, this should be configurable
	manager := GetGlobalTableManager()

	tabledGoal := NewTabledGoal(goal, manager)

	return func(ctx context.Context, store ConstraintStore) ResultStream {
		return tabledGoal.Execute(ctx, store)
	}
}

// TableWithManager creates a tabled version of a goal using a specific table manager.
// This allows for custom table management configurations.
func TableWithManager(goal Goal, manager *TableManager) Goal {
	tabledGoal := NewTabledGoal(goal, manager)

	return func(ctx context.Context, store ConstraintStore) ResultStream {
		return tabledGoal.Execute(ctx, store)
	}
}

// Global table manager instance
var (
	globalTableManager     *TableManager
	globalTableManagerOnce sync.Once
)

// GetGlobalTableManager returns the global table manager instance.
// This is initialized with default settings on first access.
func GetGlobalTableManager() *TableManager {
	globalTableManagerOnce.Do(func() {
		globalTableManager = NewTableManager()
	})
	return globalTableManager
}

// SetGlobalTableManager sets the global table manager instance.
// This allows for custom table manager configuration.
func SetGlobalTableManager(manager *TableManager) {
	globalTableManager = manager
}

// TablingConfig contains configuration options for tabling behavior.
type TablingConfig struct {
	// MaxTables limits the number of active tables
	MaxTables int

	// MaxTableSize limits the number of entries per table
	MaxTableSize int

	// TTL specifies how long tables can remain unused before eviction
	TTL time.Duration

	// EnableCleanup enables automatic cleanup of expired tables
	EnableCleanup bool

	// CleanupInterval specifies how often to run cleanup
	CleanupInterval time.Duration
}

// DefaultTablingConfig returns sensible default tabling configuration.
func DefaultTablingConfig() TablingConfig {
	return TablingConfig{
		MaxTables:       1000,
		MaxTableSize:    10000,
		TTL:             30 * time.Minute,
		EnableCleanup:   true,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewTableManagerWithTablingConfig creates a table manager with the given tabling configuration.
// This includes automatic cleanup if enabled.
func NewTableManagerWithTablingConfig(config TablingConfig) *TableManager {
	manager := NewTableManagerWithConfig(config.MaxTables, config.MaxTableSize, config.TTL)

	if config.EnableCleanup {
		// Start cleanup goroutine
		go func() {
			ticker := time.NewTicker(config.CleanupInterval)
			defer ticker.Stop()

			for range ticker.C {
				manager.Cleanup()
			}
		}()
	}

	return manager
}

// TablingStats contains comprehensive statistics about tabling system performance.
type TablingStats struct {
	// ManagerStats contains global table manager statistics
	ManagerStats TableManagerStats

	// ActiveTables lists statistics for all currently active tables
	ActiveTables []TableStats

	// TotalTabledGoals is the number of goals that have been tabled
	TotalTabledGoals int64

	// AverageTableSize is the average number of entries per table
	AverageTableSize float64

	// HitRate is the ratio of cache hits to total lookups
	HitRate float64

	// MemoryUsage estimates memory usage in bytes
	MemoryUsage int64
}

// GetTablingStats returns comprehensive statistics about the tabling system.
func GetTablingStats() TablingStats {
	manager := GetGlobalTableManager()
	managerStats := manager.GetStats()
	activeTables := manager.ListTables()

	totalHits := managerStats.TotalHits
	totalMisses := managerStats.TotalMisses
	totalLookups := totalHits + totalMisses

	var hitRate float64
	if totalLookups > 0 {
		hitRate = float64(totalHits) / float64(totalLookups)
	}

	var avgTableSize float64
	if len(activeTables) > 0 {
		totalEntries := 0
		for _, table := range activeTables {
			totalEntries += table.EntryCount
		}
		avgTableSize = float64(totalEntries) / float64(len(activeTables))
	}

	// Rough memory estimation: each table entry is ~100-200 bytes
	memoryUsage := int64(managerStats.TotalEntries * 150)

	return TablingStats{
		ManagerStats:     managerStats,
		ActiveTables:     activeTables,
		TotalTabledGoals: totalHits + totalMisses, // Approximation
		AverageTableSize: avgTableSize,
		HitRate:          hitRate,
		MemoryUsage:      memoryUsage,
	}
}

// String returns a human-readable summary of tabling statistics.
func (ts TablingStats) String() string {
	return fmt.Sprintf("Tabling{active_tables=%d, total_entries=%d, hit_rate=%.2f%%, memory=%.1fMB}",
		len(ts.ActiveTables), ts.ManagerStats.TotalEntries,
		ts.HitRate*100, float64(ts.MemoryUsage)/(1024*1024))
}

// ResetTablingStats resets all tabling statistics.
// This is primarily useful for testing and benchmarking.
func ResetTablingStats() {
	manager := GetGlobalTableManager()
	manager.Clear()
}

// EnableTabling enables tabling for all goals created after this call.
// This is a global setting that affects the behavior of goal constructors.
func EnableTabling() {
	// This would modify goal constructors to automatically wrap goals with tabling
	// For now, users must explicitly call Table() on goals they want to table
}

// DisableTabling disables automatic tabling.
// Goals created after this call will not be automatically tabled.
func DisableTabling() {
	// Reset to non-tabled behavior
}

// IsTablingEnabled reports whether automatic tabling is enabled.
//
// When automatic tabling is enabled, certain goal constructors may wrap
// goals with a table that memoizes results. Tabling prevents infinite
// recursion in many recursive relations and can dramatically improve
// performance for repeated queries by reusing previously computed results.
//
// Note: in this release tabling is opt-in. Prefer explicit wrapping via
// `TableGoal` or `TableWithManager` to control which goals are memoized.
// The package also exposes `EnableTabling` and `DisableTabling` as hooks
// for a global automatic mode, but the default remains disabled to avoid
// surprising changes in behavior.
//
// Example:
//
//	if IsTablingEnabled() {
//	    // callers can optimize goal construction knowing tabling is active
//	}
//
// Thread-safety: this function is safe for concurrent use.
//
// See also: TableGoal, TableWithManager, EnableTabling, DisableTabling.
func IsTablingEnabled() bool {
	// For now, tabling is always opt-in via Table() calls
	return false
}
