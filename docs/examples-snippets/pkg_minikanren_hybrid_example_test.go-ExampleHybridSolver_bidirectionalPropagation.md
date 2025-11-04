```go
func ExampleHybridSolver_bidirectionalPropagation() {
	// Create FD model with AllDifferent constraint
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	// FD constraint: all different
	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Initial state: all variables have domain {1, 2, 3}
	store := NewUnifiedStore()
	domain := NewBitSetDomainFromValues(10, []int{1, 2, 3})
	store, _ = store.SetDomain(x.ID(), domain)
	store, _ = store.SetDomain(y.ID(), domain)
	store, _ = store.SetDomain(z.ID(), domain)

	// HYBRID STEP 1: Relational solver binds x to 2 (e.g., from unification)
	// This is the key: a relational binding influences FD domains
	store, _ = store.AddBinding(int64(x.ID()), NewAtom(2))

	// Run hybrid propagation
	result, _ := solver.Propagate(store)

	// HYBRID RESULT 1: x's FD domain pruned to {2} (relational → FD)
	xDom := result.GetDomain(x.ID())
	fmt.Printf("x domain after relational binding: {%d}\n", xDom.SingletonValue())

	// HYBRID RESULT 2: AllDifferent removes 2 from y and z (FD propagation)
	yDom := result.GetDomain(y.ID())
	fmt.Printf("y domain size after AllDifferent: %d\n", yDom.Count())
	fmt.Printf("y contains 2: %v\n", yDom.Has(2))

	// HYBRID RESULT 3: x's binding exists (FD singleton promoted back to relational)
	xBinding := result.GetBinding(int64(x.ID()))
	fmt.Printf("x has relational binding: %v\n", xBinding != nil)

	// Output:
	// x domain after relational binding: {2}
	// y domain size after AllDifferent: 2
	// y contains 2: false
	// x has relational binding: true
}

```


