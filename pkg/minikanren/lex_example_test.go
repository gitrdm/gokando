package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewLexLessEq demonstrates using the LexLessEq constraint via the
// HLAPI to relate two short integer vectors and inspect the effect of
// propagation on their domains.
//
// The model builds two vectors x = (x1,x2) and y = (y1,y2) with compact
// domains using `IntVarValues`. We post `LexLessEq(x,y)` which enforces the
// lexicographic ordering x ≤_lex y (either x1 < y1, or x1 == y1 and x2 ≤ y2).
//
// The example triggers a brief propagation run (solver.Solve with limit=0)
// to reach a fixed-point and then prints `y1`'s domain so readers can see
// how propagation tightened (or not) the bounds. The output below shows the
// domain of `y1` after propagation.
func ExampleNewLexLessEq() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "x1")
	x1 := model.IntVarValues([]int{2, 3, 4}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{1, 2, 3}), "x2")
	x2 := model.IntVarValues([]int{1, 2, 3}, "x2")
	// y1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{3, 4, 5}), "y1")
	y1 := model.IntVarValues([]int{3, 4, 5}, "y1")
	// y2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "y2")
	y2 := model.IntVarValues([]int{2, 3, 4}, "y2")

	c, _ := NewLexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Run fixed-point propagation via a zero-solution search (limit=0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("y1: %s\n", solver.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}
