```go
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

```


