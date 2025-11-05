```go
func ExampleNewDiffn() {
	model := NewModel()
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "x1")
	x1 := model.IntVarValues([]int{1}, "x1")
	// y1 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y1")
	y1 := model.IntVarValues([]int{1}, "y1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}), "x2")
	x2 := model.IntVarValues([]int{1, 2, 3, 4}, "x2")
	// y2 := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "y2")
	y2 := model.IntVarValues([]int{1}, "y2")

	_, _ = NewDiffn(model, []*FDVariable{x1, x2}, []*FDVariable{y1, y2}, []int{2, 2}, []int{2, 2})

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x2: {3..4}
}

```


