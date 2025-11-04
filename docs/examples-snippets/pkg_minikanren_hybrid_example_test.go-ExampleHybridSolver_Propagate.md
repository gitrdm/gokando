```go
func ExampleHybridSolver_Propagate() {
	// Create FD model
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))

	// x + 2 = y
	arith, _ := NewArithmetic(x, y, 2)
	model.AddConstraint(arith)

	// Create solver and baseline store from model helper, then override domains
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{3, 4, 5}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Run propagation
	result, _ := solver.Propagate(store)

	// Check propagated domains
	yDomain := result.GetDomain(y.ID())
	fmt.Printf("After propagation, y domain size: %d\n", yDomain.Count())
	fmt.Printf("y contains 5: %v\n", yDomain.Has(5))
	fmt.Printf("y contains 6: %v\n", yDomain.Has(6))
	fmt.Printf("y contains 7: %v\n", yDomain.Has(7))

	// Output:
	// After propagation, y domain size: 3
	// y contains 5: true
	// y contains 6: true
	// y contains 7: true
}

```


