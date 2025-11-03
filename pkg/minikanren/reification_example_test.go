package minikanren

import (
	"context"
	"fmt"
)

// ExampleReifiedConstraint shows how to reify a constraint into a boolean.
//
// We reify the arithmetic equality X + 0 = Y into B, where B∈{1=false,2=true}.
func ExampleReifiedConstraint() {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(3), "X")
	y := model.NewVariableWithName(NewBitSetDomain(3), "Y")
	b := model.NewVariableWithName(NewBitSetDomain(2), "B") // {1,2} maps to {false,true}

	arith, _ := NewArithmetic(x, y, 0) // X + 0 = Y
	reified, _ := NewReifiedConstraint(arith, b)
	model.AddConstraint(reified)

	solver := NewSolver(model)
	solutions, _ := solver.Solve(context.Background(), 5)

	for i := 0; i < len(solutions) && i < 3; i++ {
		sol := solutions[i]
		fmt.Printf("X=%d Y=%d B=%t\n", sol[x.ID()], sol[y.ID()], sol[b.ID()] == 2)
	}
	// (Output omitted; solution order is not guaranteed.)
}
