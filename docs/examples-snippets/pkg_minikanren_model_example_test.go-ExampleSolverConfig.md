```go
func ExampleSolverConfig() {
	// Create config with domain-over-degree heuristic
	config := &minikanren.SolverConfig{
		VariableHeuristic: minikanren.HeuristicDomDeg,
		ValueHeuristic:    minikanren.ValueOrderAsc,
		RandomSeed:        42,
	}

	model := minikanren.NewModelWithConfig(config)
	vars := model.NewVariables(4, minikanren.NewBitSetDomain(4))
	_ = vars

	fmt.Printf("Model config: %+v\n", model.Config())

	// Output:
	// Model config: &{VariableHeuristic:0 ValueHeuristic:0 RandomSeed:42}
}

```


