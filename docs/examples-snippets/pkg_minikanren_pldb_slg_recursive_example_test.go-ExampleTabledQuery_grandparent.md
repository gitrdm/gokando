```go
func ExampleTabledQuery_grandparent() {
	// Create parent relation
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
	)

	// Query for grandparent
	gp := Fresh("gp")
	p := Fresh("p")
	gc := Fresh("gc")

	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	goal := Conj(
		TabledQuery(db, parent, "parent", gp, p),
		TabledQuery(db, parent, "parent", p, gc),
		Eq(gp, NewAtom("john")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(gc.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("john's grandchild: %s\n", atom.String())
		}
	}

	// Output:
	// john's grandchild: alice
}

```


