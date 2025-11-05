```go
func Example_hlapi_gcc() {
	m := NewModel()
	a := m.IntVar(1, 1, "a") // fixed to 1
	b := m.IntVar(1, 2, "b")
	c := m.IntVar(1, 2, "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3
	_ = m.GlobalCardinality([]*FDVariable{a, b, c}, min, max)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	fmt.Println("a:", s.GetDomain(nil, a.ID()))
	fmt.Println("b:", s.GetDomain(nil, b.ID()))
	fmt.Println("c:", s.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}

```


