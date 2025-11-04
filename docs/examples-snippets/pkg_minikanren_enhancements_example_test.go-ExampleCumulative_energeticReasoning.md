```go
func ExampleCumulative_energeticReasoning() {
	m := NewModel()
	// Three heavy tasks that cannot fit in the time window
	// Tasks: each dur=4, dem=3, capacity=5, window=[1..6]
	// Energy required: 3 * 4 * 3 = 36 work units
	// Energy available: 6 time * 5 capacity = 30 work units → OVERLOAD
	s1 := m.NewVariable(NewBitSetDomain(3))
	s2 := m.NewVariable(NewBitSetDomain(3))
	s3 := m.NewVariable(NewBitSetDomain(3))

	cum, _ := NewCumulative(
		[]*FDVariable{s1, s2, s3},
		[]int{4, 4, 4}, // durations
		[]int{3, 3, 3}, // demands
		5,              // capacity
	)
	m.AddConstraint(cum)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sols, _ := solver.Solve(ctx, 1)
	fmt.Printf("Solutions found: %d (energetic reasoning detects overload)\n", len(sols))
	// Output: Solutions found: 0 (energetic reasoning detects overload)
}

```


