```go
func ExampleSolver_SolveOptimal_boolSum() {
	m := NewModel()
	// Maximize the number of satisfied conditions (booleans set to true)
	b1 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b2 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b3 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	count := m.NewVariable(NewBitSetDomain(4)) // encoded count+1

	bs, _ := NewBoolSum([]*FDVariable{b1, b2, b3}, count)
	m.AddConstraint(bs)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize count (all booleans true)
	sol, objVal, _ := solver.SolveOptimal(ctx, count, false)
	actualCount := objVal - 1 // decode from encoded value
	fmt.Printf("Maximum count: %d (all satisfied: %v)\n", actualCount,
		sol[b1.ID()] == 2 && sol[b2.ID()] == 2 && sol[b3.ID()] == 2)
	// Output: Maximum count: 3 (all satisfied: true)
}

```


