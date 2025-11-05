```go
func ExampleNewAmong_hybrid() {
	model := NewModel()

	x1 := model.IntVarValues([]int{1, 2}, "x1")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	k := model.IntVarValues([]int{2}, "K")

	// Build the propagation constraint and register it with the model so the FD plugin can discover it.
	c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	model.AddConstraint(c)

	// Use HLAPI helper to build a HybridSolver and a UnifiedStore populated
	// from the model (domains + constraints). This reduces boilerplate.
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	// Run propagation to a fixed point.
	result, _ := solver.Propagate(store)

	fmt.Printf("x2: %s\n", result.GetDomain(x2.ID()))
	fmt.Printf("x3: %s\n", result.GetDomain(x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}

```


