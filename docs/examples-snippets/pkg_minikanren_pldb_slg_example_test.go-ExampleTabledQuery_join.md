```go
func ExampleTabledQuery_join() {
	InvalidateAll()

	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("charlie"), NewAtom("diana"))

	// TabledQuery now works correctly in joins with shared variables
	gp := Fresh("gp")
	gc := Fresh("gc")
	p := Fresh("p")

	goal := Conj(
		TabledQuery(db, parent, "parent_join_ex", gp, p),
		TabledQuery(db, parent, "parent_join_ex", p, gc),
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


