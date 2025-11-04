```go
func ExampleTabledDatabase() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	// Wrap database for automatic tabling
	tdb := WithTabledDatabase(db, "mydb")

	x := Fresh("x")
	y := Fresh("y")

	// All queries automatically use tabling
	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges with automatic tabling\n", len(results))

	// Output:
	// Found 2 edges with automatic tabling
}

```


