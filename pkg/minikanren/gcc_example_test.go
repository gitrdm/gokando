package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewGlobalCardinality demonstrates posting a global-cardinality
// (GCC) constraint and observing pruning via propagation.
//
// HLAPI note: the example uses the `model.GlobalCardinality(...)` wrapper
// which posts the constraint directly; the equivalent low-level constructor
// `NewGlobalCardinality` is preserved in comments for readers learning the
// lower-level API.
//
// Global cardinality constrains how many times each value may appear across
// a set of variables. In this example we have three variables (a,b,c) whose
// domains are {1,2}. We set the occurrence limits so that value 1 must
// appear exactly once (min[1]=max[1]=1) while value 2 can appear 0..3 times.
//
// After posting the constraint and running propagation, the solver prunes
// domains so that `a` becomes fixed to 1 and the remaining variables are
// pruned to 2 to satisfy the global counts.
//
// Note on HLAPI usage: the low-level constructor form `NewGlobalCardinality`
// returns a Constraint that you must add to the model. For concise examples
// we prefer the HLAPI wrapper `model.GlobalCardinality(...)` which posts the
// constraint directly. Both forms are equivalent; the HLAPI keeps example
// code short. Because this example inspects solver domains after propagation
// we explicitly create a `Solver` and call `Solve` (rather than using the
// SolveN helper which returns concrete solutions but does not expose the
// solver state).
func ExampleNewGlobalCardinality() {
	model := NewModel()

	// Low-level constructors are preserved as comments for reference:
	// a := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "a")
	a := model.IntVarValues([]int{1}, "a")
	// b := model.NewVariableWithName(NewBitSetDomain(2), "b")
	b := model.IntVar(1, 2, "b")
	// c := model.NewVariableWithName(NewBitSetDomain(2), "c")
	c := model.IntVar(1, 2, "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3

	// Low-level API (kept as comment):
	// gcc, err := NewGlobalCardinality([]*FDVariable{a, b, c}, min, max)
	// if err != nil {
	//     panic(err)
	// }
	// model.AddConstraint(gcc)
	// HLAPI wrapper (preferred for examples):
	_ = model.GlobalCardinality([]*FDVariable{a, b, c}, min, max)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Println("a:", solver.GetDomain(nil, a.ID()))
	fmt.Println("b:", solver.GetDomain(nil, b.ID()))
	fmt.Println("c:", solver.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}
