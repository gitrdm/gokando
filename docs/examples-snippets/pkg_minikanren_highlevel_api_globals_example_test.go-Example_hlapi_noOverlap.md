```go
func Example_hlapi_noOverlap() {
	m := NewModel()
	s := m.IntVarsWithNames([]string{"s1", "s2"}, 1, 3)
	_ = m.NoOverlap(s, []int{2, 2})

	// Enumerate solutions; only (1,3) and (3,1) are valid starts
	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 2
}

```


