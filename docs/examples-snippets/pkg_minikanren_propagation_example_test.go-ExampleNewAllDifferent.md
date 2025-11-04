```go
func ExampleNewAllDifferent() {
	model := NewModel()

	// Create three variables with domain {1, 2, 3}
	x := model.NewVariable(NewBitSetDomain(3))
	y := model.NewVariable(NewBitSetDomain(3))
	z := model.NewVariable(NewBitSetDomain(3))

	// Ensure all three variables have different values
	c, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 2) // Get first 2 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d, z=%d\n", i+1, sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2, z=3
	// Solution 2: x=1, y=3, z=2
}

```


