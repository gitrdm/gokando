// Package minikanren provides SLG (Linear resolution with Selection function for General logic programs)
// resolution engine for tabled evaluation of recursive queries.
//
// # SLG Resolution
//
// SLG resolution extends standard SLD resolution (Prolog/miniKanren) with tabling to:
//   - Detect and resolve cycles in recursive predicates
//   - Compute fixpoints for mutually recursive relations
//   - Cache intermediate results for reuse
//   - Guarantee termination for a broad class of programs
//
// # Architecture
//
// The SLG engine coordinates:
//   - Producer goroutines that evaluate goals and derive new answers
//   - Consumer goroutines that read cached answers as they become available
//   - Cycle detection using Tarjan's SCC algorithm on the dependency graph
//   - Fixpoint computation for strongly connected components
//
// # Thread Safety
//
// The engine is designed for concurrent access:
//   - SubgoalTable uses sync.Map for lock-free lookups
//   - Answer insertion is synchronized via mutex in AnswerTrie
//   - Producer/consumer coordination uses sync.Cond for efficient signaling
//   - Context cancellation propagates cleanly to all goroutines
package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SLGEngine coordinates tabled goal evaluation using SLG resolution.
//
// The engine maintains a global SubgoalTable shared across all evaluations,
// enabling answer reuse and cycle detection. Multiple goroutines can safely
// evaluate different goals concurrently.
//
// Thread safety: SLGEngine is safe for concurrent use by multiple goroutines.
type SLGEngine struct {
	// Global subgoal table (shared across all evaluations)
	subgoals *SubgoalTable

	// Configuration
	config *SLGConfig

	// Statistics (atomic counters)
	totalEvaluations atomic.Int64
	totalAnswers     atomic.Int64
	cacheHits        atomic.Int64
	cacheMisses      atomic.Int64

	// Mutex for engine-level operations
	mu sync.RWMutex

	// Optional stratification map: predicateID -> stratum (0 = base)
	strataMu sync.RWMutex
	strata   map[string]int

	// Reverse dependency index: child pattern hash -> set of parent entries
	reverseDeps sync.Map // map[uint64]*parentSet

	// Dependency graph (for unfounded set detection): parent -> child edges with polarity.
	depMu  sync.RWMutex
	depAdj map[uint64]map[uint64]*edgePolarity

	// Cached set of nodes detected as part of SCCs containing a negative edge
	negMu        sync.RWMutex
	negUndefined map[uint64]bool

	// Predicate tracking: maps predicateID to set of subgoal entry hashes
	predicateMu      sync.RWMutex
	predicateEntries map[string]map[uint64]struct{}
}

type parentSet struct {
	mu  sync.RWMutex
	set map[*SubgoalEntry]struct{}
}

// edgePolarity tracks whether a dependency edge is positive and/or negative.
// An edge can be both (if established via different paths). We mark both flags accordingly.
type edgePolarity struct{ pos, neg bool }

func (ps *parentSet) add(p *SubgoalEntry) {
	ps.mu.Lock()
	if ps.set == nil {
		ps.set = make(map[*SubgoalEntry]struct{})
	}
	ps.set[p] = struct{}{}
	ps.mu.Unlock()
}
func (ps *parentSet) remove(p *SubgoalEntry) {
	ps.mu.Lock()
	delete(ps.set, p)
	ps.mu.Unlock()
}
func (ps *parentSet) snapshot() []*SubgoalEntry {
	ps.mu.RLock()
	out := make([]*SubgoalEntry, 0, len(ps.set))
	for k := range ps.set {
		out = append(out, k)
	}
	ps.mu.RUnlock()
	return out
}

func (e *SLGEngine) addReverseDependency(child uint64, parent *SubgoalEntry) {
	value, _ := e.reverseDeps.LoadOrStore(child, &parentSet{set: make(map[*SubgoalEntry]struct{})})
	ps := value.(*parentSet)
	ps.add(parent)
}

func (e *SLGEngine) removeReverseDependency(child uint64, parent *SubgoalEntry) {
	if value, ok := e.reverseDeps.Load(child); ok {
		ps := value.(*parentSet)
		ps.remove(parent)
	}
}

func (e *SLGEngine) getReverseParents(child uint64) []*SubgoalEntry {
	if value, ok := e.reverseDeps.Load(child); ok {
		return value.(*parentSet).snapshot()
	}
	return nil
}

// addPositiveEdge records a positive dependency parent->child for unfounded set analysis.
func (e *SLGEngine) addPositiveEdge(parent, child uint64) {
	e.depMu.Lock()
	adj := e.depAdj[parent]
	if adj == nil {
		adj = make(map[uint64]*edgePolarity)
		e.depAdj[parent] = adj
	}
	ep := adj[child]
	if ep == nil {
		ep = &edgePolarity{}
		adj[child] = ep
	}
	ep.pos = true
	e.depMu.Unlock()
}

// addNegativeEdge records a negative dependency parent->child for unfounded set analysis.
func (e *SLGEngine) addNegativeEdge(parent, child uint64) {
	e.depMu.Lock()
	adj := e.depAdj[parent]
	if adj == nil {
		adj = make(map[uint64]*edgePolarity)
		e.depAdj[parent] = adj
	}
	ep := adj[child]
	if ep == nil {
		ep = &edgePolarity{}
		adj[child] = ep
	}
	ep.neg = true
	e.depMu.Unlock()
	wfsTracef("addNegativeEdge: parent=%d -> child=%d (engine=%p)", parent, child, e)
	// Recompute undefined SCCs synchronously to ensure deterministic behavior.
	// The async version created race conditions where isInNegativeSCC checks
	// could happen before or after the SCC computation completed, leading to
	// flaky test failures and non-deterministic delay set attachments.
	e.computeUndefinedSCCs()
}

// computeUndefinedSCCs runs Tarjan's SCC and marks subgoals in SCCs containing
// at least one negative edge as WFS undefined.
func (e *SLGEngine) computeUndefinedSCCs() {
	// Snapshot adjacency under read lock
	e.depMu.RLock()
	adj := make(map[uint64]map[uint64]*edgePolarity, len(e.depAdj))
	for u, m := range e.depAdj {
		mm := make(map[uint64]*edgePolarity, len(m))
		for v, ep := range m {
			cp := *ep
			mm[v] = &cp
		}
		adj[u] = mm
	}
	e.depMu.RUnlock()

	// Tarjan's algorithm
	index := 0
	indices := make(map[uint64]int)
	lowlink := make(map[uint64]int)
	onstack := make(map[uint64]bool)
	stack := make([]uint64, 0, 16)

	var sccs [][]uint64
	var strongConnect func(v uint64)
	strongConnect = func(v uint64) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onstack[v] = true

		for w := range adj[v] {
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onstack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		if lowlink[v] == indices[v] {
			// start a new SCC
			var comp []uint64
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onstack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, comp)
		}
	}

	// Visit all nodes (include those only reachable as children)
	seen := make(map[uint64]bool)
	var visitAll func(u uint64)
	visitAll = func(u uint64) {
		if seen[u] {
			return
		}
		seen[u] = true
		if _, ok := indices[u]; !ok {
			strongConnect(u)
		}
		for v := range adj[u] {
			if !seen[v] {
				visitAll(v)
			}
		}
	}
	for u := range adj {
		visitAll(u)
	}

	// For each SCC, check if any internal edge is negative
	newUndefined := make(map[uint64]bool)
	for _, comp := range sccs {
		if len(comp) == 1 {
			// A self-loop counts as SCC; check for negative self-edge
			u := comp[0]
			if ep, ok := adj[u][u]; !ok || !ep.neg {
				continue
			}
		}
		// Determine if SCC contains any negative edge among members
		member := make(map[uint64]bool, len(comp))
		for _, u := range comp {
			member[u] = true
		}
		hasNeg := false
		for _, u := range comp {
			for v, ep := range adj[u] {
				if member[v] && ep.neg {
					hasNeg = true
					break
				}
			}
			if hasNeg {
				break
			}
		}
		if !hasNeg {
			continue
		}
		// Mark all members as undefined truth (in cache)
		wfsTracef("computeUndefinedSCCs: marking SCC with %d nodes as undefined", len(comp))
		for _, u := range comp {
			newUndefined[u] = true
			wfsTracef("  - marking node %d as in negative SCC", u)
		}
	}
	e.negMu.Lock()
	e.negUndefined = newUndefined
	e.negMu.Unlock()
	wfsTracef("computeUndefinedSCCs: complete, %d nodes in negative SCCs", len(newUndefined))
}

// isInNegativeSCC reports whether the given subgoal hash is currently known
// to be in an SCC that contains at least one negative edge.
func (e *SLGEngine) isInNegativeSCC(hash uint64) bool {
	e.negMu.RLock()
	_, ok := e.negUndefined[hash]
	e.negMu.RUnlock()
	return ok
}

// hasNegativeIncoming reports whether any parent has a negative edge to this node.
// This is a conservative heuristic useful when SCC computation hasn't yet converged.
func (e *SLGEngine) hasNegativeIncoming(hash uint64) bool {
	e.depMu.RLock()
	defer e.depMu.RUnlock()
	for _, children := range e.depAdj {
		if ep, ok := children[hash]; ok && ep != nil && ep.neg {
			return true
		}
	}
	return false
}

// hasNegEdgeReachableFrom reports whether starting from 'hash' and following only
// positive edges, we can reach any negative edge (u -neg-> v). This conservatively
// detects potential unfounded-set cycles involving negation reachable from the inner goal.
func (e *SLGEngine) hasNegEdgeReachableFrom(hash uint64) bool {
	e.depMu.RLock()
	defer e.depMu.RUnlock()
	// BFS over positive edges
	visited := make(map[uint64]bool)
	queue := []uint64{hash}
	visited[hash] = true
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		children := e.depAdj[u]
		if children == nil {
			continue
		}
		for v, ep := range children {
			if ep == nil {
				continue
			}
			if ep.neg {
				return true
			}
			if ep.pos && !visited[v] {
				visited[v] = true
				queue = append(queue, v)
			}
		}
	}
	return false
}

// onChildHasAnswers is invoked when a child subgoal derives its first answer.
// It retracts all conditional answers in parents that depend on this child.
func (e *SLGEngine) onChildHasAnswers(child *SubgoalEntry) {
	childHash := child.Pattern().Hash()
	parents := e.getReverseParents(childHash)
	for _, p := range parents {
		p.RetractByChild(childHash)
		// Parent no longer needs notifications for this child.
		e.removeReverseDependency(childHash, p)
	}
}

// onChildCompletedNoAnswers simplifies delay sets in dependent parents.
func (e *SLGEngine) onChildCompletedNoAnswers(child *SubgoalEntry) {
	childHash := child.Pattern().Hash()
	parents := e.getReverseParents(childHash)
	for _, p := range parents {
		_, still := p.SimplifyDelaySets(childHash)
		if !still {
			// Remove parent from reverse deps, no longer depends on child
			e.removeReverseDependency(childHash, p)
		}
	}
}

// SLGConfig holds configuration for the SLG engine.
type SLGConfig struct {
	// MaxTableSize limits the total number of subgoals (0 = unlimited)
	MaxTableSize int64

	// MaxAnswersPerSubgoal limits answers per subgoal (0 = unlimited)
	MaxAnswersPerSubgoal int64

	// MaxFixpointIterations limits iterations for cyclic computations
	MaxFixpointIterations int

	// EnableParallelProducers allows multiple producers per subgoal
	EnableParallelProducers bool

	// EnableSubsumptionChecking enables answer subsumption (future enhancement)
	EnableSubsumptionChecking bool

	// EnforceStratification controls whether negation is restricted by strata.
	// When true (default), a predicate may only negate predicates in the same or
	// lower stratum; negating a higher stratum is a violation and yields no answers.
	// When false, general WFS with unfounded-set handling applies.
	EnforceStratification bool

	// DebugWFS enables verbose tracing for WFS/negation synchronization paths.
	// Prefer enabling via environment variable gokanlogic_WFS_TRACE=1 when possible.
	DebugWFS bool

	// NegationPeekTimeout is deprecated and ignored.
	// Negation now uses a timing-free, race-free event sequence + handshake,
	// so no peek window is needed. This field is retained for backward
	// compatibility and will be removed in a future major version.
	NegationPeekTimeout time.Duration
}

// DefaultSLGConfig returns the default SLG configuration.
func DefaultSLGConfig() *SLGConfig {
	return &SLGConfig{
		MaxTableSize:              0,     // Unlimited
		MaxAnswersPerSubgoal:      10000, // Reasonable default
		MaxFixpointIterations:     1000,  // Prevent infinite loops
		EnableParallelProducers:   false, // Sequential by default
		EnableSubsumptionChecking: false, // Future enhancement
		EnforceStratification:     true,  // Enforce by default; equal stratum allowed
		// Deprecated: ignored (kept for backward compatibility)
		// Default retained to avoid surprising behavior differences
		NegationPeekTimeout: time.Millisecond,
	}
}

// NewSLGEngine creates a new SLG engine with the given configuration.
func NewSLGEngine(config *SLGConfig) *SLGEngine {
	if config == nil {
		config = DefaultSLGConfig()
	}
	if config.DebugWFS {
		enableWFSTrace()
	}
	return &SLGEngine{
		subgoals:         NewSubgoalTable(),
		config:           config,
		strata:           make(map[string]int),
		depAdj:           make(map[uint64]map[uint64]*edgePolarity),
		negUndefined:     make(map[uint64]bool),
		predicateEntries: make(map[string]map[uint64]struct{}),
	}
}

// InvalidateByDomain notifies the engine that the FD domain for a variable has changed.
// It retracts any tabled answers across all subgoals that bind varID to an integer
// value not contained in the provided domain. Returns the total number of answers
// retracted across all subgoals.
//
// Thread-safety: safe for concurrent use. Iterates over a snapshot of all entries
// and delegates to entry-level invalidation which handles its own synchronization.
func (e *SLGEngine) InvalidateByDomain(varID int64, dom Domain) int {
	if e == nil || dom == nil {
		return 0
	}
	entries := e.subgoals.AllEntries()
	total := 0
	for _, se := range entries {
		total += se.InvalidateByDomain(varID, dom)
	}
	return total
}

// Global SLG engine instance for convenience (can be overridden per-evaluation)
var globalEngine *SLGEngine
var globalEngineMu sync.RWMutex

// GlobalEngine returns the global SLG engine, creating it if necessary.
func GlobalEngine() *SLGEngine {
	globalEngineMu.RLock()
	if globalEngine != nil {
		globalEngineMu.RUnlock()
		return globalEngine
	}
	globalEngineMu.RUnlock()

	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()

	// Double-check after acquiring write lock
	if globalEngine == nil {
		globalEngine = NewSLGEngine(DefaultSLGConfig())
	}
	return globalEngine
}

// SetGlobalEngine sets the global SLG engine.
// This is useful for testing or custom configurations.
func SetGlobalEngine(engine *SLGEngine) {
	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()
	globalEngine = engine
}

// ResetGlobalEngine clears the global engine's cache and resets it.
func ResetGlobalEngine() {
	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()
	if globalEngine != nil {
		globalEngine.subgoals.Clear()
		globalEngine.totalEvaluations.Store(0)
		globalEngine.totalAnswers.Store(0)
		globalEngine.cacheHits.Store(0)
		globalEngine.cacheMisses.Store(0)
	}
}

// SLGStats provides statistics about engine performance.
type SLGStats struct {
	// Total number of subgoals evaluated
	TotalEvaluations int64

	// Total answers derived across all subgoals
	TotalAnswers int64

	// Number of cache hits (subgoal already evaluated)
	CacheHits int64

	// Number of cache misses (new subgoal evaluation)
	CacheMisses int64

	// Current number of cached subgoals
	CachedSubgoals int64

	// Cache hit ratio (hits / (hits + misses))
	HitRatio float64
}

// Stats returns current engine statistics.
func (e *SLGEngine) Stats() *SLGStats {
	hits := e.cacheHits.Load()
	misses := e.cacheMisses.Load()
	total := hits + misses

	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hits) / float64(total)
	}

	return &SLGStats{
		TotalEvaluations: e.totalEvaluations.Load(),
		TotalAnswers:     e.totalAnswers.Load(),
		CacheHits:        hits,
		CacheMisses:      misses,
		CachedSubgoals:   e.subgoals.TotalSubgoals(),
		HitRatio:         hitRatio,
	}
}

// Clear removes all cached subgoals and resets statistics.
func (e *SLGEngine) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subgoals.Clear()
	e.totalEvaluations.Store(0)
	e.totalAnswers.Store(0)
	e.cacheHits.Store(0)
	e.cacheMisses.Store(0)
	e.strataMu.Lock()
	e.strata = make(map[string]int)
	e.strataMu.Unlock()

	// Clear dependency tracking structures
	e.reverseDeps.Range(func(key, value interface{}) bool {
		e.reverseDeps.Delete(key)
		return true
	})
	e.depMu.Lock()
	e.depAdj = make(map[uint64]map[uint64]*edgePolarity)
	e.depMu.Unlock()
	e.negMu.Lock()
	e.negUndefined = make(map[uint64]bool)
	e.negMu.Unlock()

	// Clear predicate tracking
	e.predicateMu.Lock()
	e.predicateEntries = make(map[string]map[uint64]struct{})
	e.predicateMu.Unlock()
}

// registerPredicate registers a subgoal entry with the predicate tracking system.
// This should be called when a new entry is created.
func (e *SLGEngine) registerPredicate(pattern *CallPattern, hash uint64) {
	if pattern == nil {
		return
	}
	predicateID := pattern.PredicateID()

	e.predicateMu.Lock()
	defer e.predicateMu.Unlock()

	if e.predicateEntries[predicateID] == nil {
		e.predicateEntries[predicateID] = make(map[uint64]struct{})
	}
	e.predicateEntries[predicateID][hash] = struct{}{}
}

// ClearPredicate removes all cached subgoals for a specific predicate.
// This enables fine-grained invalidation when a relation's facts change.
// Returns the number of subgoal entries that were invalidated.
func (e *SLGEngine) ClearPredicate(predicateID string) int {
	e.predicateMu.Lock()
	hashes := e.predicateEntries[predicateID]
	if hashes == nil {
		e.predicateMu.Unlock()
		return 0
	}
	// Copy hashes to avoid holding the lock during deletion
	hashList := make([]uint64, 0, len(hashes))
	for hash := range hashes {
		hashList = append(hashList, hash)
	}
	// Clear the predicate's entry set
	delete(e.predicateEntries, predicateID)
	e.predicateMu.Unlock()

	// Remove each subgoal entry from the table
	count := 0
	for _, hash := range hashList {
		entry := e.subgoals.GetByHash(hash)
		if entry != nil {
			// Remove from subgoal table
			e.subgoals.Delete(hash)
			count++

			// Clean up dependency tracking for this entry
			e.removeReverseDependency(hash, entry)

			// Clean up WFS dependency graph
			e.depMu.Lock()
			delete(e.depAdj, hash)
			// Also remove as child from all parents
			for _, children := range e.depAdj {
				delete(children, hash)
			}
			e.depMu.Unlock()

			// Remove from negative SCC tracking if present
			e.negMu.Lock()
			delete(e.negUndefined, hash)
			e.negMu.Unlock()
		}
	}

	return count
}

// GoalEvaluator is a function that evaluates a goal and returns answer bindings.
// It's called by the SLG engine to produce answers for a tabled subgoal.
//
// The evaluator should:
//   - Yield answer bindings via the channel
//   - Close the channel when done
//   - Respect context cancellation
//   - Return any error encountered
type GoalEvaluator func(ctx context.Context, answers chan<- map[int64]Term) error

// stratification accessors
// SetStrata sets fixed predicate strata for WFS enforcement where lower strata
// must not depend negatively on same-or-higher strata. Keys are predicate IDs
// as used by CallPattern.PredicateID(). Missing keys default to stratum 0.
func (e *SLGEngine) SetStrata(strata map[string]int) {
	e.strataMu.Lock()
	defer e.strataMu.Unlock()
	e.strata = make(map[string]int, len(strata))
	for k, v := range strata {
		e.strata[k] = v
	}
}

// Stratum returns the configured stratum for a predicate, or 0 if unspecified.
func (e *SLGEngine) Stratum(predicateID string) int {
	e.strataMu.RLock()
	defer e.strataMu.RUnlock()
	if s, ok := e.strata[predicateID]; ok {
		return s
	}
	return 0
}

// internal context key for parent subgoal entry for dependency tracking
type slgParentKey struct{}

// Evaluate evaluates a tabled goal using SLG resolution.
//
// The process:
//  1. Normalize the call pattern
//  2. Check if subgoal exists in table (cache hit/miss)
//  3. If new: start producer to evaluate goal
//  4. If existing: consume from answer trie
//  5. Handle cycles via dependency tracking
//
// Returns a channel that yields answer bindings as they become available.
// The channel is closed when evaluation completes or context is cancelled.
//
// Thread safety: Safe for concurrent calls with different patterns.
func (e *SLGEngine) Evaluate(ctx context.Context, pattern *CallPattern, evaluator GoalEvaluator) (<-chan map[int64]Term, error) {
	if pattern == nil {
		return nil, fmt.Errorf("SLGEngine.Evaluate: nil call pattern")
	}
	if evaluator == nil {
		return nil, fmt.Errorf("SLGEngine.Evaluate: nil evaluator")
	}

	e.totalEvaluations.Add(1)

	// Check if subgoal already exists
	entry, isNew := e.subgoals.GetOrCreate(pattern)

	// Register new entry with predicate tracking
	if isNew {
		e.registerPredicate(pattern, pattern.Hash())
	}

	// If called within another evaluator, record dependency (parent -> entry)
	if parentRaw := ctx.Value(slgParentKey{}); parentRaw != nil {
		if parent, ok := parentRaw.(*SubgoalEntry); ok && parent != entry {
			parent.AddDependency(entry)
			// Positive dependency for unfounded set analysis
			e.addPositiveEdge(parent.Pattern().Hash(), entry.Pattern().Hash())
		}
	}

	if !isNew {
		// Cache hit
		e.cacheHits.Add(1)
		// If this Evaluate is being called from within the same subgoal's evaluator
		// (direct self-recursion), don't create a consumer that would block waiting
		// on answers from itself. Instead, return an immediately-closed channel so
		// the caller can proceed without deadlocking. Recursive derivations will be
		// discovered via non-recursive branches producing base answers.
		if parentRaw := ctx.Value(slgParentKey{}); parentRaw != nil {
			if parent, ok := parentRaw.(*SubgoalEntry); ok && parent == entry {
				ch := make(chan map[int64]Term)
				close(ch)
				return ch, nil
			}
		}
		return e.consumeAnswers(ctx, entry), nil
	}

	// Cache miss: need to evaluate
	e.cacheMisses.Add(1)

	// Store evaluator for potential re-evaluation during fixpoint
	entry.evaluator = evaluator

	// Start producer/consumer
	return e.produceAndConsume(ctx, entry, evaluator), nil
}

// evaluateWithHandshake evaluates a subgoal and returns:
// - the answer channel
// - the subgoal entry
// - the pre-start event sequence (captured before starting a new producer)
// - the producer started channel
// This enables race-free initial-shape decisions without timers.
func (e *SLGEngine) evaluateWithHandshake(ctx context.Context, pattern *CallPattern, evaluator GoalEvaluator) (<-chan map[int64]Term, *SubgoalEntry, uint64, <-chan struct{}, error) {
	if pattern == nil {
		return nil, nil, 0, nil, fmt.Errorf("Evaluate: nil pattern")
	}
	if evaluator == nil {
		return nil, nil, 0, nil, fmt.Errorf("Evaluate: nil evaluator")
	}

	e.totalEvaluations.Add(1)

	// Get or create subgoal entry
	entry, isNew := e.subgoals.GetOrCreate(pattern)

	// Register new entry with predicate tracking
	if isNew {
		e.registerPredicate(pattern, pattern.Hash())
	}

	// Capture pre-start sequence before potentially starting producer
	preSeq := entry.EventSeq()

	if !isNew {
		// Cache hit: consume existing answers
		e.cacheHits.Add(1)
		return e.consumeAnswers(ctx, entry), entry, preSeq, entry.Started(), nil
	}

	// Cache miss: need to evaluate
	e.cacheMisses.Add(1)

	// Store evaluator for potential re-evaluation during fixpoint
	entry.evaluator = evaluator

	// Start producer/consumer
	ch := e.produceAndConsume(ctx, entry, evaluator)
	return ch, entry, preSeq, entry.Started(), nil
}

// produceAndConsume starts producer and consumer goroutines for a new subgoal.
// Producer: evaluates goal and inserts answers into trie
// Consumer: reads from trie and streams to output channel
func (e *SLGEngine) produceAndConsume(ctx context.Context, entry *SubgoalEntry, evaluator GoalEvaluator) <-chan map[int64]Term {
	answerChan := make(chan map[int64]Term, 100) // Buffered for batching

	// Producer goroutine: evaluate and populate trie
	go func() {

		// Channel for evaluator to send answers
		producerChan := make(chan map[int64]Term, 100)

		// Run evaluator in separate goroutine
		var evalErr error
		evalDone := make(chan struct{})
		go func() {
			defer close(producerChan)
			// Provide child context carrying:
			// - current entry for dependency tracking (slgParentKey)
			// - current entry for metadata attachment (slgProducerEntryKey)
			childCtx := context.WithValue(ctx, slgParentKey{}, entry)
			childCtx = context.WithValue(childCtx, slgProducerEntryKey{}, entry)
			evalErr = evaluator(childCtx, producerChan)
			close(evalDone)
		}()

		// Immediate, race-free first check: handle an immediate answer or completion
		select {
		case <-ctx.Done():
			entry.SetStatus(StatusFailed)
			entry.signalEvent()
			entry.answerCond.Broadcast()
			entry.signalStarted() // Unblock any waiters
			return
		case answer, ok := <-producerChan:
			if !ok {
				// Evaluator finished immediately
				<-evalDone // Ensure evalErr is set
				if evalErr != nil {
					entry.SetStatus(StatusFailed)
				} else {
					entry.SetStatus(StatusComplete)
				}
				entry.signalEvent()
				entry.answerCond.Broadcast()
				if entry.Answers().Count() == 0 && evalErr == nil {
					go e.onChildCompletedNoAnswers(entry)
				}
				entry.signalStarted()
				return
			}

			// Received an immediate answer
			wasNew, _ := entry.InsertAnswerWithSubsumption(answer)
			if wasNew {
				entry.derivationCount.Add(1)
				e.totalAnswers.Add(1)

				entry.signalEvent()
				entry.answerCond.Broadcast()

				if entry.Answers().Count() == 1 {
					go e.onChildHasAnswers(entry)
				}
			}
			// Producer is active; signal started and continue to normal loop
			entry.signalStarted()
		default:
			// No immediate outcome; producer is started
			entry.signalStarted()
		}

		// Receive answers from evaluator and insert into trie
		for {
			select {
			case <-ctx.Done():
				entry.SetStatus(StatusFailed)
				// Signal event for failure
				entry.signalEvent()
				entry.answerCond.Broadcast()
				return

			case answer, ok := <-producerChan:
				if !ok {
					// Producer finished
					<-evalDone // Wait for evaluator to complete
					if evalErr != nil {
						entry.SetStatus(StatusFailed)
					} else {
						entry.SetStatus(StatusComplete)
					}
					// Signal event for completion
					entry.signalEvent()
					entry.answerCond.Broadcast()
					// If child completed with no answers, simplify dependents
					if entry.Answers().Count() == 0 && evalErr == nil {
						go e.onChildCompletedNoAnswers(entry)
					}
					return
				}

				// Insert into answer trie with subsumption
				wasNew, _ := entry.InsertAnswerWithSubsumption(answer)
				if wasNew {
					entry.derivationCount.Add(1)
					e.totalAnswers.Add(1)

					// Signal event for new answer
					entry.signalEvent()
					// Notify waiting consumers
					entry.answerCond.Broadcast()

					// If this is the first answer for this child, trigger retraction in dependents
					if entry.Answers().Count() == 1 {
						go e.onChildHasAnswers(entry)
					}
				}

				// Check answer limit
				if e.config.MaxAnswersPerSubgoal > 0 &&
					entry.Answers().Count() >= e.config.MaxAnswersPerSubgoal {
					entry.SetStatus(StatusComplete)
					// Signal event for completion
					entry.signalEvent()
					entry.answerCond.Broadcast()
					if entry.Answers().Count() == 0 {
						go e.onChildCompletedNoAnswers(entry)
					}
					return
				}
			}
		}
	}()

	// Consumer goroutine: stream answers from trie
	go func() {
		defer close(answerChan)

		iter := entry.Answers().Iterator()
		var lastCount int64 = 0

		for {
			// Drain current snapshot
			for {
				answer, ok := iter.Next()
				if !ok {
					break
				}
				lastCount++
				entry.consumptionCount.Add(1)

				// Skip retracted answers (WFS retraction)
				idx := int(lastCount - 1)
				if entry.IsRetracted(idx) {
					continue
				}
				// If this answer is conditional and any dependency now has answers,
				// retract and skip streaming it.
				if ds := entry.DelaySetFor(idx); ds != nil && !ds.Empty() {
					shouldRetract := false
					for dep := range ds {
						if child := e.subgoals.GetByHash(dep); child != nil {
							if child.Answers().Count() > 0 {
								shouldRetract = true
								break
							}
						}
					}
					if shouldRetract {
						entry.RetractAnswer(idx)
						continue
					}
				}

				select {
				case answerChan <- answer:
				case <-ctx.Done():
					return
				}
			}

			// If new answers have been added, refresh iterator and continue
			currentCount := entry.Answers().Count()
			if currentCount > lastCount {
				iter = entry.Answers().IteratorFrom(int(lastCount))
				continue
			}

			// If producer completed and we've consumed all answers, we're done
			status := entry.Status()
			if status == StatusComplete || status == StatusFailed {
				return
			}

			// Otherwise wait for more answers to arrive
			entry.answerMu.Lock()
			entry.answerCond.Wait()
			entry.answerMu.Unlock()
		}
	}()

	return answerChan
}

// consumeAnswers creates a consumer for an existing subgoal's answers.
func (e *SLGEngine) consumeAnswers(ctx context.Context, entry *SubgoalEntry) <-chan map[int64]Term {
	answerChan := make(chan map[int64]Term, 100)

	go func() {
		defer close(answerChan)

		iter := entry.Answers().Iterator()
		var consumed int64 = 0
		for {
			// Drain current snapshot
			for {
				if answer, ok := iter.Next(); ok {
					consumed++
					entry.consumptionCount.Add(1)
					// Skip retracted answers
					idx := int(consumed - 1)
					if entry.IsRetracted(idx) {
						continue
					}
					// If conditional and dependency has answers, retract and skip
					if ds := entry.DelaySetFor(idx); ds != nil && !ds.Empty() {
						shouldRetract := false
						for dep := range ds {
							if child := e.subgoals.GetByHash(dep); child != nil {
								if child.Answers().Count() > 0 {
									shouldRetract = true
									break
								}
							}
						}
						if shouldRetract {
							entry.RetractAnswer(idx)
							continue
						}
					}
					select {
					case answerChan <- answer:
						continue
					case <-ctx.Done():
						return
					}
				} else {
					break
				}
			}

			// If new answers have appeared, refresh iterator
			current := entry.Answers().Count()
			if current > consumed {
				iter = entry.Answers().IteratorFrom(int(consumed))
				continue
			}

			// If complete and no new answers pending, we're done
			status := entry.Status()
			if status == StatusComplete || status == StatusFailed {
				return
			}

			// Otherwise wait for more answers
			entry.answerMu.Lock()
			entry.answerCond.Wait()
			entry.answerMu.Unlock()
		}
	}()

	return answerChan
}

// SCC represents a strongly connected component in the dependency graph.
// Used for cycle detection and fixpoint computation.
type SCC struct {
	// Nodes in this SCC
	nodes []*SubgoalEntry

	// SCC index (for topological ordering)
	index int
}

// Contains checks if the SCC contains the given entry.
func (scc *SCC) Contains(entry *SubgoalEntry) bool {
	for _, node := range scc.nodes {
		if node == entry {
			return true
		}
	}
	return false
}

// AnswerCount returns the total number of answers across all nodes in the SCC.
func (scc *SCC) AnswerCount() int64 {
	var total int64
	for _, node := range scc.nodes {
		total += node.Answers().Count()
	}
	return total
}

// DetectCycles finds strongly connected components in the dependency graph
// using Tarjan's algorithm.
//
// Returns all SCCs in reverse topological order.
func (e *SLGEngine) DetectCycles() []*SCC {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get all subgoal entries
	entries := e.subgoals.AllEntries()
	if len(entries) == 0 {
		return nil
	}

	// Tarjan's algorithm state
	index := 0
	stack := make([]*SubgoalEntry, 0)
	onStack := make(map[*SubgoalEntry]bool)
	indices := make(map[*SubgoalEntry]int)
	lowlinks := make(map[*SubgoalEntry]int)
	sccs := make([]*SCC, 0)

	var strongConnect func(*SubgoalEntry)
	strongConnect = func(entry *SubgoalEntry) {
		// Set depth index
		indices[entry] = index
		lowlinks[entry] = index
		index++

		// Push to stack
		stack = append(stack, entry)
		onStack[entry] = true

		// Consider successors (dependencies)
		for _, dep := range entry.Dependencies() {
			if _, visited := indices[dep]; !visited {
				// Successor not yet visited; recurse
				strongConnect(dep)
				if lowlinks[dep] < lowlinks[entry] {
					lowlinks[entry] = lowlinks[dep]
				}
			} else if onStack[dep] {
				// Successor is on stack (part of current SCC)
				if indices[dep] < lowlinks[entry] {
					lowlinks[entry] = indices[dep]
				}
			}
		}

		// If entry is a root node, pop the stack to form SCC
		if lowlinks[entry] == indices[entry] {
			scc := &SCC{
				nodes: make([]*SubgoalEntry, 0),
				index: len(sccs),
			}

			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc.nodes = append(scc.nodes, w)
				if w == entry {
					break
				}
			}

			sccs = append(sccs, scc)
		}
	}

	// Find SCCs for all entries
	for _, entry := range entries {
		if _, visited := indices[entry]; !visited {
			strongConnect(entry)
		}
	}

	return sccs
}

// ComputeFixpoint computes the least fixpoint for a strongly connected component.
//
// This is used when a cycle is detected in the dependency graph. The algorithm:
//  1. Iteratively re-evaluate all subgoals in the SCC
//  2. Check if new answers were derived
//  3. Repeat until no new answers (fixpoint reached) or max iterations exceeded
//
// Returns error if max iterations exceeded without convergence.
func (e *SLGEngine) ComputeFixpoint(ctx context.Context, scc *SCC) error {
	if scc == nil || len(scc.nodes) == 0 {
		return nil
	}

	// Single node SCC without self-loop is not cyclic
	if len(scc.nodes) == 1 {
		node := scc.nodes[0]
		// Check for self-dependency
		hasSelfDep := false
		for _, dep := range node.Dependencies() {
			if dep == node {
				hasSelfDep = true
				break
			}
		}
		if !hasSelfDep {
			return nil // Not cyclic
		}
	}

	maxIterations := e.config.MaxFixpointIterations
	if maxIterations <= 0 {
		maxIterations = 1000
	}

	for iteration := 0; iteration < maxIterations; iteration++ {
		oldCount := scc.AnswerCount()

		// Re-evaluate all subgoals in SCC using stored evaluators
		for _, node := range scc.nodes {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Get the stored evaluator
			if node.evaluator == nil {
				continue // No evaluator stored, skip
			}

			// Re-evaluate to derive new answers based on updated dependencies
			// Don't reset the trie - we want to accumulate answers
			answerChan := make(chan map[int64]Term, 100)

			// Run evaluator
			go func(eval GoalEvaluator, ch chan<- map[int64]Term) {
				defer close(ch)
				eval(ctx, ch)
			}(node.evaluator, answerChan)

			// Insert new answers into existing trie
			for answer := range answerChan {
				if node.Answers().Insert(answer) {
					node.derivationCount.Add(1)
					e.totalAnswers.Add(1)
					node.answerCond.Broadcast()
				}
			}
		}

		newCount := scc.AnswerCount()

		// Check for fixpoint (no new answers)
		if newCount == oldCount {
			// Fixpoint reached
			for _, entry := range scc.nodes {
				entry.SetStatus(StatusComplete)
			}
			return nil
		}

		// Check context between iterations
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return fmt.Errorf("fixpoint computation exceeded max iterations (%d) for SCC with %d nodes",
		maxIterations, len(scc.nodes))
}

// IsCyclic checks if the dependency graph contains any cycles.
func (e *SLGEngine) IsCyclic() bool {
	sccs := e.DetectCycles()
	for _, scc := range sccs {
		if len(scc.nodes) > 1 {
			return true
		}
		// Check for self-loop in single-node SCC
		if len(scc.nodes) == 1 {
			node := scc.nodes[0]
			for _, dep := range node.Dependencies() {
				if dep == node {
					return true
				}
			}
		}
	}
	return false
}
