```go
func ExampleSLGEngine_Evaluate_streaming() {
	engine := NewSLGEngine(nil)

	pattern := NewCallPattern("range", []Term{NewAtom(5)})

	// Evaluator that produces answers incrementally
	evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
		for i := 1; i <= 5; i++ {
			answer := map[int64]Term{1: NewAtom(i)}
			answers <- answer
		}
		return nil
	}

	ctx := context.Background()
	resultChan, _ := engine.Evaluate(ctx, pattern, evaluator)

	// Process answers as they arrive
	for answer := range resultChan {
		value := answer[1]
		fmt.Printf("Got answer: %v\n", value)
	}

	// Output:
	// Got answer: 1
	// Got answer: 2
	// Got answer: 3
	// Got answer: 4
	// Got answer: 5
}

```


