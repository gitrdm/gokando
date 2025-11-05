```go
func ExampleTabledRelation() {
	// Clear cache for clean test
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

	// Create tabled predicate constructor
	edgePred := TabledRelation(db, edge, "edge_example")

	x := Fresh("x")
	y := Fresh("y")

	// Use it like a normal predicate
	goal := edgePred(x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 3 edges
}

```


