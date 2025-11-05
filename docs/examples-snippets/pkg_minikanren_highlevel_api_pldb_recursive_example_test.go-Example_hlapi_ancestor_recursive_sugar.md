```go
func Example_hlapi_ancestor_recursive_sugar() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// ancestor2(X,Y) :- parent(X,Y).
	// ancestor2(X,Y) :- parent(X,Z), ancestor2(Z,Y).
	ancestor2 := RecursiveTablePred(db, parent, "ancestor2",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Query(parent, x, z),
				self(z, y),
			)
		})

	x := Fresh("x")

	// Mix Terms and native values at call sites
	goal := ancestor2(x, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	fmt.Println(len(rows))
	// Output:
	// 2
}

```


