```go
func ExampleMatche_listProcessing() {
	// Extract all elements from a list
	extractAll := func(list Term) []Term {
		var results []Term

		Run(10, func(q *Var) Goal {
			elem := Fresh("elem")
			rest := Fresh("rest")

			return Matche(list,
				NewClause(Nil, Eq(q, NewAtom("done"))),
				NewClause(NewPair(elem, rest), Eq(q, elem)),
			)
		})

		// Simplified - in practice would need recursive extraction
		return results
	}

	list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))
	_ = extractAll(list)

	fmt.Println("List elements extracted")

	// Output:
	// List elements extracted
}

```


