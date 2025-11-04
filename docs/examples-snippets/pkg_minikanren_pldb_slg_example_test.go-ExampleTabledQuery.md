```go
func ExampleTabledQuery() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	x := Fresh("x")
	y := Fresh("y")

	// Tabled query caches results
	goal := TabledQuery(db, edge, "edge", x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 2 edges
}

```


