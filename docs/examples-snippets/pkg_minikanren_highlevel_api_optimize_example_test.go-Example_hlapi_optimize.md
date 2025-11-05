```go
func Example_hlapi_optimize() {
	m := NewModel()
	xs := m.IntVars(2, 1, 3, "x") // x1, x2 in [1..3]
	total := m.IntVar(0, 10, "t")
	_ = m.LinearSum(xs, []int{1, 2}, total) // t = x1 + 2*x2

	// Minimize t
	_, best, err := Optimize(m, total, true)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best=%d\n", best)
	// Output:
	// best=3
}

```


