```go
func Example_hlapi_optimize_withOptions() {
	m := NewModel()
	xs := m.IntVars(2, 1, 3, "x")
	total := m.IntVar(0, 10, "t")
	_ = m.LinearSum(xs, []int{1, 2}, total)

	// Use context and one option as a smoke test for the wrapper
	ctx := context.Background()
	_, best, err := OptimizeWithOptions(ctx, m, total, true, WithParallelWorkers(2))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

```


