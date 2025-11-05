```go
func ExamplePluso_generate() {
	result := Run(6, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Conj(
			Pluso(x, y, NewAtom(5)),
			Eq(q, NewPair(x, y)),
		)
	})

	fmt.Printf("Generated %d pairs\n", len(result))
	// Verify all pairs sum to 5
	for _, r := range result {
		pair := r.(*Pair)
		x, _ := extractNumber(pair.Car())
		y, _ := extractNumber(pair.Cdr())
		if x+y == 5 {
			fmt.Println("Valid pair")
		}
	}
	// Output:
	// Generated 6 pairs
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
	// Valid pair
} // ExampleMinuso demonstrates basic subtraction with Minuso.

```


