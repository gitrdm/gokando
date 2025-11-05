```go
func Example_hlapi_cumulative() {
	m := NewModel()
	A := m.IntVar(2, 2, "A") // fixed start=2
	B := m.IntVar(1, 4, "B") // start in [1..4]
	_ = m.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 1) // trigger propagation

	fmt.Println("A:", s.GetDomain(nil, A.ID()))
	fmt.Println("B:", s.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


