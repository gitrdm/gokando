```go
func ExampleSLGEngine_Stats() {
	engine := NewSLGEngine(nil)

	// Evaluate several subgoals
	for i := 1; i <= 3; i++ {
		pattern := NewCallPattern("test", []Term{NewAtom(i)})
		evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
			answer := map[int64]Term{1: NewAtom(fmt.Sprintf("result%d", i))}
			answers <- answer
			return nil
		}

		resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
		for range resultChan {
		}
	}

	// Re-evaluate first subgoal (cache hit)
	pattern := NewCallPattern("test", []Term{NewAtom(1)})
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answer := map[int64]Term{1: NewAtom("result1")}
		answers <- answer
		return nil
	}
	resultChan, _ := engine.Evaluate(context.Background(), pattern, evaluator)
	for range resultChan {
	}

	stats := engine.Stats()
	fmt.Printf("Total evaluations: %d\n", stats.TotalEvaluations)
	fmt.Printf("Cached subgoals: %d\n", stats.CachedSubgoals)
	fmt.Printf("Cache hits: %d\n", stats.CacheHits)
	fmt.Printf("Cache misses: %d\n", stats.CacheMisses)
	fmt.Printf("Hit ratio: %.2f\n", stats.HitRatio)

	// Output:
	// Total evaluations: 4
	// Cached subgoals: 3
	// Cache hits: 1
	// Cache misses: 3
	// Hit ratio: 0.25
}

```


