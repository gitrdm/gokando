package minikanren

import (
	"fmt"
)

// ExampleNewTable demonstrates the Table (extensional) global constraint.
//
// We model two variables x,y and restrict the pair (x,y) to be one of the
// allowed rows: (1,1), (2,3), (3,2). Given y ∈ {1,2}, propagation prunes x to
// {1,3} because only rows (1,1) and (3,2) are compatible with the current y.
func ExampleNewTable() {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(5), "x")
	y := model.NewVariableWithName(NewBitSetDomain(5), "y")

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	c, _ := NewTable([]*FDVariable{x, y}, rows)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Set y ∈ {1,2}
	state, _ := solver.SetDomain(nil, y.ID(), NewBitSetDomainFromValues(5, []int{1, 2}))

	// Propagate once; solver runs to fixed-point internally during Solve, but
	// we can invoke the constraint directly for illustration.
	newState, _ := solver.propagate(state)

	xd := solver.GetDomain(newState, x.ID())
	yd := solver.GetDomain(newState, y.ID())

	fmt.Printf("x: %v\n", xd)
	fmt.Printf("y: %v\n", yd)
	// Output:
	// x: {1,3}
	// y: {1..2}
}
