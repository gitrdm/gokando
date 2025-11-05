```go
func ExampleSolver_SolveParallel_limit() {
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)
	ctx := context.Background()

	// Ask for at most 3 solutions (there are 4! = 24 total)
	solutions, err := solver.SolveParallel(ctx, 4, 3)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 3
}

```


