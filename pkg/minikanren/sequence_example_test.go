package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewSequence demonstrates the `Sequence` constraint which enforces
// sliding-window (cardinality) conditions over a sequence of variables.
//
// Here we set S={1} (the value set of interest), window size k=3, and
// require at least `min=2` occurrences of values from S inside each window of
// length k. One variable (x2) is fixed to 2 (not in S) which forces the two
// other variables in the first window to take the value 1. The example runs a
// short propagation pass and prints the domains for x1 and x3 to show the
// pruning result.
func ExampleNewSequence() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "x2") // forced not in S
	x2 := model.IntVarValues([]int{2}, "x2") // forced not in S
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x3")
	x3 := model.IntVarValues([]int{1, 2}, "x3")
	// x4 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x4")
	x4 := model.IntVarValues([]int{1, 2}, "x4")
	// x5 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x5")
	x5 := model.IntVarValues([]int{1, 2}, "x5")

	_, _ = NewSequence(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, 3, 2, 3)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// Window [x1,x2,x3] needs at least two 1s; since x2!=1, both x1 and x3 become 1
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x1: {1}
	// x3: {1}
}
