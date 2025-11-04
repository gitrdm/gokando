```go
func ExampleNewInequality_lessThan() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{2}))
	y := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2 < y=3
	// x=2 < y=4
	// x=2 < y=5
}

```


