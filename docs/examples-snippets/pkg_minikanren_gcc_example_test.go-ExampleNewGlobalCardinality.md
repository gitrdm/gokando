```go
func ExampleNewGlobalCardinality() {
	model := NewModel()

	a := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "a")
	b := model.NewVariableWithName(NewBitSetDomain(2), "b")
	c := model.NewVariableWithName(NewBitSetDomain(2), "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3

	gcc, err := NewGlobalCardinality([]*FDVariable{a, b, c}, min, max)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(gcc)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Println("a:", solver.GetDomain(nil, a.ID()))
	fmt.Println("b:", solver.GetDomain(nil, b.ID()))
	fmt.Println("c:", solver.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}

```


