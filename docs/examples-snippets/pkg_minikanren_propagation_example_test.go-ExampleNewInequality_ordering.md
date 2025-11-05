```go
func ExampleNewInequality_ordering() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y < Z (ascending order)
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewInequality(y, z, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 5) // Get first 5 solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d < z=%d\n", sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// x=1 < y=2 < z=3
	// x=1 < y=2 < z=4
	// x=1 < y=2 < z=5
	// x=1 < y=3 < z=4
	// x=1 < y=3 < z=5
}

```


