```go
func ExampleDatabase_Query_join() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

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


