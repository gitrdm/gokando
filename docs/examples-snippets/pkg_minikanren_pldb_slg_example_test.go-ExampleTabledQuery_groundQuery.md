```go
func ExampleTabledQuery_groundQuery() {
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	// Fully ground query - checks existence
	goal := TabledQuery(db, edge, "edge_ground_ex", NewAtom("a"), NewAtom("b"))

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		fmt.Println("Edge a->b exists")
	} else {
		fmt.Println("Edge a->b does not exist")
	}

	// Output:
	// Edge a->b exists
}

```


