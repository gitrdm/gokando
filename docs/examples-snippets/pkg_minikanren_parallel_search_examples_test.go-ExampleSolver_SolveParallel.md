```go
func ExampleSolver_SolveParallel() {
	// 1) Build a model with 3 variables, each from 1..3
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))

	// 2) Constrain them to be all different
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	// 3) Create a solver and run in parallel with up to 4 workers
	solver := NewSolver(model)
	ctx := context.Background()

	// Find up to 6 solutions in parallel (there are 3! = 6)
	solutions, err := solver.SolveParallel(ctx, 4, 6)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("solutions: %d\n", len(solutions))
	// Output:
	// solutions: 6
}

```


