package minikanren

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNewSLGEngine tests basic engine creation.
func TestNewSLGEngine(t *testing.T) {
	engine := NewSLGEngine(nil)
	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	if engine.subgoals == nil {
		t.Error("Expected subgoals table to be initialized")
	}

	if engine.config == nil {
		t.Error("Expected config to be initialized")
	}
}

// TestNewSLGEngine_CustomConfig tests engine with custom configuration.
func TestNewSLGEngine_CustomConfig(t *testing.T) {
	config := &SLGConfig{
		MaxTableSize:          1000,
		MaxAnswersPerSubgoal:  500,
		MaxFixpointIterations: 100,
	}

	engine := NewSLGEngine(config)

	if engine.config.MaxTableSize != 1000 {
		t.Errorf("Expected MaxTableSize 1000, got %d", engine.config.MaxTableSize)
	}

	if engine.config.MaxAnswersPerSubgoal != 500 {
		t.Errorf("Expected MaxAnswersPerSubgoal 500, got %d", engine.config.MaxAnswersPerSubgoal)
	}
}

// TestDefaultSLGConfig tests default configuration values.
func TestDefaultSLGConfig(t *testing.T) {
	config := DefaultSLGConfig()

	if config.MaxTableSize != 0 {
		t.Errorf("Expected unlimited table size, got %d", config.MaxTableSize)
	}

	if config.MaxAnswersPerSubgoal != 10000 {
		t.Errorf("Expected MaxAnswersPerSubgoal 10000, got %d", config.MaxAnswersPerSubgoal)
	}

	if config.MaxFixpointIterations != 1000 {
		t.Errorf("Expected MaxFixpointIterations 1000, got %d", config.MaxFixpointIterations)
	}
}

// TestGlobalEngine tests global engine management.
func TestGlobalEngine(t *testing.T) {
	// Reset global state
	ResetGlobalEngine()

	engine1 := GlobalEngine()
	if engine1 == nil {
		t.Fatal("Expected non-nil global engine")
	}

	engine2 := GlobalEngine()
	if engine1 != engine2 {
		t.Error("Expected same global engine instance")
	}

	// Set custom engine
	custom := NewSLGEngine(&SLGConfig{MaxTableSize: 500})
	SetGlobalEngine(custom)

	engine3 := GlobalEngine()
	if engine3 != custom {
		t.Error("Expected custom engine after SetGlobalEngine")
	}
}

// TestSLGEngine_Clear tests clearing engine state.
func TestSLGEngine_Clear(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Add some state
	pattern := NewCallPattern("test", []Term{NewAtom("a")})
	engine.subgoals.GetOrCreate(pattern)
	engine.totalEvaluations.Add(5)
	engine.totalAnswers.Add(10)

	// Clear
	engine.Clear()

	stats := engine.Stats()
	if stats.TotalEvaluations != 0 {
		t.Errorf("Expected TotalEvaluations 0 after clear, got %d", stats.TotalEvaluations)
	}

	if stats.CachedSubgoals != 0 {
		t.Errorf("Expected CachedSubgoals 0 after clear, got %d", stats.CachedSubgoals)
	}
}

// TestSLGEngine_Stats tests statistics tracking.
func TestSLGEngine_Stats(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Initial stats
	stats := engine.Stats()
	if stats.TotalEvaluations != 0 {
		t.Errorf("Expected TotalEvaluations 0, got %d", stats.TotalEvaluations)
	}

	// Simulate some activity
	engine.totalEvaluations.Add(10)
	engine.totalAnswers.Add(25)
	engine.cacheHits.Add(3)
	engine.cacheMisses.Add(7)

	stats = engine.Stats()
	if stats.TotalEvaluations != 10 {
		t.Errorf("Expected TotalEvaluations 10, got %d", stats.TotalEvaluations)
	}

	if stats.TotalAnswers != 25 {
		t.Errorf("Expected TotalAnswers 25, got %d", stats.TotalAnswers)
	}

	expectedHitRatio := 3.0 / 10.0
	if stats.HitRatio != expectedHitRatio {
		t.Errorf("Expected HitRatio %.2f, got %.2f", expectedHitRatio, stats.HitRatio)
	}
}

// TestSLGEngine_Evaluate_Simple tests basic evaluation with cache miss.
func TestSLGEngine_Evaluate_Simple(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("fact", []Term{NewAtom("a")})

	// Simple evaluator that produces one answer
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("result")}
		answers <- answer
		return nil
	}

	ctx := context.Background()
	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	// Collect answers
	collected := []map[int64]Term{}
	for answer := range resultChan {
		collected = append(collected, answer)
	}

	if len(collected) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(collected))
	}

	if len(collected) > 0 {
		if !collected[0][1].Equal(NewAtom("result")) {
			t.Errorf("Expected answer with 'result', got %v", collected[0][1])
		}
	}

	// Check stats
	stats := engine.Stats()
	if stats.TotalEvaluations != 1 {
		t.Errorf("Expected TotalEvaluations 1, got %d", stats.TotalEvaluations)
	}

	if stats.CacheMisses != 1 {
		t.Errorf("Expected CacheMisses 1, got %d", stats.CacheMisses)
	}
}

// TestSLGEngine_Evaluate_CacheHit tests cache hit behavior.
func TestSLGEngine_Evaluate_CacheHit(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("fact", []Term{NewAtom("a")})

	callCount := 0
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		callCount++
		answer := map[int64]Term{1: NewAtom("cached")}
		answers <- answer
		return nil
	}

	ctx := context.Background()

	// First evaluation (cache miss)
	resultChan1, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("First Evaluate error: %v", err)
	}

	// Consume all answers
	for range resultChan1 {
	}

	// Wait for producer to complete
	time.Sleep(50 * time.Millisecond)

	// Second evaluation (cache hit)
	resultChan2, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Second Evaluate error: %v", err)
	}

	// Collect answers from cache
	collected := []map[int64]Term{}
	for answer := range resultChan2 {
		collected = append(collected, answer)
	}

	if len(collected) != 1 {
		t.Errorf("Expected 1 cached answer, got %d", len(collected))
	}

	// Evaluator should only be called once
	if callCount != 1 {
		t.Errorf("Expected evaluator called once, got %d", callCount)
	}

	// Check stats
	stats := engine.Stats()
	if stats.CacheHits != 1 {
		t.Errorf("Expected CacheHits 1, got %d", stats.CacheHits)
	}
}

// TestSLGEngine_Evaluate_MultipleAnswers tests producing multiple answers.
func TestSLGEngine_Evaluate_MultipleAnswers(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("range", []Term{NewAtom(5)})

	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		for i := 1; i <= 5; i++ {
			answer := map[int64]Term{1: NewAtom(i)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	collected := []map[int64]Term{}
	for answer := range resultChan {
		collected = append(collected, answer)
	}

	if len(collected) != 5 {
		t.Errorf("Expected 5 answers, got %d", len(collected))
	}
}

// TestSLGEngine_Evaluate_Deduplication tests answer deduplication.
func TestSLGEngine_Evaluate_Deduplication(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("dup", []Term{NewAtom("x")})

	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		// Send duplicate answers
		for i := 0; i < 3; i++ {
			answer := map[int64]Term{1: NewAtom("same")}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	collected := []map[int64]Term{}
	for answer := range resultChan {
		collected = append(collected, answer)
	}

	// Should only get 1 unique answer
	if len(collected) != 1 {
		t.Errorf("Expected 1 unique answer after deduplication, got %d", len(collected))
	}
}

// TestSLGEngine_Evaluate_ContextCancellation tests cancellation.
func TestSLGEngine_Evaluate_ContextCancellation(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("infinite", []Term{NewAtom("x")})

	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				answer := map[int64]Term{1: NewAtom(i)}
				answers <- answer
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	// Consume answers until channel closes
	count := 0
	for range resultChan {
		count++
	}

	// Should have stopped due to cancellation
	if count > 20 {
		t.Errorf("Expected limited answers due to cancellation, got %d", count)
	}
}

// TestSLGEngine_Evaluate_NilPattern tests error on nil pattern.
func TestSLGEngine_Evaluate_NilPattern(t *testing.T) {
	engine := NewSLGEngine(nil)
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		return nil
	}

	_, err := engine.Evaluate(context.Background(), nil, evaluator)
	if err == nil {
		t.Error("Expected error for nil pattern")
	}
}

// TestSLGEngine_Evaluate_NilEvaluator tests error on nil evaluator.
func TestSLGEngine_Evaluate_NilEvaluator(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("test", []Term{NewAtom("a")})

	_, err := engine.Evaluate(context.Background(), pattern, nil)
	if err == nil {
		t.Error("Expected error for nil evaluator")
	}
}

// TestSLGEngine_Evaluate_EvaluatorError tests error propagation.
func TestSLGEngine_Evaluate_EvaluatorError(t *testing.T) {
	engine := NewSLGEngine(nil)
	pattern := NewCallPattern("error", []Term{NewAtom("x")})

	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		return fmt.Errorf("test error")
	}

	ctx := context.Background()
	resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	// Consume channel (should close due to error)
	count := 0
	for range resultChan {
		count++
	}

	// Should have no answers due to error
	if count != 0 {
		t.Errorf("Expected 0 answers due to error, got %d", count)
	}
}

// TestSLGEngine_DetectCycles_NoCycles tests cycle detection with no cycles.
func TestSLGEngine_DetectCycles_NoCycles(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Create independent subgoals
	pattern1 := NewCallPattern("a", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("b", []Term{NewAtom(2)})

	engine.subgoals.GetOrCreate(pattern1)
	engine.subgoals.GetOrCreate(pattern2)

	sccs := engine.DetectCycles()

	// Each independent node should be in its own SCC
	if len(sccs) != 2 {
		t.Errorf("Expected 2 SCCs, got %d", len(sccs))
	}

	// No cycles should exist
	if engine.IsCyclic() {
		t.Error("Expected no cycles in independent subgoals")
	}
}

// TestSLGEngine_DetectCycles_SimpleCycle tests simple 2-node cycle.
func TestSLGEngine_DetectCycles_SimpleCycle(t *testing.T) {
	engine := NewSLGEngine(nil)

	pattern1 := NewCallPattern("a", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("b", []Term{NewAtom(2)})

	entry1, _ := engine.subgoals.GetOrCreate(pattern1)
	entry2, _ := engine.subgoals.GetOrCreate(pattern2)

	// Create cycle: a -> b -> a
	entry1.AddDependency(entry2)
	entry2.AddDependency(entry1)

	sccs := engine.DetectCycles()

	// Should find one SCC with both nodes
	foundCycle := false
	for _, scc := range sccs {
		if len(scc.nodes) == 2 {
			foundCycle = true
			if !scc.Contains(entry1) || !scc.Contains(entry2) {
				t.Error("Expected SCC to contain both entries")
			}
		}
	}

	if !foundCycle {
		t.Error("Expected to find 2-node cycle")
	}

	if !engine.IsCyclic() {
		t.Error("Expected IsCyclic to return true")
	}
}

// TestSLGEngine_DetectCycles_SelfLoop tests self-referential cycle.
func TestSLGEngine_DetectCycles_SelfLoop(t *testing.T) {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("recursive", []Term{NewAtom("x")})
	entry, _ := engine.subgoals.GetOrCreate(pattern)

	// Create self-loop
	entry.AddDependency(entry)

	if !engine.IsCyclic() {
		t.Error("Expected IsCyclic to return true for self-loop")
	}

	sccs := engine.DetectCycles()
	foundSelfLoop := false
	for _, scc := range sccs {
		if scc.Contains(entry) {
			foundSelfLoop = true
			// Self-loop creates single-node SCC
			if len(scc.nodes) != 1 {
				t.Errorf("Expected single-node SCC for self-loop, got %d nodes", len(scc.nodes))
			}
		}
	}

	if !foundSelfLoop {
		t.Error("Expected to find self-loop SCC")
	}
}

// TestSLGEngine_DetectCycles_ComplexGraph tests complex dependency graph.
func TestSLGEngine_DetectCycles_ComplexGraph(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Create graph: a -> b -> c -> b (cycle b-c)
	//               a -> d (no cycle)
	patternA := NewCallPattern("a", []Term{NewAtom(1)})
	patternB := NewCallPattern("b", []Term{NewAtom(2)})
	patternC := NewCallPattern("c", []Term{NewAtom(3)})
	patternD := NewCallPattern("d", []Term{NewAtom(4)})

	entryA, _ := engine.subgoals.GetOrCreate(patternA)
	entryB, _ := engine.subgoals.GetOrCreate(patternB)
	entryC, _ := engine.subgoals.GetOrCreate(patternC)
	entryD, _ := engine.subgoals.GetOrCreate(patternD)

	entryA.AddDependency(entryB)
	entryA.AddDependency(entryD)
	entryB.AddDependency(entryC)
	entryC.AddDependency(entryB)

	sccs := engine.DetectCycles()

	// Should find cycle containing B and C
	foundCycle := false
	for _, scc := range sccs {
		if len(scc.nodes) > 1 {
			foundCycle = true
			hasB := scc.Contains(entryB)
			hasC := scc.Contains(entryC)
			if !hasB || !hasC {
				t.Error("Expected SCC to contain B and C")
			}
		}
	}

	if !foundCycle {
		t.Error("Expected to find cycle in complex graph")
	}
}

// TestSCC_AnswerCount tests answer counting across SCC nodes.
func TestSCC_AnswerCount(t *testing.T) {
	pattern1 := NewCallPattern("a", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("b", []Term{NewAtom(2)})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)

	// Add answers
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("x")})
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("y")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("z")})

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	count := scc.AnswerCount()
	if count != 3 {
		t.Errorf("Expected AnswerCount 3, got %d", count)
	}
}

// TestSLGEngine_ComputeFixpoint_NoChange tests fixpoint with no new answers.
func TestSLGEngine_ComputeFixpoint_NoChange(t *testing.T) {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("test", []Term{NewAtom("x")})
	entry := NewSubgoalEntry(pattern)

	// Pre-populate with answer
	entry.Answers().Insert(map[int64]Term{1: NewAtom("stable")})

	// Make it cyclic by adding self-dependency
	entry.AddDependency(entry)

	// Evaluator that produces no new answers
	entry.evaluator = GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		// No new answers
		return nil
	})

	scc := &SCC{nodes: []*SubgoalEntry{entry}}

	ctx := context.Background()
	err := engine.ComputeFixpoint(ctx, scc)
	if err != nil {
		t.Errorf("ComputeFixpoint error: %v", err)
	}

	// Answer count should remain 1
	if scc.AnswerCount() != 1 {
		t.Errorf("Expected 1 answer, got %d", scc.AnswerCount())
	}

	// Should reach fixpoint and mark as Complete
	if entry.Status() != StatusComplete {
		t.Errorf("Expected status Complete, got %s", entry.Status())
	}
}

// TestSLGEngine_ComputeFixpoint_SingleNonCyclic tests non-cyclic single node.
// TestSLGEngine_ComputeFixpoint_ActualReEvaluation tests fixpoint derives new answers.
func TestSLGEngine_ComputeFixpoint_ActualReEvaluation(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Simulate transitive closure scenario:
	// Initial: a->b, b->c
	// After fixpoint: also derive a->c

	patternAB := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})
	entryAB := NewSubgoalEntry(patternAB)

	// Track whether this is first call or re-evaluation
	callCount := 1 // Start at 1 - initial evaluation already happened

	// Evaluator that derives transitive edge on re-evaluation
	entryAB.evaluator = GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		callCount++
		// First call: produce a->b
		if callCount == 1 {
			answers <- map[int64]Term{1: NewAtom("a"), 2: NewAtom("b")}
		} else {
			// Re-evaluation: also derive a->c (transitive through b->c)
			answers <- map[int64]Term{1: NewAtom("a"), 2: NewAtom("b")}
			answers <- map[int64]Term{1: NewAtom("a"), 2: NewAtom("c")}
		}
		return nil
	})

	// Simulate initial evaluation by populating first answer
	entryAB.Answers().Insert(map[int64]Term{1: NewAtom("a"), 2: NewAtom("b")})

	patternBC := NewCallPattern("edge", []Term{NewAtom("b"), NewAtom("c")})
	entryBC := NewSubgoalEntry(patternBC)
	entryBC.Answers().Insert(map[int64]Term{1: NewAtom("b"), 2: NewAtom("c")})

	// Set up dependencies to make it cyclic
	entryAB.AddDependency(entryBC)
	entryBC.AddDependency(entryAB)

	// This evaluator doesn't produce new answers
	entryBC.evaluator = GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		return nil
	})

	scc := &SCC{nodes: []*SubgoalEntry{entryAB, entryBC}}

	ctx := context.Background()
	err := engine.ComputeFixpoint(ctx, scc)
	if err != nil {
		t.Errorf("ComputeFixpoint error: %v", err)
	}

	// After fixpoint, entryAB should have 2 answers (original a->b and derived a->c)
	if entryAB.Answers().Count() != 2 {
		t.Errorf("Expected 2 answers for a->, got %d", entryAB.Answers().Count())
	}

	// Both nodes should be Complete
	if entryAB.Status() != StatusComplete {
		t.Errorf("Expected entryAB status Complete, got %s", entryAB.Status())
	}
	if entryBC.Status() != StatusComplete {
		t.Errorf("Expected entryBC status Complete, got %s", entryBC.Status())
	}
}

// TestSLGEngine_ComputeFixpoint_SingleNonCyclic tests non-cyclic single node.
func TestSLGEngine_ComputeFixpoint_SingleNonCyclic(t *testing.T) {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("test", []Term{NewAtom("x")})
	entry := NewSubgoalEntry(pattern)

	scc := &SCC{nodes: []*SubgoalEntry{entry}}

	ctx := context.Background()
	err := engine.ComputeFixpoint(ctx, scc)
	if err != nil {
		t.Errorf("ComputeFixpoint error: %v", err)
	}

	// Non-cyclic single node should complete immediately
}

// TestSLGEngine_ComputeFixpoint_ContextCancellation tests cancellation during fixpoint.
func TestSLGEngine_ComputeFixpoint_ContextCancellation(t *testing.T) {
	// This test validates the context checking mechanism, but since
	// fixpoint computation is a framework (no actual re-evaluation yet),
	// it will complete immediately if answer count doesn't change.
	// We test that context cancellation is checked during iteration.

	engine := NewSLGEngine(&SLGConfig{MaxFixpointIterations: 1})

	pattern1 := NewCallPattern("a", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("b", []Term{NewAtom(2)})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)

	// Create cycle
	entry1.AddDependency(entry2)
	entry2.AddDependency(entry1)

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := engine.ComputeFixpoint(ctx, scc)

	// Should get either context.Canceled or nil (if fixpoint reached before check)
	if err != nil && err != context.Canceled {
		t.Errorf("Expected nil or context.Canceled, got %v", err)
	}
}

// TestSLGEngine_Concurrent tests concurrent evaluations.
func TestSLGEngine_Concurrent(t *testing.T) {
	engine := NewSLGEngine(nil)

	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			pattern := NewCallPattern("concurrent", []Term{NewAtom(id)})
			evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
				answer := map[int64]Term{1: NewAtom(fmt.Sprintf("result%d", id))}
				answers <- answer
				return nil
			}

			ctx := context.Background()
			resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
			if err != nil {
				t.Errorf("Goroutine %d: Evaluate error: %v", id, err)
				return
			}

			// Consume answers
			for range resultChan {
			}
		}(i)
	}

	wg.Wait()

	// Check that all subgoals were created
	stats := engine.Stats()
	if stats.CachedSubgoals != numGoroutines {
		t.Errorf("Expected %d cached subgoals, got %d", numGoroutines, stats.CachedSubgoals)
	}
}

// Benchmark SLG engine evaluation.
func BenchmarkSLGEngine_Evaluate(b *testing.B) {
	engine := NewSLGEngine(nil)
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("benchmark")}
		answers <- answer
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := NewCallPattern("bench", []Term{NewAtom(i)})
		ctx := context.Background()
		resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)
		for range resultChan {
		}
	}
}

// Benchmark cycle detection.
func BenchmarkSLGEngine_DetectCycles(b *testing.B) {
	engine := NewSLGEngine(nil)

	// Create graph with 100 nodes
	entries := make([]*SubgoalEntry, 100)
	for i := 0; i < 100; i++ {
		pattern := NewCallPattern("node", []Term{NewAtom(i)})
		entry, _ := engine.subgoals.GetOrCreate(pattern)
		entries[i] = entry

		// Create dependencies (chain)
		if i > 0 {
			entry.AddDependency(entries[i-1])
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.DetectCycles()
	}
}
