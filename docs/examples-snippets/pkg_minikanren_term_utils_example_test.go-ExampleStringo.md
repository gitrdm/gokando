```go
func ExampleStringo() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Stringo(q),
			Membero(q, List(
				NewAtom("hello"),
				NewAtom(42),
				NewAtom("world"),
				NewAtom(true),
			)),
		)
	})

	fmt.Printf("String values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// String values: 2 results
	//   hello
	//   world
}

```


