```go
func ExampleNewStretch() {
	model := NewModel()
	// Domains over {1,2}; constrain value 1 to appear only in runs of length exactly 2.
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "x2") // fix a 1
	x2 := model.IntVarValues([]int{1}, "x2") // fix a 1
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "x3") // separator
	x3 := model.IntVarValues([]int{2}, "x3") // separator
	// x4 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "x4") // fix a 1
	x4 := model.IntVarValues([]int{1}, "x4") // fix a 1
	// x5 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1, 2}), "x5")
	x5 := model.IntVarValues([]int{1, 2}, "x5")

	_, _ = NewStretch(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, []int{2}, []int{2})

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// Single 1s at x2 and x4, with a separator at x3, force x1 and x5 to 1 to satisfy run length = 2.
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x5: %s\n", solver.GetDomain(nil, x5.ID()))
	// Output:
	// x1: {1}
	// x5: {1}
}

```


