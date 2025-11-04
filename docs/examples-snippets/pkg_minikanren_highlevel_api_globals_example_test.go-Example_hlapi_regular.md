```go
func Example_hlapi_regular() {
	// Build DFA: accepts sequences ending with symbol 1 over alphabet {1,2}
	numStates, start, accept, delta := endsWith1DFA()

	m := NewModel()
	// x1 := m.NewVariableWithName(NewBitSetDomain(2), "x1")
	x1 := m.IntVar(1, 2, "x1")
	// x2 := m.NewVariableWithName(NewBitSetDomain(2), "x2")
	x2 := m.IntVar(1, 2, "x2")
	// x3 := m.NewVariableWithName(NewBitSetDomain(2), "x3")
	x3 := m.IntVar(1, 2, "x3")
	_ = m.Regular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)
	fmt.Println("x1:", s.GetDomain(nil, x1.ID()))
	fmt.Println("x2:", s.GetDomain(nil, x2.ID()))
	fmt.Println("x3:", s.GetDomain(nil, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}

```


