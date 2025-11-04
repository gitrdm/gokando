```go
func ExampleNewSolver() {
	// Create a model: 3 variables, each can be 1, 2, or 3
	model := minikanren.NewModel()
	vars := model.NewVariables(3, minikanren.NewBitSetDomain(3))

	// Add constraint: all variables must be different
	// (Note: We'll use the new architecture in future phases)
	_ = vars // Constraints not yet integrated in this phase

	// Create a solver
	solver := minikanren.NewSolver(model)

	// Solve with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d solutions\n", len(solutions))
	if len(solutions) > 0 {
		fmt.Printf("First solution: %v\n", solutions[0])
	}

	// Output will vary based on solver implementation
	// Output:
	// Found 27 solutions
	// First solution: [1 1 1]
}

```


