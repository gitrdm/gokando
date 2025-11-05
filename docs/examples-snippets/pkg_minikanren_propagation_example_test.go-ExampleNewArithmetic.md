```go
func ExampleNewArithmetic() {
	model := NewModel()

	// Create variables with specific domains
	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 5, 7}))
	x := model.IntVarValues([]int{2, 5, 7}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X + 3
	c, err := NewArithmetic(x, y, 3)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0) // Get all solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x + 3)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2, y=5 (y = x + 3)
	// x=5, y=8 (y = x + 3)
	// x=7, y=10 (y = x + 3)
}

```


