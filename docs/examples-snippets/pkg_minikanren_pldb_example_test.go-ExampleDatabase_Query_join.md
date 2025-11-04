```go
func ExampleDatabase_Query_join() {
	parent, _ := DbRel("parent", 2, 0, 1)

	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("charlie"), NewAtom("diana"))

	// Query: Find grandparent-grandchild pairs
	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	gp := Fresh("grandparent")
	gc := Fresh("grandchild")
	p := Fresh("parent")

	goal := Conj(
		db.Query(parent, gp, p),
		db.Query(parent, p, gc),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d grandparent relationships\n", len(results))

	// Output:
	// Found 2 grandparent relationships
}

```


