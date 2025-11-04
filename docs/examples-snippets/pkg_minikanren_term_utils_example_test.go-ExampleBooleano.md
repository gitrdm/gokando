```go
func ExampleBooleano() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Membero(q, List(
				NewAtom(true),
				NewAtom("not-bool"),
				NewAtom(false),
				NewAtom(42),
			)),
		)
	})

	fmt.Printf("Boolean values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// Boolean values: 2 results
	//   true
	//   false
}

```


