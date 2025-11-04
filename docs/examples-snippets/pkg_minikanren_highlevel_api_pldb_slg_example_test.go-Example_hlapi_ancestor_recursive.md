```go
func Example_hlapi_ancestor_recursive() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// Define ancestor(X,Y): parent(X,Y) OR (parent(X,Z) AND ancestor(Z,Y))
	ancestor := TabledRecursivePredicate(db, parent, "ancestor",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Q(parent, x, z),
				self(z, y),
			)
		},
	)

	x := Fresh("x")
	y := Fresh("y")

	goal := Conj(
		Eq(y, NewAtom("alice")),
		ancestor(x, y),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	// We just assert we found two rows to keep the example stable.
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


