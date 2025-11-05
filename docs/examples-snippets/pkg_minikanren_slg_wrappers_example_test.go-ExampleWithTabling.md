```go
func ExampleWithTabling() {
	engine := NewSLGEngine(nil)
	eval := WithTabling(engine)

	// Simple evaluator that yields a single answer
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{1: NewAtom("ok")}
		return nil
	})

	ch, err := eval(context.Background(), "demo", []Term{NewAtom("x")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	stats := engine.Stats()
	fmt.Printf("evaluations=%d cached=%d\n", stats.TotalEvaluations, stats.CachedSubgoals)
	// Output:
	// evaluations=1 cached=1
}

```


