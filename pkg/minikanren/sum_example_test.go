package minikanren_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleNewLinearSum demonstrates a weighted sum equality with bounds
// propagation. It prunes the total and variables based on feasible ranges.
func ExampleNewLinearSum() {
	model := NewModel()

	// Three variables with small ranges
	a := model.NewVariable(NewBitSetDomain(5)) // [1..5]
	b := model.NewVariable(NewBitSetDomain(5)) // [1..5]
	c := model.NewVariable(NewBitSetDomain(9)) // [1..9]

	// Total starts wide and will be pruned
	total := model.NewVariable(NewBitSetDomain(100))

	coeffs := []int{1, 2, 3}
	ls, err := NewLinearSum([]*FDVariable{a, b, c}, coeffs, total)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Solve without search to showcase propagation effects only.
	// The first propagation pass happens before search begins.
	solutions, _ := solver.Solve(ctx, 1)
	_ = solutions // not used; we print domains instead

	// Read pruned domains from root propagated state
	aDom := solver.GetDomain(nil, a.ID())
	bDom := solver.GetDomain(nil, b.ID())
	cDom := solver.GetDomain(nil, c.ID())
	tDom := solver.GetDomain(nil, total.ID())

	fmt.Printf("a=[%d..%d] b=[%d..%d] c=[%d..%d] total=[%d..%d]\n",
		aDom.Min(), aDom.Max(), bDom.Min(), bDom.Max(), cDom.Min(), cDom.Max(), tDom.Min(), tDom.Max())

	// Output:
	// a=[1..5] b=[1..5] c=[1..9] total=[6..42]
}
