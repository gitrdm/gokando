```go
func ExampleSLGEngine_Evaluate() {
	engine := NewSLGEngine(nil)

	// Define a call pattern for a "fact" predicate
	pattern := NewCallPattern("color", []Term{NewAtom("x")})

	// Simple evaluator that produces three color answers
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		colors := []string{"red", "green", "blue"}
		for _, color := range colors {
			answer := map[int64]Term{1: NewAtom(color)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Collect all answers
	count := 0
	for range resultChan {
		count++
	}

	fmt.Printf("Derived %d answers\n", count)

	// Second evaluation should hit cache
	resultChan2, _ := engine.Evaluate(ctx, pattern, evaluator)
	for range resultChan2 {
	}

	stats := engine.Stats()
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)

	// Output:
	// Derived 3 answers
	// Cache hits: 1
}

```


