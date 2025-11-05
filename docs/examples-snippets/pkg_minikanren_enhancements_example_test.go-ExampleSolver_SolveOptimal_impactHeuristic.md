```go
func ExampleSolver_SolveOptimal_impactHeuristic() {
	m := NewModel()
	// Minimize total cost: cost = 2*x + 3*y
	x := m.NewVariable(NewBitSetDomain(4))
	y := m.NewVariable(NewBitSetDomain(4))
	cost := m.NewVariable(NewBitSetDomain(30))

	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{2, 3}, cost)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Use impact-based heuristic to focus on objective-relevant variables
	sol, objVal, _ := solver.SolveOptimalWithOptions(ctx, cost, true,
		WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))

	fmt.Printf("Minimum cost: %d (x=%d, y=%d)\n", objVal, sol[x.ID()], sol[y.ID()])
	// Output: Minimum cost: 5 (x=1, y=1)
}

```


