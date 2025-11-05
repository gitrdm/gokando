```go
func ExampleModel_helpers_allDifferent() {
	m := NewModel()
	xs := m.IntVars(3, 1, 3, "x")
	_ = m.AllDifferent(xs...)

	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 6
}

```


