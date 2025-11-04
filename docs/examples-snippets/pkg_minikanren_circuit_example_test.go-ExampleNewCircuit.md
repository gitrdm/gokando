```go
func ExampleNewCircuit() {
	model := NewModel()
	n := 4

	// succ[i] ∈ [1..n]
	succ := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		succ[i] = model.NewVariableWithName(NewBitSetDomain(n), fmt.Sprintf("succ_%d", i+1))
	}

	// Build Circuit with start at node 1
	c, _ := NewCircuit(model, succ, 1)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Run propagation
	newState, _ := solver.propagate(nil)

	// Inspect two successor domains to see self-loop removal
	d1 := solver.GetDomain(newState, succ[0].ID())
	d2 := solver.GetDomain(newState, succ[1].ID())
	fmt.Printf("succ1=%s\n", d1.String())
	fmt.Printf("succ2=%s\n", d2.String())

	// Output:
	// succ1={2..4}
	// succ2={1,3,4}
}

```


