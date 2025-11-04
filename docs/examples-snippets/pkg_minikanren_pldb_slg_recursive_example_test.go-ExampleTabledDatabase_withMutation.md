```go
func ExampleTabledDatabase_withMutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutable")

	x := Fresh("x")
	y := Fresh("y")

	// Query once to populate cache
	goal := tdb.Query(edge, x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results1, _ := stream.Take(10)

	fmt.Printf("Before update: %d edges\n", len(results1))

	// Add a new fact
	db2, _ := db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb2 := WithTabledDatabase(db2, "mutable")

	// Clear cache for this predicate
	InvalidateAll()

	// Query again with new database
	goal2 := tdb2.Query(edge, x, y)
	stream2 := goal2(ctx, store)
	results2, _ := stream2.Take(10)

	fmt.Printf("After update: %d edges\n", len(results2))

	// Output:
	// Before update: 1 edges
	// After update: 2 edges
}

```


