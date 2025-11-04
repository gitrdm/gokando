```go
func ExampleSimpleTermo() {
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewAtom(42)),
			Eq(q, NewAtom("atom-is-simple")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewPair(NewAtom("a"), NewAtom("b"))),
			Eq(q, NewAtom("pair-is-simple")),
		)
	})

	fmt.Printf("Atom is simple: %s\n", result1[0])
	fmt.Printf("Pair is not simple: %d results\n", len(result2))
	// Output:
	// Atom is simple: atom-is-simple
	// Pair is not simple: 0 results
}

```


