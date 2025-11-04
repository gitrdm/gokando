```go
func ExampleNewNValue() {
	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x2")
	// Exact NValue=1 ⇒ NPlus1=2
	nPlus1 := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "N+1")

	_, _ = NewNValue(model, []*FDVariable{x1, x2}, nPlus1)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0)

	// No pruning here, but the composition is established and will prune
	// as soon as one side gets fixed by other constraints or decisions.
	fmt.Printf("x1: %s\n", solver.GetDomain(nil, x1.ID()))
	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
}

```


