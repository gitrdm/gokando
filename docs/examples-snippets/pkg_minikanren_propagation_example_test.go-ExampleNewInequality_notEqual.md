```go
func ExampleNewInequality_notEqual() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomain(3))
	y := model.NewVariable(NewBitSetDomain(3))

	// Enforce: X ≠ Y
	c, err := NewInequality(x, y, NotEqual)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 3) // Get first 3 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d\n", i+1, sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2
	// Solution 2: x=1, y=3
	// Solution 3: x=2, y=1
}

```


