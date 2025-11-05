```go
func ExampleMatche() {
	// Classify a list by structure
	list := List(NewAtom(1), NewAtom(2))

	result := Run(5, func(q *Var) Goal {
		return Matche(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	// Matches "multiple" clause only
	for _, r := range result {
		if atom, ok := r.(*Atom); ok {
			fmt.Println(atom.value)
		}
	}

	// Output:
	// multiple
}

```


