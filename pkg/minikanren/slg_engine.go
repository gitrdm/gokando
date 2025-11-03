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
}

// DefaultSLGConfig returns the default SLG configuration.
func DefaultSLGConfig() *SLGConfig {
	return &SLGConfig{
		MaxTableSize:              0,     // Unlimited
		MaxAnswersPerSubgoal:      10000, // Reasonable default
		MaxFixpointIterations:     1000,  // Prevent infinite loops
		EnableParallelProducers:   false, // Sequential by default
		EnableSubsumptionChecking: false, // Future enhancement
	}
}

// NewSLGEngine creates a new SLG engine with the given configuration.
func NewSLGEngine(config *SLGConfig) *SLGEngine {
	if config == nil {
		config = DefaultSLGConfig()
	}
	return &SLGEngine{
		subgoals: NewSubgoalTable(),
		config:   config,
	}
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

	if !isNew {
		// Cache hit: consume existing answers
		e.cacheHits.Add(1)
		return e.consumeAnswers(ctx, entry), nil
	}

	// Cache miss: need to evaluate
	e.cacheMisses.Add(1)

	// Store evaluator for potential re-evaluation during fixpoint
	entry.evaluator = evaluator

	// Start producer/consumer
	return e.produceAndConsume(ctx, entry, evaluator), nil
}

// produceAndConsume starts producer and consumer goroutines for a new subgoal.
// Producer: evaluates goal and inserts answers into trie
// Consumer: reads from trie and streams to output channel
func (e *SLGEngine) produceAndConsume(ctx context.Context, entry *SubgoalEntry, evaluator GoalEvaluator) <-chan map[int64]Term {
	answerChan := make(chan map[int64]Term, 100) // Buffered for batching

	// Producer goroutine: evaluate and populate trie
	go func() {
		defer func() {
			// Mark as complete when done producing
			entry.SetStatus(StatusComplete)
			entry.answerCond.Broadcast()
		}()

		// Channel for evaluator to send answers
		producerChan := make(chan map[int64]Term, 100)

		// Run evaluator in separate goroutine
		var evalErr error
		evalDone := make(chan struct{})
		go func() {
			defer close(producerChan)
			evalErr = evaluator(ctx, producerChan)
			close(evalDone)
		}()

		// Receive answers from evaluator and insert into trie
		for {
			select {
			case <-ctx.Done():
				entry.SetStatus(StatusFailed)
				return

			case answer, ok := <-producerChan:
				if !ok {
					// Producer finished
					<-evalDone // Wait for evaluator to complete
					if evalErr != nil {
						entry.SetStatus(StatusFailed)
					}
					return
				}

				// Insert into answer trie (deduplication happens here)
				if entry.Answers().Insert(answer) {
					entry.derivationCount.Add(1)
					e.totalAnswers.Add(1)

					// Notify waiting consumers
					entry.answerCond.Broadcast()
				}

				// Check answer limit
				if e.config.MaxAnswersPerSubgoal > 0 &&
					entry.Answers().Count() >= e.config.MaxAnswersPerSubgoal {
					entry.SetStatus(StatusComplete)
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
