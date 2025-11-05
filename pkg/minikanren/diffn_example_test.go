package minikanren

import (
	"context"
	"fmt"
)

// ExampleNewDiffn demonstrates the `Diffn` (non-overlap in 2D) constraint
// and how it prunes position domains for rectangular objects.
//
// We place two 2×2 squares; the first has fixed Y=1 and X=1 while the second
// also has Y=1 but X is free in {1,2,3,4}. Posting `Diffn` enforces that the
// two rectangles do not overlap in the plane. After a brief propagation run
// the domain of X2 is tightened to {3,4} (so X2 ≥ 3) to avoid overlap with
// the fixed first square. The example prints the domain of X2 to demonstrate
// this pruning effect.
func ExampleNewDiffn() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "x1")
	x1 := model.IntVarValues([]int{1}, "x1")
	// y1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y1")
	y1 := model.IntVarValues([]int{1}, "y1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}), "x2")
	x2 := model.IntVarValues([]int{1, 2, 3, 4}, "x2")
	// y2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y2")
	y2 := model.IntVarValues([]int{1}, "y2")

	_, _ = NewDiffn(model, []*FDVariable{x1, x2}, []*FDVariable{y1, y2}, []int{2, 2}, []int{2, 2})

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x2: {3..4}
}
