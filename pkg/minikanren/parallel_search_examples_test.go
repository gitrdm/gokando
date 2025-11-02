package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleSolver_SolveParallel demonstrates basic usage of parallel solving
// with a small AllDifferent problem. It creates 3 variables with domains {1,2,3},
// posts an AllDifferent constraint, and asks the solver to find 6 solutions
// using several workers.
func ExampleSolver_SolveParallel() {
	// 1) Build a model with 3 variables, each from 1..3
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))

	// 2) Constrain them to be all different
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	// 3) Create a solver and run in parallel with up to 4 workers
	solver := NewSolver(model)
	ctx := context.Background()

	// Find up to 6 solutions in parallel (there are 3! = 6)
	solutions, err := solver.SolveParallel(ctx, 4, 6)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 6
}

// ExampleSolver_SolveParallel_limit shows how to cap the number of solutions
// returned by parallel search using the maxSolutions parameter.
func ExampleSolver_SolveParallel_limit() {
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)
	ctx := context.Background()

	// Ask for at most 3 solutions (there are 4! = 24 total)
	solutions, err := solver.SolveParallel(ctx, 4, 3)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 3
}

// ExampleSolver_SolveParallel_cancel demonstrates cooperative cancellation
// using a context with timeout. In real applications this is useful to stop
// long searches or to enforce time budgets.
func ExampleSolver_SolveParallel_cancel() {
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)

	// A tiny timeout to illustrate cancellation. In practice, use a sensible value.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Request many solutions so the timeout is likely to fire first.
	// We intentionally do not assert output for examples that depend on timing.
	_, _ = solver.SolveParallel(ctx, 8, 0)
}

// ExampleDefaultParallelSearchConfig shows how to inspect the default parallel
// search configuration. Use this as a guide to choose worker counts and queue size.
func ExampleDefaultParallelSearchConfig() {
	cfg := DefaultParallelSearchConfig()
	// You can use cfg.NumWorkers to size solver parallelism, and cfg.WorkQueueSize
	// to adjust throughput vs memory. Only queue size is deterministic here.
	fmt.Printf("queue=%d\n", cfg.WorkQueueSize)
	// Output:
	// queue=1000
}
