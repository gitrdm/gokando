package minikanren

import (
	"context"
	"fmt"
)

// ExampleCount demonstrates how to count how many variables equal a target value.
//
// We build a small model with three variables X,Y,Z in {1,2,3}, then post
// Count([X,Y,Z], value=2, N). The count variable N is encoded as N=actual+1
// due to 1-indexed domains, so N∈[1..4] represents counts 0..3.
func ExampleCount() {
	model := NewModel()
	dom := NewBitSetDomain(3)
	x := model.NewVariableWithName(dom, "X")
	y := model.NewVariableWithName(dom, "Y")
	z := model.NewVariableWithName(dom, "Z")
	// N encodes count+1, therefore use domain [1..4]
	N := model.NewVariableWithName(NewBitSetDomain(4), "N")

	// Post Count constraint: number of vars equal to 2
	_, _ = NewCount(model, []*FDVariable{x, y, z}, 2, N)

	solver := NewSolver(model)
	solutions, _ := solver.Solve(context.Background(), 10)

	// Print a couple of solutions
	for i := 0; i < 3 && i < len(solutions); i++ {
		sol := solutions[i]
		fmt.Printf("X=%d Y=%d Z=%d count=%d\n", sol[x.ID()], sol[y.ID()], sol[z.ID()], sol[N.ID()]-1)
	}
	// (Output omitted; solution order is not guaranteed.)
}
