```go
func ExampleSolver_parallelSearch() {
	model := minikanren.NewModel()
	model.NewVariables(3, minikanren.NewBitSetDomain(5))

	// CORRECT: Single model shared by all workers (zero GC cost)
	// Each worker creates its own Solver with independent state chains
	worker1Solver := minikanren.NewSolver(model)
	worker2Solver := minikanren.NewSolver(model)

	// Both solvers share the same immutable model
	fmt.Printf("Worker 1 model variables: %d\n", worker1Solver.Model().VariableCount())
	fmt.Printf("Worker 2 model variables: %d\n", worker2Solver.Model().VariableCount())
	fmt.Printf("Models are shared (same pointer): %v\n", worker1Solver.Model() == worker2Solver.Model())

	// State changes are isolated per worker via copy-on-write SolverState
	// No model cloning needed - this is the key architectural insight

	// Output:
	// Worker 1 model variables: 3
	// Worker 2 model variables: 3
	// Models are shared (same pointer): true
}

```


