```go
func ExampleNewNoOverlap() {
	model := NewModel()

	// Task A fixed at start=2, duration=2 ⇒ executes over [2,3]
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B can start in [1..4], duration=2
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	noov, err := NewNoOverlap([]*FDVariable{A, B}, []int{2, 2})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(noov)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root via a short search
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


