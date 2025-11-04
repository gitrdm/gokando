```go
func ExampleNewRegular() {
	// Build DFA: states 1=start, 2=last=1, 3=last=2; accept={2}
	numStates, start, accept, delta := buildEndsWith1DFA()

	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomain(2), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomain(2), "x2")
	x3 := model.NewVariableWithName(NewBitSetDomain(2), "x3")

	c, _ := NewRegular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)
	model.AddConstraint(c)
	solver := NewSolver(model)

	st, _ := solver.propagate(nil)
	fmt.Println("x1:", solver.GetDomain(st, x1.ID()))
	fmt.Println("x2:", solver.GetDomain(st, x2.ID()))
	fmt.Println("x3:", solver.GetDomain(st, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}

```


