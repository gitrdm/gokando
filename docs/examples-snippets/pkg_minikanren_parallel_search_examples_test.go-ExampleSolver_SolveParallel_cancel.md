```go
func ExampleSolver_SolveParallel_cancel() {
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)

	// Cancel the context immediately to deterministically demonstrate cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Because we cancelled before calling, SolveParallel should return quickly
	// with an error; print a short message to make the example deterministic.
	_, err := solver.SolveParallel(ctx, 8, 0)
	if err != nil {
		fmt.Println("cancelled")
	} else {
		fmt.Println("no-error")
	}
	// Output:
	// cancelled
}

```


