```go
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

```


