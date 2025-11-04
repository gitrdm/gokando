```go
func Example_hlapi_values_projection() {
	x := Fresh("x")
	goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
	sols := Solutions(goal, x)
	ints := ValuesInt(sols, "x")
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range ints {
		sum += v
	}
	fmt.Printf("%d %d\n", len(ints), sum)
	// Output:
	// 2 3
}

```


