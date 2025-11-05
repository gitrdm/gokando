```go
func ExampleNewCumulative() {
	model := NewModel()

	// Task A: fixed at start=2, duration=2, demand=2
	// A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	A := model.IntVarValues([]int{2}, "A")
	// Task B: start in [1..4], duration=2, demand=1
	// B := model.NewVariableWithName(NewBitSetDomain(4), "B")
	B := model.IntVar(1, 4, "B")

	// Low-level API (kept as comment):
	// cum, err := NewCumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)
	// if err != nil {
	//     panic(err)
	// }
	// model.AddConstraint(cum)
	// HLAPI wrapper:
	_ = model.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	// If you only need concrete solutions (assignments), the HLAPI helper
	// SolveN(ctx, model, maxSolutions) is a convenient wrapper that creates
	// a solver, runs the search, and returns solutions. Example:
	//
	//    sols, err := SolveN(ctx, model, 1)
	//
	// However, when you want to inspect solver internals (domains after
	// propagation) or call methods like GetDomain/propagate, create the
	// Solver explicitly as done below and call Solve on it. That allows
	// reading the pruned domains from the solver state.
	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Propagate at root by running a one-solution search (will stop at root if none).
	_, _ = solver.Solve(ctx, 1)

	fmt.Println("A:", solver.GetDomain(nil, A.ID()))
	fmt.Println("B:", solver.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

```


