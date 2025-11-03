// Package minikanren provides SLG (Linear resolution with Selection function for General logic programs)
// tabling infrastructure for terminating recursive queries and improving performance through memoization.
//
// # What is Tabling?
//
// Tabling (also called tabulation or memoization for logic programs) is a technique that:
//   - Prevents infinite loops in recursive relations by detecting and resolving cycles
//   - Improves performance by caching and reusing intermediate results
//   - Enables negation through stratification and well-founded semantics
//   - Guarantees termination for a broad class of programs
//
// # SLG Resolution
//
// SLG combines:
//   - SLD resolution (standard Prolog/miniKanren evaluation)
//   - Tabling to handle recursion through fixpoint computation
//   - Well-Founded Semantics for stratified negation
//
// # Architecture
//
// The tabling infrastructure uses lock-free data structures for parallel evaluation:
//   - AnswerTrie: Stores answer substitutions with structural sharing
//   - SubgoalTable: Maps call patterns to cached results using sync.Map
//   - CallPattern: Normalized representation of subgoal calls for efficient lookup
//
// All data structures are designed for concurrent access and follow the same
// copy-on-write and pooling patterns as the core solver (Phase 1-4).
package minikanren

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
	"sort"
	"sync"
	"sync/atomic"
)

// CallPattern represents a normalized subgoal call for use as a tabling key.
// CallPatterns must be comparable and efficiently hashable.
//
// The pattern abstracts away specific variable identities, replacing them with
// canonical positions (e.g., "path(X0, X1)" instead of "path(_42, _73)").
// This allows different calls with the same structure to share cached answers.
//
// Thread safety: CallPattern is immutable after creation.
type CallPattern struct {
	// Predicate identifier (name or unique ID)
	predicateID string

	// Canonical argument structure with variables abstracted to positions
	// Example: "X0,atom(a),X1" for args [Var(42), Atom("a"), Var(73)]
	argStructure string

	// Pre-computed hash for O(1) map lookup
	hashValue uint64
}

// NewCallPattern creates a normalized call pattern from a predicate name and arguments.
// Variables are abstracted to canonical positions (X0, X1, ...) based on first occurrence.
//
// Example:
//
//	args := []Term{NewVar(42, "x"), NewAtom("a"), NewVar(42, "x")}
//	pattern := NewCallPattern("path", args)
//	// pattern.argStructure == "X0,atom(a),X0"
func NewCallPattern(predicateID string, args []Term) *CallPattern {
	varMap := make(map[int64]int) // Maps variable IDs to canonical positions
	nextPos := 0

	structure := make([]string, len(args))
	for i, arg := range args {
		structure[i] = canonicalizeTerm(arg, varMap, &nextPos)
	}

	argStructure := joinStrings(structure, ",")
	hashValue := computeHash(predicateID, argStructure)

	return &CallPattern{
		predicateID:  predicateID,
		argStructure: argStructure,
		hashValue:    hashValue,
	}
}

// PredicateID returns the predicate identifier.
func (cp *CallPattern) PredicateID() string {
	return cp.predicateID
}

// ArgStructure returns the canonical argument structure.
func (cp *CallPattern) ArgStructure() string {
	return cp.argStructure
}

// Hash returns the pre-computed hash value for efficient map lookup.
func (cp *CallPattern) Hash() uint64 {
	return cp.hashValue
}

// Equal checks if two call patterns are structurally equal.
func (cp *CallPattern) Equal(other *CallPattern) bool {
	if other == nil {
		return false
	}
	return cp.predicateID == other.predicateID &&
		cp.argStructure == other.argStructure
}

// String returns a human-readable representation of the call pattern.
func (cp *CallPattern) String() string {
	return fmt.Sprintf("%s(%s)", cp.predicateID, cp.argStructure)
}

// canonicalizeTerm converts a term to its canonical string representation.
// Variables are mapped to canonical positions (X0, X1, ...).
func canonicalizeTerm(t Term, varMap map[int64]int, nextPos *int) string {
	if t == nil {
		return "nil"
	}

	if t.IsVar() {
		// Map variable to canonical position
		if v, ok := t.(*Var); ok {
			varID := v.ID()
			if pos, exists := varMap[varID]; exists {
				return fmt.Sprintf("X%d", pos)
			}
			pos := *nextPos
			varMap[varID] = pos
			*nextPos++
			return fmt.Sprintf("X%d", pos)
		}
	}

	// Atoms and other ground terms
	if atom, ok := t.(*Atom); ok {
		return fmt.Sprintf("atom(%v)", atom.Value())
	}

	// Pairs (cons cells)
	if pair, ok := t.(*Pair); ok {
		car := canonicalizeTerm(pair.Car(), varMap, nextPos)
		cdr := canonicalizeTerm(pair.Cdr(), varMap, nextPos)
		return fmt.Sprintf("pair(%s,%s)", car, cdr)
	}

	// Fallback for other term types
	return fmt.Sprintf("term(%s)", t.String())
}

// joinStrings efficiently concatenates strings with a separator.
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	// Pre-calculate total length to avoid reallocations
	totalLen := len(sep) * (len(parts) - 1)
	for _, s := range parts {
		totalLen += len(s)
	}

	// Build string efficiently
	result := make([]byte, 0, totalLen)
	for i, s := range parts {
		if i > 0 {
			result = append(result, sep...)
		}
		result = append(result, s...)
	}
	return string(result)
}

// computeHash computes a 64-bit hash for the call pattern.
func computeHash(predicateID, argStructure string) uint64 {
	h := sha256.New()
	h.Write([]byte(predicateID))
	h.Write([]byte{0}) // Separator
	h.Write([]byte(argStructure))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// SubgoalStatus represents the evaluation state of a tabled subgoal.
type SubgoalStatus int32

const (
	// StatusActive indicates the subgoal is currently being evaluated.
	StatusActive SubgoalStatus = iota

	// StatusComplete indicates all answers have been derived.
	StatusComplete

	// StatusFailed indicates the subgoal has no solutions.
	StatusFailed

	// StatusInvalidated indicates cached answers are stale (for incremental tabling).
	StatusInvalidated
)

// String returns a human-readable representation of the status.
func (s SubgoalStatus) String() string {
	switch s {
	case StatusActive:
		return "Active"
	case StatusComplete:
		return "Complete"
	case StatusFailed:
		return "Failed"
	case StatusInvalidated:
		return "Invalidated"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// SubgoalEntry represents a tabled subgoal with its cached answers.
//
// Thread safety:
//   - Status is accessed atomically
//   - Answer trie supports concurrent read/write
//   - Dependencies protected by RWMutex
//   - Condition variable for producer/consumer synchronization
type SubgoalEntry struct {
	// Call pattern (immutable)
	pattern *CallPattern

	// Answer trie containing all derived answers
	answers *AnswerTrie

	// Evaluator for re-evaluation during fixpoint computation
	// Stored to enable re-derivation when dependencies change
	evaluator GoalEvaluator

	// Current evaluation status (atomic access)
	status atomic.Int32

	// Dependencies for cycle detection and fixpoint computation
	dependencies []*SubgoalEntry
	dependencyMu sync.RWMutex

	// Stratification level for negation (0 = base stratum)
	stratum int

	// Condition variable for answer availability signaling
	answerCond *sync.Cond
	answerMu   sync.Mutex

	// Statistics (atomic counters)
	consumptionCount atomic.Int64 // Times answers were read
	derivationCount  atomic.Int64 // Times new answers were added

	// Reference count for memory management
	refCount atomic.Int64

	// WFS metadata: delay sets per answer index (conditional answers)
	// Maps answer index (insertion order) to DelaySet
	// Protected by metadataMu for thread-safe access
	answerMetadata map[int]DelaySet
	metadataMu     sync.RWMutex

	// Pending metadata for the next answer to be inserted
	// Evaluators queue metadata here before emitting answers
	pendingDelaySet DelaySet
	pendingMu       sync.Mutex

	// Event channel: closed on any answer insertion or status change,
	// then replaced with a new channel. Enables non-blocking observers
	// to detect immediate changes without polling or sleeps.
	eventMu sync.Mutex
	eventCh chan struct{}

	// Monotonic change sequence incremented on every signalEvent.
	// Used with WaitChangeSince for race-free subscription.
	changeSeq atomic.Uint64

	// Producer start signal: closed exactly once when the producer goroutine
	// for this entry has started and is ready to emit answers or status changes.
	startMu    sync.Mutex
	startedCh  chan struct{}
	startFired bool

	// Retracted answers (by index). Retracted answers are hidden from
	// WFS-aware iterators but remain in the underlying trie to preserve
	// insertion order and avoid structural mutation.
	retracted map[int]struct{}
}

// NewSubgoalEntry creates a new subgoal entry with the given call pattern.
func NewSubgoalEntry(pattern *CallPattern) *SubgoalEntry {
	entry := &SubgoalEntry{
		pattern:        pattern,
		answers:        NewAnswerTrie(),
		dependencies:   make([]*SubgoalEntry, 0, 4),
		stratum:        0,
		answerMetadata: make(map[int]DelaySet),
	}
	entry.answerCond = sync.NewCond(&entry.answerMu)
	entry.status.Store(int32(StatusActive))
	entry.refCount.Store(1)
	// Initialize event channel
	entry.eventCh = make(chan struct{})
	// Initialize start channel
	entry.startedCh = make(chan struct{})
	entry.retracted = make(map[int]struct{})
	return entry
}

// Pattern returns the call pattern for this subgoal.
func (se *SubgoalEntry) Pattern() *CallPattern {
	return se.pattern
}

// Answers returns the answer trie.
func (se *SubgoalEntry) Answers() *AnswerTrie {
	return se.answers
}

// Status returns the current evaluation status.
func (se *SubgoalEntry) Status() SubgoalStatus {
	return SubgoalStatus(se.status.Load())
}

// SetStatus updates the evaluation status.
func (se *SubgoalEntry) SetStatus(status SubgoalStatus) {
	se.status.Store(int32(status))
}

// Event returns a read-only channel that will be closed upon the next
// answer insertion or status change. After being closed, a new channel
// will be created for subsequent events.
func (se *SubgoalEntry) Event() <-chan struct{} {
	se.eventMu.Lock()
	ch := se.eventCh
	se.eventMu.Unlock()
	return ch
}

// EventSeq returns the current change sequence.
// Each call to signalEvent() increments this value.
func (se *SubgoalEntry) EventSeq() uint64 {
	return se.changeSeq.Load()
}

// WaitChangeSince returns a channel that will be closed when a change
// occurs with sequence strictly greater than 'since'. This method is
// race-free: it registers for the current event channel under lock and
// re-checks the sequence while still holding the lock to avoid missing
// an event between registration and check. If a change has already
// occurred (EventSeq() > since), it returns an already-closed channel.
func (se *SubgoalEntry) WaitChangeSince(since uint64) <-chan struct{} {
	se.eventMu.Lock()
	// If change already occurred, return a closed channel immediately
	if se.changeSeq.Load() > since {
		se.eventMu.Unlock()
		done := make(chan struct{})
		close(done)
		return done
	}
	ch := se.eventCh
	se.eventMu.Unlock()
	return ch
}

// signalEvent closes the current event channel (if not already closed)
// and replaces it with a new channel to signal future events.
func (se *SubgoalEntry) signalEvent() {
	se.eventMu.Lock()
	// Increment change sequence under the same lock to maintain ordering
	se.changeSeq.Add(1)
	// Safely close current channel (recover if already closed)
	defer func() {
		// Replace with a fresh channel for the next event
		se.eventCh = make(chan struct{})
		se.eventMu.Unlock()
	}()
	// Close current event channel
	select {
	case <-se.eventCh:
		// already closed, do nothing (will be replaced in defer)
	default:
		close(se.eventCh)
	}
}

// Started returns a channel that is closed when the producer goroutine for this
// subgoal has started. It is closed exactly once.
func (se *SubgoalEntry) Started() <-chan struct{} {
	se.startMu.Lock()
	ch := se.startedCh
	se.startMu.Unlock()
	return ch
}

// signalStarted closes the startedCh if not already closed.
func (se *SubgoalEntry) signalStarted() {
	se.startMu.Lock()
	if !se.startFired {
		close(se.startedCh)
		se.startFired = true
	}
	se.startMu.Unlock()
}

// AddDependency records that this subgoal depends on another.
func (se *SubgoalEntry) AddDependency(other *SubgoalEntry) {
	se.dependencyMu.Lock()
	defer se.dependencyMu.Unlock()
	se.dependencies = append(se.dependencies, other)
	other.refCount.Add(1) // Retain reference
}

// Dependencies returns a snapshot of current dependencies.
func (se *SubgoalEntry) Dependencies() []*SubgoalEntry {
	se.dependencyMu.RLock()
	defer se.dependencyMu.RUnlock()
	result := make([]*SubgoalEntry, len(se.dependencies))
	copy(result, se.dependencies)
	return result
}

// ConsumptionCount returns the number of times answers were consumed.
func (se *SubgoalEntry) ConsumptionCount() int64 {
	return se.consumptionCount.Load()
}

// DerivationCount returns the number of answers derived.
func (se *SubgoalEntry) DerivationCount() int64 {
	return se.derivationCount.Load()
}

// Retain increments the reference count.
func (se *SubgoalEntry) Retain() {
	se.refCount.Add(1)
}

// Release decrements the reference count and returns true if it reaches zero.
func (se *SubgoalEntry) Release() bool {
	newCount := se.refCount.Add(-1)
	if newCount < 0 {
		panic("SubgoalEntry: negative reference count")
	}
	return newCount == 0
}

// AttachDelaySet associates a DelaySet with the answer at the given index.
// This marks the answer as conditional on the resolution of the dependencies
// in the delay set. Thread-safe for concurrent access.
//
// If ds is nil or empty, the answer remains unconditional.
func (se *SubgoalEntry) AttachDelaySet(answerIndex int, ds DelaySet) {
	if ds == nil || ds.Empty() {
		return // No delay set to attach
	}
	se.metadataMu.Lock()
	defer se.metadataMu.Unlock()
	// Copy to prevent external mutation
	dsCopy := make(DelaySet, len(ds))
	for k := range ds {
		dsCopy[k] = struct{}{}
	}
	se.answerMetadata[answerIndex] = dsCopy
}

// DelaySetFor retrieves the DelaySet for the answer at the given index.
// Returns nil if the answer is unconditional or the index is out of range.
// Thread-safe for concurrent access.
func (se *SubgoalEntry) DelaySetFor(answerIndex int) DelaySet {
	se.metadataMu.RLock()
	defer se.metadataMu.RUnlock()
	if ds, ok := se.answerMetadata[answerIndex]; ok {
		// Return a copy to prevent external mutation
		dsCopy := make(DelaySet, len(ds))
		for k := range ds {
			dsCopy[k] = struct{}{}
		}
		return dsCopy
	}
	return nil
}

// AnswerRecords returns an iterator over AnswerRecord (bindings + delay sets).
// This is the WFS-aware iterator that provides metadata for conditional answers.
func (se *SubgoalEntry) AnswerRecords() *AnswerRecordIterator {
	delayProvider := func(index int) DelaySet {
		return se.DelaySetFor(index)
	}
	include := func(index int) bool { return !se.IsRetracted(index) }
	return NewAnswerRecordIterator(se.answers, delayProvider).WithInclude(include)
}

// AnswerRecordsFrom returns a WFS-aware iterator starting at the given index.
func (se *SubgoalEntry) AnswerRecordsFrom(start int) *AnswerRecordIterator {
	delayProvider := func(index int) DelaySet {
		return se.DelaySetFor(index)
	}
	include := func(index int) bool { return !se.IsRetracted(index) }
	return NewAnswerRecordIteratorFrom(se.answers, start, delayProvider).WithInclude(include)
}

// QueueDelaySetForNextAnswer queues a DelaySet to be attached to the next
// answer inserted into this entry's answer trie. This allows evaluators to
// associate metadata with answers they are about to emit.
// Thread-safe for concurrent access.
func (se *SubgoalEntry) QueueDelaySetForNextAnswer(ds DelaySet) {
	se.pendingMu.Lock()
	defer se.pendingMu.Unlock()
	if ds != nil && !ds.Empty() {
		// Copy to prevent external mutation
		dsCopy := make(DelaySet, len(ds))
		for k := range ds {
			dsCopy[k] = struct{}{}
		}
		se.pendingDelaySet = dsCopy
	}
}

// consumePendingDelaySet retrieves and clears any queued DelaySet.
// Called by the producer after inserting an answer.
// Thread-safe for concurrent access.
func (se *SubgoalEntry) consumePendingDelaySet() DelaySet {
	se.pendingMu.Lock()
	defer se.pendingMu.Unlock()
	ds := se.pendingDelaySet
	se.pendingDelaySet = nil
	return ds
}

// RetractAnswer marks the answer at the given index as retracted (invisible).
// Thread-safe for concurrent access with metadata operations.
func (se *SubgoalEntry) RetractAnswer(index int) {
	se.metadataMu.Lock()
	se.retracted[index] = struct{}{}
	se.metadataMu.Unlock()
	se.signalEvent()
}

// IsRetracted reports whether the answer at the given index is retracted.
func (se *SubgoalEntry) IsRetracted(index int) bool {
	se.metadataMu.RLock()
	_, ok := se.retracted[index]
	se.metadataMu.RUnlock()
	return ok
}

// SimplifyDelaySets removes the provided child dependency from all delay sets
// in this entry. Returns two booleans: anyChanged indicates any DS was modified;
// stillDepends indicates whether any delay set in this entry still references child.
func (se *SubgoalEntry) SimplifyDelaySets(child uint64) (anyChanged bool, stillDepends bool) {
	se.metadataMu.Lock()
	for idx, ds := range se.answerMetadata {
		if ds == nil {
			continue
		}
		if _, ok := ds[child]; ok {
			anyChanged = true
			delete(ds, child)
			if len(ds) == 0 {
				delete(se.answerMetadata, idx)
			} else {
				se.answerMetadata[idx] = ds
				stillDepends = true
			}
		}
	}
	// Check if any DS still contains child
	if !stillDepends {
		for _, ds := range se.answerMetadata {
			if ds != nil {
				if _, ok := ds[child]; ok {
					stillDepends = true
					break
				}
			}
		}
	}
	se.metadataMu.Unlock()
	if anyChanged {
		se.signalEvent()
	}
	return anyChanged, stillDepends
}

// RetractByChild retracts all answers whose delay set contains the given child.
// Returns the count of answers retracted.
func (se *SubgoalEntry) RetractByChild(child uint64) int {
	count := 0
	se.metadataMu.Lock()
	for idx, ds := range se.answerMetadata {
		if ds != nil {
			if _, ok := ds[child]; ok {
				se.retracted[idx] = struct{}{}
				count++
			}
		}
	}
	se.metadataMu.Unlock()
	if count > 0 {
		se.signalEvent()
	}
	return count
}

// SubgoalTable manages all tabled subgoals using a concurrent map.
//
// Thread safety: Uses sync.Map for lock-free concurrent access.
// The map is read-heavy (many lookups, few insertions), making sync.Map ideal.
type SubgoalTable struct {
	// Maps call pattern hash to SubgoalEntry
	entries sync.Map // map[uint64]*SubgoalEntry

	// Total subgoals created (for statistics)
	totalSubgoals atomic.Int64
}

// NewSubgoalTable creates an empty subgoal table.
func NewSubgoalTable() *SubgoalTable {
	return &SubgoalTable{}
}

// GetOrCreate retrieves an existing subgoal entry or creates a new one.
// Returns the entry and a boolean indicating if it was newly created.
//
// Thread safety: Uses sync.Map.LoadOrStore for atomic get-or-create.
func (st *SubgoalTable) GetOrCreate(pattern *CallPattern) (*SubgoalEntry, bool) {
	hash := pattern.Hash()

	// Try to load existing entry
	if value, ok := st.entries.Load(hash); ok {
		if entry, ok := value.(*SubgoalEntry); ok {
			return entry, false
		}
	}

	// Create new entry
	newEntry := NewSubgoalEntry(pattern)

	// Atomically insert if not present
	actual, loaded := st.entries.LoadOrStore(hash, newEntry)
	if loaded {
		// Another goroutine created it first
		return actual.(*SubgoalEntry), false
	}

	// We created it
	st.totalSubgoals.Add(1)
	return newEntry, true
}

// Get retrieves an existing subgoal entry by call pattern.
// Returns nil if not found.
func (st *SubgoalTable) Get(pattern *CallPattern) *SubgoalEntry {
	if value, ok := st.entries.Load(pattern.Hash()); ok {
		return value.(*SubgoalEntry)
	}
	return nil
}

// GetByHash retrieves an existing subgoal entry by hash.
// Returns nil if not found.
func (st *SubgoalTable) GetByHash(hash uint64) *SubgoalEntry {
	if value, ok := st.entries.Load(hash); ok {
		return value.(*SubgoalEntry)
	}
	return nil
}

// AllEntries returns a snapshot of all subgoal entries.
// This is an O(n) operation used for debugging and statistics.
func (st *SubgoalTable) AllEntries() []*SubgoalEntry {
	entries := make([]*SubgoalEntry, 0, 16)
	st.entries.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*SubgoalEntry); ok {
			entries = append(entries, entry)
		}
		return true
	})
	return entries
}

// Clear removes all entries from the table.
func (st *SubgoalTable) Clear() {
	st.entries.Range(func(key, value interface{}) bool {
		st.entries.Delete(key)
		return true
	})
	st.totalSubgoals.Store(0)
}

// TotalSubgoals returns the total number of subgoals created.
func (st *SubgoalTable) TotalSubgoals() int64 {
	return st.totalSubgoals.Load()
}

// AnswerTrie represents a trie of answer substitutions for a tabled subgoal.
// Uses structural sharing to minimize memory overhead.
//
// Thread safety: The trie supports concurrent reads, and writes are coordinated
// via an internal mutex to ensure safety. Iteration returns copies of stored
// answers to prevent external mutation. In typical usage, writes are also
// coordinated at a higher level (e.g., by SubgoalEntry) to avoid unnecessary
// contention.
type AnswerTrie struct {
	// Root node of the trie
	root *AnswerTrieNode

	// Ordered list of answers for deterministic iteration
	answers []map[int64]Term

	// Cached answer count for O(1) size queries
	count atomic.Int64

	// Pool for trie nodes (zero-allocation reuse)
	nodePool *sync.Pool

	// Mutex for coordinating insertions
	mu sync.Mutex
}

// NewAnswerTrie creates an empty answer trie.
func NewAnswerTrie() *AnswerTrie {
	return &AnswerTrie{
		root: &AnswerTrieNode{
			varID:    -1, // Sentinel for root
			children: make(map[nodeKey]*AnswerTrieNode),
		},
		nodePool: &sync.Pool{
			New: func() interface{} {
				return &AnswerTrieNode{
					children: make(map[nodeKey]*AnswerTrieNode),
				}
			},
		},
	}
}

// nodeKey represents a (varID, value) pair for trie indexing.
type nodeKey struct {
	varID     int64
	valueHash uint64 // Hash of the bound value
}

// AnswerTrieNode represents a node in the answer trie.
// Thread safety: children map is protected by the trie's global mutex during writes,
// and is safe for concurrent reads after insertion since nodes are structurally shared.
type AnswerTrieNode struct {
	// Variable ID at this level (-1 for root)
	varID int64

	// Bound value at this node (nil if unbound)
	value Term

	// Children indexed by (varID, valueHash) pairs
	// Protected by trie-level mutex during modifications
	children map[nodeKey]*AnswerTrieNode

	// Marks this as a complete answer (leaf node)
	isAnswer bool

	// Depth in trie (for debugging)
	depth int
}

// Insert adds an answer to the trie.
// Returns true if the answer was new, false if it was a duplicate.
//
// Answers are represented as variable bindings. The trie is organized by
// variable ID, with each path representing a complete answer.
func (at *AnswerTrie) Insert(bindings map[int64]Term) bool {
	at.mu.Lock()
	defer at.mu.Unlock()

	// Sort variable IDs for consistent trie structure
	varIDs := make([]int64, 0, len(bindings))
	for varID := range bindings {
		varIDs = append(varIDs, varID)
	}
	sort.Slice(varIDs, func(i, j int) bool { return varIDs[i] < varIDs[j] })

	// Traverse/build trie path
	current := at.root
	for _, varID := range varIDs {
		value := bindings[varID]
		key := nodeKey{
			varID:     varID,
			valueHash: hashTerm(value),
		}

		child, exists := current.children[key]
		if !exists {
			// Create new node
			child = at.nodePool.Get().(*AnswerTrieNode)
			child.varID = varID
			child.value = value
			child.isAnswer = false
			child.depth = current.depth + 1
			if child.children == nil {
				child.children = make(map[nodeKey]*AnswerTrieNode)
			}
			current.children[key] = child
		}
		current = child
	}

	// Mark leaf as answer
	if current.isAnswer {
		return false // Duplicate answer
	}
	current.isAnswer = true

	// Store answer in insertion order for deterministic iteration
	answerCopy := make(map[int64]Term, len(bindings))
	for k, v := range bindings {
		answerCopy[k] = v
	}
	at.answers = append(at.answers, answerCopy)

	at.count.Add(1)
	return true
}

// Count returns the number of answers in the trie.
func (at *AnswerTrie) Count() int64 {
	return at.count.Load()
}

// Iterator returns an iterator over all answers in the trie.
// Answers are returned in insertion order for deterministic iteration.
// The iterator creates a snapshot of the answer list to avoid
// concurrent modification issues during iteration.
func (at *AnswerTrie) Iterator() *AnswerIterator {
	// Take a snapshot of current answers under the trie's lock
	at.mu.Lock()
	snapshot := make([]map[int64]Term, len(at.answers))
	copy(snapshot, at.answers)
	at.mu.Unlock()

	return &AnswerIterator{
		snapshot: snapshot,
		idx:      0,
	}
}

// IteratorFrom returns an iterator starting at the given index over a snapshot
// of the current answers. If start >= len(snapshot), the iterator is exhausted.
// Use this to resume iteration without re-reading already-consumed answers
// when new answers may have been appended concurrently.
func (at *AnswerTrie) IteratorFrom(start int) *AnswerIterator {
	at.mu.Lock()
	snapshot := make([]map[int64]Term, len(at.answers))
	copy(snapshot, at.answers)
	at.mu.Unlock()

	if start < 0 {
		start = 0
	}
	if start > len(snapshot) {
		start = len(snapshot)
	}

	return &AnswerIterator{
		snapshot: snapshot,
		idx:      start,
	}
}

// AnswerIterator iterates over answers in insertion order.
type AnswerIterator struct {
	// snapshot holds a point-in-time copy of the trie's answer slice headers.
	// Individual answers are still copied on return by Next() to prevent external mutation.
	// To observe new answers appended after iterator creation, construct a new iterator.
	snapshot []map[int64]Term
	idx      int
	mu       sync.Mutex // Protects idx
}

// Next returns the next answer or nil if exhausted.
// Thread safety: This method uses internal locks and is safe to call from
// multiple goroutines, but using a single goroutine per iterator preserves
// deterministic ordering and minimizes contention.
func (ai *AnswerIterator) Next() (map[int64]Term, bool) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	if ai.idx >= len(ai.snapshot) {
		return nil, false
	}

	// Return a copy to prevent external mutation of stored answers
	stored := ai.snapshot[ai.idx]
	answer := make(map[int64]Term, len(stored))
	for k, v := range stored {
		answer[k] = v
	}
	ai.idx++
	return answer, true
}

// hashTerm computes a hash for a term.
func hashTerm(t Term) uint64 {
	if t == nil {
		return 0
	}

	var h hash.Hash = sha256.New()

	if t.IsVar() {
		if v, ok := t.(*Var); ok {
			h.Write([]byte("var"))
			binary.Write(h, binary.BigEndian, v.ID())
		}
	} else if atom, ok := t.(*Atom); ok {
		h.Write([]byte("atom"))
		h.Write([]byte(fmt.Sprintf("%v", atom.Value())))
	} else if pair, ok := t.(*Pair); ok {
		h.Write([]byte("pair"))
		carHash := hashTerm(pair.Car())
		cdrHash := hashTerm(pair.Cdr())
		binary.Write(h, binary.BigEndian, carHash)
		binary.Write(h, binary.BigEndian, cdrHash)
	} else {
		h.Write([]byte(t.String()))
	}

	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}
