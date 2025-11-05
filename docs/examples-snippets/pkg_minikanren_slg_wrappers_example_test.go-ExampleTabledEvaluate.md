```go
func ExampleTabledEvaluate() {
	// Use the global engine implicitly
	ResetGlobalEngine()
	inner := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{42: NewAtom(1)}
		return nil
	})

	ch, err := TabledEvaluate(context.Background(), "test", []Term{NewAtom("a")}, inner)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for range ch { /* drain */
	}

	fmt.Println("ok")
	// Output:
	// ok
}

```


