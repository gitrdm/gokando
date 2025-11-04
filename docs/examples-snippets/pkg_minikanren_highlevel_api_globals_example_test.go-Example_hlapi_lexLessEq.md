```go
func Example_hlapi_lexLessEq() {
	m := NewModel()
	// Use compact range helpers instead of explicit value sets
	x1 := m.IntVar(2, 4, "x1")
	x2 := m.IntVar(1, 3, "x2")
	y1 := m.IntVar(3, 5, "y1")
	y2 := m.IntVar(2, 4, "y2")
	_ = m.LexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})

	// Minimal propagation: background context and solution cap 0
	s := NewSolver(m)
	_, _ = s.Solve(context.Background(), 0)

	fmt.Printf("y1: %s\n", s.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}

```


