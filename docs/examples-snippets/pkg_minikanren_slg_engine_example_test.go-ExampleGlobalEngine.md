```go
func ExampleGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	// Get global engine (created on first access)
	engine1 := GlobalEngine()
	engine2 := GlobalEngine()

	if engine1 == engine2 {
		fmt.Println("Same engine instance")
	}

	// Evaluate using global engine
	pattern := NewCallPattern("global", []Term{NewAtom("test")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("answer")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine1.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	// State is shared
	stats := engine2.Stats()
	fmt.Printf("Shared state - evaluations: %d\n", stats.TotalEvaluations)

	// Output:
	// Same engine instance
	// Shared state - evaluations: 1
}

```


