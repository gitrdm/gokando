```go
func ExampleMatcha_deterministicChoice() {
	// Process different data types deterministically
	process := func(data Term) string {
		result := Run(1, func(q *Var) Goal {
			return Matcha(data,
				// Check for Nil first
				NewClause(Nil, Eq(q, NewAtom("empty-list"))),
				// Then check for pair
				NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
				// Default case
				NewClause(Fresh("_"), Eq(q, NewAtom("atom"))),
			)
		})

		if len(result) == 0 {
			return "error"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(process(Nil))
	fmt.Println(process(NewPair(NewAtom(1), NewAtom(2))))
	fmt.Println(process(NewAtom(42)))

	// Output:
	// empty-list
	// pair
	// atom
}

```


