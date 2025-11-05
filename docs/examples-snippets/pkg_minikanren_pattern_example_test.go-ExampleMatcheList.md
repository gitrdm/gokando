```go
func ExampleMatcheList() {
	// Simple list pattern matching
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return MatcheList(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("head"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	fmt.Println(result[0])

	// Output:
	// multiple
}

```


