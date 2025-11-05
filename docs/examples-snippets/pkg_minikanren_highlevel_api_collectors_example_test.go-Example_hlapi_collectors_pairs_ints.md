```go
func Example_hlapi_collectors_pairs_ints() {
	x, y := Fresh("x"), Fresh("y")
	goal := Disj(
		Conj(Eq(x, A(1)), Eq(y, A(2))),
		Conj(Eq(x, A(3)), Eq(y, A(4))),
	)
	pairs := PairsInts(goal, x, y)
	// Print count and sum of all elements for stable output
	sum := 0
	for _, p := range pairs {
		sum += p[0] + p[1]
	}
	fmt.Printf("%d %d\n", len(pairs), sum)
	// Output:
	// 2 10
}

```


