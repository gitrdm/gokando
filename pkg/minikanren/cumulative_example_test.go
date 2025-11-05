package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewCumulative demonstrates time-table pruning using the HLAPI
// wrapper `Cumulative`.
//
// HLAPI note: the example uses `model.Cumulative(...)` for brevity; the
// low-level `NewCumulative` constructor is preserved in comments for readers
// who need to see the explicit constraint construction.
//
// The model encodes two tasks with fixed durations and resource demands.
// Task A is fixed to start at time 2 (duration 2, demand 2). Task B can
// start in the interval [1..4] (duration 2, demand 1). The cumulative
// constraint with capacity=2 forces propagation: the high-demand Task A
// restricts feasible starts for Task B so the solver prunes B's domain.
//
// The example shows both the low-level constructor form (commented) and the
// HLAPI wrapper used in examples for brevity. Both produce the same
// propagation effects; the HLAPI call keeps example code short and clear.
func ExampleNewCumulative() {
	model := NewModel()

	// Task A: fixed at start=2, duration=2, demand=2
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B: start in [1..4], duration=2, demand=1
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	// Low-level API (kept as comment):
	// cum, err := NewCumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)
	// if err != nil {
	//     panic(err)
	// }
	// model.AddConstraint(cum)
	// HLAPI wrapper:
	_ = model.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	// If you only need concrete solutions (assignments), the HLAPI helper
	// SolveN(ctx, model, maxSolutions) is a convenient wrapper that creates
	// a solver, runs the search, and returns solutions. Example:
	//
	//    sols, err := SolveN(ctx, model, 1)
	//
	// However, when you want to inspect solver internals (domains after
	// propagation) or call methods like GetDomain/propagate, create the
	// Solver explicitly as done below and call Solve on it. That allows
	// reading the pruned domains from the solver state.
	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root by running a one-solution search (will stop at root if none).
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}
