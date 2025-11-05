```go
func ExampleNewArithmetic_negative() {
	model := NewModel()

	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 5, 8}))
	x := model.IntVarValues([]int{3, 5, 8}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X - 2 (using negative offset)
	c, err := NewArithmetic(x, y, -2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x - 2)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=3, y=1 (y = x - 2)
	// x=5, y=3 (y = x - 2)
	// x=8, y=6 (y = x - 2)
}

```


