```go
func ExampleMatchu() {
	// Classify numbers with mutually exclusive ranges
	classify := func(n int) string {
		result := Run(1, func(q *Var) Goal {
			return Matchu(NewAtom(n),
				NewClause(NewAtom(0), Eq(q, NewAtom("zero"))),
				NewClause(NewAtom(1), Eq(q, NewAtom("one"))),
				NewClause(NewAtom(2), Eq(q, NewAtom("two"))),
			)
		})

		if len(result) == 0 {
			return "unknown"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(classify(0))
	fmt.Println(classify(1))
	fmt.Println(classify(5))

	// Output:
	// zero
	// one
	// unknown
}

```


