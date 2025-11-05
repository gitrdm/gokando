```go
func ExampleResetGlobalEngine() {
	// Reset to ensure clean state for this example
	ResetGlobalEngine()

	engine := GlobalEngine()

	// Add some state
	pattern := NewCallPattern("temp", []Term{NewAtom("x")})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("data")}
		answers <- answer
		return nil
	}

	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	statsBefore := engine.Stats()
	fmt.Printf("Before reset - evaluations: %d\n", statsBefore.TotalEvaluations)

	// Reset state
	ResetGlobalEngine()

	statsAfter := engine.Stats()
	fmt.Printf("After reset - evaluations: %d\n", statsAfter.TotalEvaluations)
	fmt.Printf("After reset - cached subgoals: %d\n", statsAfter.CachedSubgoals)

	// Output:
	// Before reset - evaluations: 1
	// After reset - evaluations: 0
	// After reset - cached subgoals: 0
}

```


