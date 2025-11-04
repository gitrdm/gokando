```go
func ExampleWithTabledDatabase_mutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutdb")

	// Add more facts - cache invalidates automatically
	tdb, _ = tdb.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb, _ = tdb.AddFact(edge, NewAtom("c"), NewAtom("d"))

	x := Fresh("x")
	y := Fresh("y")

	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges after additions\n", len(results))

	// Output:
	// Found 3 edges after additions
}

```


