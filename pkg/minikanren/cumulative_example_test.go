package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewCumulative shows time-table pruning where a fixed high-demand task
// restricts the feasible starts of another task.
func ExampleNewCumulative() {
	model := NewModel()

	// Task A: fixed at start=2, duration=2, demand=2
	A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	// Task B: start in [1..4], duration=2, demand=1
	B := model.NewVariableWithName(NewBitSetDomain(4), "B")

	cum, err := NewCumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(cum)

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
