package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewSLGEngine demonstrates creating a new SLG engine with default configuration.
func ExampleNewSLGEngine() {
	engine := NewSLGEngine(nil)

	stats := engine.Stats()
	fmt.Printf("Initial subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Max answers per subgoal: %d\n", engine.config.MaxAnswersPerSubgoal)

	// Output:
	// Initial subgoals: 0
	// Max answers per subgoal: 10000
}

// ExampleNewSLGEngine_customConfig demonstrates custom configuration.
func ExampleNewSLGEngine_customConfig() {
	config := &SLGConfig{
		MaxTableSize:          5000,
		MaxAnswersPerSubgoal:  100,
		MaxFixpointIterations: 500,
	}

	engine := NewSLGEngine(config)
	fmt.Printf("Max table size: %d\n", engine.config.MaxTableSize)
	fmt.Printf("Max fixpoint iterations: %d\n", engine.config.MaxFixpointIterations)

	// Output:
	// Max table size: 5000
	// Max fixpoint iterations: 500
}

// ExampleSLGEngine_Evaluate demonstrates basic tabled evaluation.
func ExampleSLGEngine_Evaluate() {
	engine := NewSLGEngine(nil)

	// Define a call pattern for a "fact" predicate
	pattern := NewCallPattern("color", []Term{NewAtom("x")})

	// Simple evaluator that produces three color answers
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		colors := []string{"red", "green", "blue"}
		for _, color := range colors {
			answer := map[int64]Term{1: NewAtom(color)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Collect all answers
	count := 0
	for range resultChan {
		count++
	}

	fmt.Printf("Derived %d answers\n", count)

	// Second evaluation should hit cache
	resultChan2, _ := engine.Evaluate(ctx, pattern, evaluator)
	for range resultChan2 {
	}

	stats := engine.Stats()
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)

	// Output:
	// Derived 3 answers
	// Cache hits: 1
}

// ExampleSLGEngine_Evaluate_streaming demonstrates streaming answers as they're produced.
func ExampleSLGEngine_Evaluate_streaming() {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("range", []Term{NewAtom(5)})

	// Evaluator that produces answers incrementally
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		for i := 1; i <= 5; i++ {
			answer := map[int64]Term{1: NewAtom(i)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Process answers as they arrive
	for answer := range resultChan {
		value := answer[1]
		fmt.Printf("Got answer: %v\n", value)
	}

	// Output:
	// Got answer: 1
	// Got answer: 2
	// Got answer: 3
	// Got answer: 4
	// Got answer: 5
}

// ExampleSLGEngine_DetectCycles demonstrates cycle detection in dependency graphs.
func ExampleSLGEngine_DetectCycles() {
	engine := NewSLGEngine(nil)

	// Create three subgoals with dependencies
	patternA := NewCallPattern("ancestor", []Term{NewAtom("alice"), NewAtom("x")})
	patternB := NewCallPattern("ancestor", []Term{NewAtom("bob"), NewAtom("x")})
	patternC := NewCallPattern("ancestor", []Term{NewAtom("charlie"), NewAtom("x")})

	entryA, _ := engine.subgoals.GetOrCreate(patternA)
	entryB, _ := engine.subgoals.GetOrCreate(patternB)
	entryC, _ := engine.subgoals.GetOrCreate(patternC)

	// Create cycle: A -> B -> C -> B
	entryA.AddDependency(entryB)
	entryB.AddDependency(entryC)
	entryC.AddDependency(entryB)

	// Detect cycles
	sccs := engine.DetectCycles()

	fmt.Printf("Found %d SCCs\n", len(sccs))

	// Check if cyclic
	if engine.IsCyclic() {
		fmt.Println("Graph contains cycles")
	}

	// Find the cyclic SCC
	for _, scc := range sccs {
		if len(scc.nodes) > 1 {
			fmt.Printf("Cyclic SCC has %d nodes\n", len(scc.nodes))
		}
	}

	// Output:
	// Found 2 SCCs
	// Graph contains cycles
	// Cyclic SCC has 2 nodes
}

// ExampleSLGEngine_DetectCycles_selfLoop demonstrates detecting self-referential predicates.
func ExampleSLGEngine_DetectCycles_selfLoop() {
	engine := NewSLGEngine(nil)

	// Create a recursive predicate: path(X, Y)
	pattern := NewCallPattern("path", []Term{NewAtom("x"), NewAtom("y")})
	entry, _ := engine.subgoals.GetOrCreate(pattern)

	// Create self-loop (path depends on path)
	entry.AddDependency(entry)

	if engine.IsCyclic() {
		fmt.Println("Self-referential predicate detected")
	}

	sccs := engine.DetectCycles()
	for _, scc := range sccs {
		if scc.Contains(entry) {
			fmt.Printf("SCC contains %d node(s)\n", len(scc.nodes))
		}
	}

	// Output:
	// Self-referential predicate detected
	// SCC contains 1 node(s)
}

// ExampleSLGEngine_Stats demonstrates statistics tracking.
func ExampleSLGEngine_Stats() {
	engine := NewSLGEngine(nil)

	// Evaluate several subgoals
	for i := 1; i <= 3; i++ {
		pattern := NewCallPattern("test", []Term{NewAtom(i)})
		evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
			answer := map[int64]Term{1: NewAtom(fmt.Sprintf("result%d", i))}
			answers <- answer
			return nil
		}

		resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
		for range resultChan {
		}
	}

	// Re-evaluate first subgoal (cache hit)
	pattern := NewCallPattern("test", []Term{NewAtom(1)})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("result1")}
		answers <- answer
		return nil
	}
	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	stats := engine.Stats()
	fmt.Printf("Total evaluations: %d\n", stats.TotalEvaluations)
	fmt.Printf("Cached subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache misses: %d\n", stats.CacheMisses)
	fmt.Printf("Hit ratio: %.2f\n", stats.HitRatio)

	// Output:
	// Total evaluations: 4
	// Cached subgoals: 3
	// Cache hits: 1
	// Cache misses: 3
	// Hit ratio: 0.25
}

// ExampleGlobalEngine demonstrates using the global engine singleton.
func ExampleGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	// Get global engine (created on first access)
	engine1 := GlobalEngine()
	engine2 := GlobalEngine()

	if engine1 == engine2 {
		fmt.Println("Same engine instance")
	}

	// Evaluate using global engine
	pattern := NewCallPattern("global", []Term{NewAtom("test")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("answer")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine1.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	// State is shared
	stats := engine2.Stats()
	fmt.Printf("Shared state - evaluations: %d\n", stats.TotalEvaluations)

	// Output:
	// Same engine instance
	// Shared state - evaluations: 1
}

// ExampleResetGlobalEngine demonstrates resetting global engine state.
func ExampleResetGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	engine := GlobalEngine()

	// Add some state
	pattern := NewCallPattern("temp", []Term{NewAtom("x")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("data")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	statsBefore := engine.Stats()
	fmt.Printf("Before reset - evaluations: %d\n", statsBefore.TotalEvaluations)

	// Reset state
	ResetGlobalEngine()

	statsAfter := engine.Stats()
	fmt.Printf("After reset - evaluations: %d\n", statsAfter.TotalEvaluations)
	fmt.Printf("After reset - cached subgoals: %d\n", statsAfter.CachedSubgoals)

	// Output:
	// Before reset - evaluations: 1
	// After reset - evaluations: 0
	// After reset - cached subgoals: 0
}

// ExampleSCC_AnswerCount demonstrates answer counting across SCC nodes.
func ExampleSCC_AnswerCount() {
	pattern1 := NewCallPattern("p", []Term{NewAtom(1)})
	pattern2 := NewCallPattern("q", []Term{NewAtom(2)})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)

	// Add answers to both entries
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("a")})
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("b")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("c")})

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	fmt.Printf("Total answers in SCC: %d\n", scc.AnswerCount())

	// Output:
	// Total answers in SCC: 3
}

// ExampleSLGEngine_ComputeFixpoint demonstrates fixpoint computation framework.
func ExampleSLGEngine_ComputeFixpoint() {
	engine := NewSLGEngine(nil)

	// Create two mutually dependent subgoals
	pattern1 := NewCallPattern("reaches", []Term{NewAtom("a"), NewAtom("x")})
	pattern2 := NewCallPattern("reaches", []Term{NewAtom("b"), NewAtom("x")})

	entry1, _ := engine.subgoals.GetOrCreate(pattern1)
	entry2, _ := engine.subgoals.GetOrCreate(pattern2)

	// Add initial answers
	entry1.Answers().Insert(map[int64]Term{1: NewAtom("node1")})
	entry2.Answers().Insert(map[int64]Term{1: NewAtom("node2")})

	// Create mutual dependency (cycle)
	entry1.AddDependency(entry2)
	entry2.AddDependency(entry1)

	scc := &SCC{nodes: []*SubgoalEntry{entry1, entry2}}

	// Compute fixpoint
	err := engine.ComputeFixpoint(context.Background(), scc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Fixpoint computed successfully")
		fmt.Printf("Total answers: %d\n", scc.AnswerCount())
	}

	// Output:
	// Fixpoint computed successfully
	// Total answers: 2
}
