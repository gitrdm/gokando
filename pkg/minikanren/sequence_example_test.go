package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewSequence shows sliding-window pruning with S={1}, k=3, min=2.
func ExampleNewSequence() {
	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "x2") // forced not in S
	x3 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x3")
	x4 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x4")
	x5 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x5")

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
