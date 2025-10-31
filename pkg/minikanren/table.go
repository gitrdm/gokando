package minikanren

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// TableEntry represents a single cached result in a table.
// Each entry contains the constraint store that satisfied the goal
// along with metadata for cache management.
type TableEntry struct {
	// Store is the constraint store representing one solution
	Store ConstraintStore

	// Timestamp records when this entry was created
	Timestamp time.Time

	// AccessCount tracks how many times this entry has been accessed
	AccessCount int64

	// LastAccessed records the last time this entry was used
	LastAccessed time.Time
}

// NewTableEntry creates a new table entry with the current timestamp.
func NewTableEntry(store ConstraintStore) *TableEntry {
	now := time.Now()
	return &TableEntry{
		Store:        store,
		Timestamp:    now,
		AccessCount:  0,
		LastAccessed: now,
	}
}

// Access marks the entry as accessed and updates metadata.
func (te *TableEntry) Access() {
	atomic.AddInt64(&te.AccessCount, 1)
	te.LastAccessed = time.Now()
}

// Table represents a memoization cache for a specific goal variant.
// Tables store all solutions found for a particular goal pattern,
// enabling reuse of results for identical subgoals.
type Table struct {
	mu sync.RWMutex

	// variant is the normalized goal representation used as cache key
	variant string

	// entries contains all cached solutions for this table
	entries []*TableEntry

	// completed indicates whether all possible solutions have been found
	completed bool

	// consumers tracks active consumers waiting for this table
	consumers []chan ConstraintStore

	// created records when this table was created
	created time.Time

	// lastUsed records the last time this table was accessed
	lastUsed time.Time

	// hitCount tracks how many times this table has been used
	hitCount int64

	// missCount tracks how many times this table was missed (new computation needed)
	missCount int64
}

// NewTable creates a new table for the given variant.
func NewTable(variant string) *Table {
	now := time.Now()
	return &Table{
		variant:   variant,
		entries:   make([]*TableEntry, 0),
		completed: false,
		consumers: make([]chan ConstraintStore, 0),
		created:   now,
		lastUsed:  now,
	}
}

// AddEntry adds a new solution to the table.
// This is called when a new solution is found during tabled execution.
func (t *Table) AddEntry(store ConstraintStore) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry := NewTableEntry(store)
	t.entries = append(t.entries, entry)

	// Notify all waiting consumers
	for _, consumer := range t.consumers {
		select {
		case consumer <- store:
		default:
			// Consumer channel is full, skip
		}
	}
}

// GetEntries returns a copy of all cached entries.
// This provides thread-safe access to the table contents.
func (t *Table) GetEntries() []*TableEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*TableEntry, len(t.entries))
	copy(result, t.entries)
	return result
}

// MarkCompleted marks the table as completed, indicating all solutions have been found.
func (t *Table) MarkCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.completed = true

	// Close all consumer channels to signal completion
	for _, consumer := range t.consumers {
		close(consumer)
	}
	t.consumers = nil // Clear consumers list
}

// IsCompleted returns whether the table has all possible solutions.
func (t *Table) IsCompleted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.completed
}

// AddConsumer adds a consumer that will receive new solutions as they are found.
func (t *Table) AddConsumer(consumer chan ConstraintStore) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.completed {
		// Table is already complete, close consumer immediately
		close(consumer)
		return
	}

	t.consumers = append(t.consumers, consumer)
}

// RecordHit records a cache hit for this table.
func (t *Table) RecordHit() {
	atomic.AddInt64(&t.hitCount, 1)
	t.lastUsed = time.Now()
}

// RecordMiss records a cache miss for this table.
func (t *Table) RecordMiss() {
	atomic.AddInt64(&t.missCount, 1)
	t.lastUsed = time.Now()
}

// GetStats returns statistics about this table's usage.
func (t *Table) GetStats() TableStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TableStats{
		Variant:       t.variant,
		EntryCount:    len(t.entries),
		Completed:     t.completed,
		Created:       t.created,
		LastUsed:      t.lastUsed,
		HitCount:      atomic.LoadInt64(&t.hitCount),
		MissCount:     atomic.LoadInt64(&t.missCount),
		ConsumerCount: len(t.consumers),
	}
}

// TableStats contains statistics about a table's usage and performance.
type TableStats struct {
	// Variant is the normalized goal representation
	Variant string

	// EntryCount is the number of cached solutions
	EntryCount int

	// Completed indicates if all solutions have been found
	Completed bool

	// Created is when the table was created
	Created time.Time

	// LastUsed is when the table was last accessed
	LastUsed time.Time

	// HitCount is the number of cache hits
	HitCount int64

	// MissCount is the number of cache misses
	MissCount int64

	// ConsumerCount is the number of active consumers
	ConsumerCount int
}

// String returns a human-readable representation of table statistics.
func (ts TableStats) String() string {
	return fmt.Sprintf("Table{variant=%s, entries=%d, completed=%v, hits=%d, misses=%d, consumers=%d}",
		ts.Variant, ts.EntryCount, ts.Completed, ts.HitCount, ts.MissCount, ts.ConsumerCount)
}

// VariantGenerator creates normalized representations of goals for caching.
// Variants are designed to be equivalent for goals that should share cached results.
type VariantGenerator struct{}

// NewVariantGenerator creates a new variant generator.
func NewVariantGenerator() *VariantGenerator {
	return &VariantGenerator{}
}

// GenerateVariant creates a normalized string representation of a goal.
// This representation is used as a cache key for tabling.
// The variant should be equivalent for goals that produce the same results.
func (vg *VariantGenerator) GenerateVariant(goal Goal, store ConstraintStore) string {
	// For now, use a simple hash of the goal function pointer and store state
	// In a more sophisticated implementation, this would analyze the goal structure
	goalPtr := fmt.Sprintf("%p", goal)
	storeHash := vg.hashConstraintStore(store)

	combined := goalPtr + "|" + storeHash
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter keys
}

// hashConstraintStore creates a hash of the constraint store's relevant state.
// This includes variable bindings and active constraints.
func (vg *VariantGenerator) hashConstraintStore(store ConstraintStore) string {
	// Get substitution bindings
	sub := store.GetSubstitution()

	// Sort variable IDs for consistent hashing
	varIDs := make([]int64, 0)
	sub.mu.RLock()
	for id := range sub.bindings {
		varIDs = append(varIDs, id)
	}
	sub.mu.RUnlock()

	sort.Slice(varIDs, func(i, j int) bool { return varIDs[i] < varIDs[j] })

	bindings := make([]string, 0, len(varIDs))
	sub.mu.RLock()
	for _, id := range varIDs {
		term := sub.bindings[id]
		bindings = append(bindings, fmt.Sprintf("%d=%s", id, term.String()))
	}
	sub.mu.RUnlock()

	// Include active constraints in hash
	constraints := store.GetConstraints()
	constraintStrs := make([]string, len(constraints))
	for i, constraint := range constraints {
		constraintStrs[i] = constraint.String()
	}
	sort.Strings(constraintStrs)

	combined := fmt.Sprintf("bindings:{%s}|constraints:{%s}",
		fmt.Sprintf("%v", bindings),
		fmt.Sprintf("%v", constraintStrs))

	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash[:8]) // Shorter hash for constraints
}

// TableManager manages the lifecycle of tables in the tabling system.
// It handles table creation, lookup, invalidation, and memory management.
type TableManager struct {
	mu sync.RWMutex

	// tables maps variant keys to their corresponding tables
	tables map[string]*Table

	// maxTables limits the number of active tables to prevent memory exhaustion
	maxTables int

	// maxTableSize limits the number of entries per table
	maxTableSize int

	// ttl specifies how long tables can remain unused before eviction
	ttl time.Duration

	// variantGenerator creates cache keys for goals
	variantGenerator *VariantGenerator

	// stats tracks overall tabling system statistics
	stats TableManagerStats
}

// TableManagerStats contains global statistics for the table manager.
type TableManagerStats struct {
	// TotalTables is the current number of active tables
	TotalTables int

	// TotalEntries is the total number of cached entries across all tables
	TotalEntries int

	// TotalHits is the total number of cache hits across all tables
	TotalHits int64

	// TotalMisses is the total number of cache misses across all tables
	TotalMisses int64

	// TablesEvicted is the number of tables evicted due to memory limits
	TablesEvicted int64

	// EntriesEvicted is the number of entries evicted due to size limits
	EntriesEvicted int64
}

// NewTableManager creates a new table manager with default settings.
func NewTableManager() *TableManager {
	return NewTableManagerWithConfig(1000, 10000, 30*time.Minute)
}

// NewTableManagerWithConfig creates a table manager with custom configuration.
func NewTableManagerWithConfig(maxTables, maxTableSize int, ttl time.Duration) *TableManager {
	return &TableManager{
		tables:           make(map[string]*Table),
		maxTables:        maxTables,
		maxTableSize:     maxTableSize,
		ttl:              ttl,
		variantGenerator: NewVariantGenerator(),
	}
}

// GetOrCreateTable retrieves an existing table for the given goal and store,
// or creates a new one if it doesn't exist.
func (tm *TableManager) GetOrCreateTable(goal Goal, store ConstraintStore) *Table {
	variant := tm.variantGenerator.GenerateVariant(goal, store)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if table already exists
	if table, exists := tm.tables[variant]; exists {
		table.RecordHit()
		tm.stats.TotalHits++
		return table
	}

	// Create new table
	table := NewTable(variant)
	tm.tables[variant] = table
	tm.stats.TotalMisses++

	// Evict old tables if we exceed the limit
	if len(tm.tables) > tm.maxTables {
		tm.evictOldestTable()
	}

	return table
}

// GetTable retrieves an existing table without creating a new one.
func (tm *TableManager) GetTable(goal Goal, store ConstraintStore) (*Table, bool) {
	variant := tm.variantGenerator.GenerateVariant(goal, store)

	tm.mu.RLock()
	table, exists := tm.tables[variant]
	tm.mu.RUnlock()

	if exists {
		table.RecordHit()
		tm.mu.Lock()
		tm.stats.TotalHits++
		tm.mu.Unlock()
	}

	return table, exists
}

// RemoveTable removes a table from the manager.
func (tm *TableManager) RemoveTable(variant string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tables[variant]; exists {
		delete(tm.tables, variant)
		return true
	}
	return false
}

// Clear removes all tables from the manager.
func (tm *TableManager) Clear() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tables = make(map[string]*Table)
	tm.stats = TableManagerStats{}
}

// GetStats returns current statistics about the table manager.
func (tm *TableManager) GetStats() TableManagerStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := tm.stats
	stats.TotalTables = len(tm.tables)

	totalEntries := 0
	for _, table := range tm.tables {
		totalEntries += len(table.entries)
	}
	stats.TotalEntries = totalEntries

	return stats
}

// ListTables returns statistics for all active tables.
func (tm *TableManager) ListTables() []TableStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := make([]TableStats, 0, len(tm.tables))
	for _, table := range tm.tables {
		stats = append(stats, table.GetStats())
	}
	return stats
}

// evictOldestTable removes the least recently used table to free memory.
func (tm *TableManager) evictOldestTable() {
	var oldestVariant string
	var oldestTime time.Time

	for variant, table := range tm.tables {
		if oldestVariant == "" || table.lastUsed.Before(oldestTime) {
			oldestVariant = variant
			oldestTime = table.lastUsed
		}
	}

	if oldestVariant != "" {
		delete(tm.tables, oldestVariant)
		tm.stats.TablesEvicted++
	}
}

// Cleanup removes expired tables based on TTL.
func (tm *TableManager) Cleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	for variant, table := range tm.tables {
		if now.Sub(table.lastUsed) > tm.ttl {
			delete(tm.tables, variant)
			tm.stats.TablesEvicted++
		}
	}
}
