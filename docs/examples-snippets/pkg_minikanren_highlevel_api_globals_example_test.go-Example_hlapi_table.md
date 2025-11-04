```go
func Example_hlapi_table() {
	m := NewModel()
	x := m.NewVariableWithName(NewBitSetDomain(5), "x")
	// y ∈ {1,2} upfront so we can avoid internal propagation calls
	y := m.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "y")

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	_ = m.Table([]*FDVariable{x, y}, rows)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	xd := s.GetDomain(nil, x.ID())
	yd := s.GetDomain(nil, y.ID())

	fmt.Printf("x: %v\n", xd)
	fmt.Printf("y: %v\n", yd)
	// Output:
	// x: {1,3}
	// y: {1..2}
}

```


