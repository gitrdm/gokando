package minikanren

import (
	"context"
	"fmt"
)

// ExampleSolver_SolveOptimal demonstrates minimizing a linear objective.
func ExampleSolver_SolveOptimal() {
	model := NewModel()
	// x,y in {1,2,3}
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	// total T = x + 2*y
	tvar := model.NewVariable(NewBitSetDomain(20))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), tvar, true)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best objective: %d\n", obj)
	_ = sol // values per variable in model order
	// Output:
	// best objective: 3
}

// ExampleSolver_SolveOptimalWithOptions demonstrates using options such as parallel workers.
// The example finds the minimum value of x + 2y where x,y are in [0,9].
func ExampleSolver_SolveOptimalWithOptions() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	tvar := model.NewVariable(NewBitSetDomain(40))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	// Use parallel workers without timeout for deterministic results
	ctx := context.Background()
	sol, best, err := solver.SolveOptimalWithOptions(ctx, tvar, true, WithParallelWorkers(4))
	_ = sol // solution slice omitted in example output for brevity
	
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

// ExampleSolver_SolveOptimal_MinOfArray shows maximizing the minimum of two variables.
func ExampleSolver_SolveOptimal_minOfArray() {
	model := NewModel()
	// Two variables with overlapping ranges
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 4, 5}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 4, 5, 6, 7}))
	// r = min(x,y)
	r := model.NewVariable(NewBitSetDomain(10))
	c, _ := NewMin([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Maximize the minimum value achievable across x and y
	_, best, _ := solver.SolveOptimal(context.Background(), r, false)
	fmt.Println("max min:", best)
	// Output:
	// max min: 5
}
