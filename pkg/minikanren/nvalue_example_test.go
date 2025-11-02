package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewAtMostNValues shows how fixing one variable and enforcing
// at most one distinct value prunes others to the same value.
func ExampleNewAtMostNValues() {
	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1}), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x3")
	limit := model.NewVariableWithName(NewBitSetDomain(2), "limit") // distinct ≤ 1

	_, _ = NewAtMostNValues(model, []*FDVariable{x1, x2, x3}, limit)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate only

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {1}
	// x3: {1}
}

// ExampleNewNValue shows wiring for an exact distinct-count.
func ExampleNewNValue() {
	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	// Exact NValue=1 ⇒ NPlus1=2
	nPlus1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "N+1")

	_, _ = NewNValue(model, []*FDVariable{x1, x2}, nPlus1)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// No pruning here, but the composition is established and will prune
	// as soon as one side gets fixed by other constraints or decisions.
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
}
