```go
func ExampleCompoundTermo() {
	// Pairs are compound
	pair := NewPair(NewAtom("a"), NewAtom("b"))

	// Atoms are not compound
	atom := NewAtom(42)

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(pair),
			Eq(q, NewAtom("pair-is-compound")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(atom),
			Eq(q, NewAtom("atom-is-compound")),
		)
	})

	fmt.Printf("Pair is compound: %d results\n", len(result1))
	fmt.Printf("Atom is compound: %d results\n", len(result2))
	// Output:
	// Pair is compound: 1 results
	// Atom is compound: 0 results
}

```


