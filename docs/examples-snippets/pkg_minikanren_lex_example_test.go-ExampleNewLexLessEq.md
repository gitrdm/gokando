```go
func ExampleNewLexLessEq() {
	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{1, 2, 3}), "x2")
	y1 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{3, 4, 5}), "y1")
	y2 := model.NewVariableWithName(NewBitSetDomainFromValues(9, []int{2, 3, 4}), "y2")

	c, _ := NewLexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Run fixed-point propagation via a zero-solution search (limit=0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("y1: %s\n", solver.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}

```


