```go
func ExamplePatternClause_nestedPatterns() {
	// Match nested structure: ((a b) (c d))
	data := List(
		List(NewAtom("x"), NewAtom("y")),
		List(NewAtom("z"), NewAtom("w")),
	)

	result := Run(1, func(q *Var) Goal {
		a := Fresh("a")
		b := Fresh("b")

		return Matche(data,
			NewClause(
				NewPair(
					NewPair(a, NewPair(b, Nil)),
					Fresh("_"),
				),
				Eq(q, List(a, b)),
			),
		)
	})

	if len(result) > 0 {
		fmt.Printf("Extracted first pair: %v\n", result[0])
	}

	// Output:
	// Extracted first pair: (x . (y . <nil>))
}

```


