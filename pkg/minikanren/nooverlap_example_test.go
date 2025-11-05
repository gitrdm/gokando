package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewNoOverlap demonstrates modeling single-machine scheduling with
// the `NoOverlap` helper. It is a thin wrapper around `Cumulative`
// specialized to capacity=1 and is convenient for expressing that a set of
// tasks (with start times and durations) must not overlap on a single
// resource.
//
// In this example:
//   - Task A has a fixed start at 2 and duration 2 (so occupies [2,4) in our
//     discrete time indexing); Task B may start in [1..4] and also has
//     duration 2.
//   - We post `NoOverlap([A,B], durations=[2,2])` and run a short propagation
//     search to reach a fixed-point. Finally we print the tightened domains
//     for A and B to demonstrate propagation.
//
// The printed domains show that A remains fixed at 2 and B is constrained to
// start at 4 (so the two tasks do not overlap), which is the expected result
// after propagation.
func ExampleNewNoOverlap() {
	model := NewModel()

	// Task A fixed at start=2, duration=2 ⇒ executes over [2,3]
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B can start in [1..4], duration=2
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	noov, err := NewNoOverlap([]*FDVariable{A, B}, []int{2, 2})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(noov)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root via a short search
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}
