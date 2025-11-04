```go
func ExampleNewElementValues() {
	model := NewModel()

	// index initially in [1..5]
	// low-level: idx := model.NewVariable(NewBitSetDomain(5))
	idx := model.IntVar(1, 5, "idx")
	// result initially in [1..10]
	// low-level: res := model.NewVariable(NewBitSetDomain(10))
	res := model.IntVar(1, 10, "res")

	vals := []int{2, 4, 4, 7, 9}
	c, _ := NewElementValues(idx, vals, res)
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Force result to be either 4 or 7; this should prune index to {2,3,4}
	state := (*SolverState)(nil)
	state, _ = solver.SetDomain(state, res.ID(), NewBitSetDomainFromValues(10, []int{4, 7}))

	// Trigger propagation directly and inspect the resulting state domains.
	newState, err := c.Propagate(solver, state)
	if err != nil {
		// No solution under these restrictions (shouldn't happen here)
		fmt.Println("propagation error:", err)
		return
	}

	idxDom := solver.GetDomain(newState, idx.ID())
	resDom := solver.GetDomain(newState, res.ID())

	fmt.Printf("idx=%v res=%v\n", idxDom, resDom)
	// Output:
	// idx={2..4} res={4,7}
}

```


