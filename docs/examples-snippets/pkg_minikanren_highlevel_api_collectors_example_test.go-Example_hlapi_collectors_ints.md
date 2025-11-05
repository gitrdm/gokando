```go
func Example_hlapi_collectors_ints() {
	x := Fresh("x")
	goal := Disj(Eq(x, A(1)), Eq(x, A(3)), Eq(x, A(5)))
	vals := Ints(goal, x)
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range vals {
		sum += v
	}
	fmt.Printf("%d %d\n", len(vals), sum)
	// Output:
	// 3 9
}

```


