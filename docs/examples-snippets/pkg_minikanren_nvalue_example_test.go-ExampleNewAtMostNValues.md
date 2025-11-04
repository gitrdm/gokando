```go
func ExampleNewAtMostNValues() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1}), "x1")
	x1 := model.IntVarValues([]int{1}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	x2 := model.IntVarValues([]int{1, 2}, "x2")
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x3")
	x3 := model.IntVarValues([]int{1, 2}, "x3")
	// low-level: limit := model.NewVariableWithName(NewBitSetDomain(2), "limit") // distinct ≤ 1
	// HLAPI: express the same compact integer domain using IntVar
	limit := model.IntVar(1, 2, "limit") // distinct ≤ 1 encoded over {1,2}

	_, _ = NewAtMostNValues(model, []*FDVariable{x1, x2, x3}, limit)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate only

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {1}
	// x3: {1}
}

```


