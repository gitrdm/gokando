```go
func ExampleNewAmong() {
	model := NewModel()

	// Low-level API (kept as comments):
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{2, 3}), "x2")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{3, 4}), "x3")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	// K encodes count+1; here we want exactly 1 variable in S → K={2}
	// k := model.NewVariableWithName(NewBitSetDomainFromValues(4, []int{2}), "K")
	k := model.IntVarValues([]int{2}, "K")

	// S = {1,2}. With K=1 (encoded 2) and x1⊆S, x2 is forced OUT of S
	// Low-level API (kept as comment):
	// c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	// model.AddConstraint(c)
	// HLAPI:
	_ = model.Among([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}

```


