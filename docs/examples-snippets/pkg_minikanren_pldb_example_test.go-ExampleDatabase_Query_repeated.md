```go
func ExampleDatabase_Query_repeated() {
	edge, _ := DbRel("edge", 2, 0, 1)

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("c")) // self-loop
	db, _ = db.AddFact(edge, NewAtom("d"), NewAtom("d")) // self-loop

	// Query: Find all self-loops
	x := Fresh("x")
	goal := db.Query(edge, x, x) // same variable in both positions

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d self-loops\n", len(results))

	// Output:
	// Found 2 self-loops
}

```


