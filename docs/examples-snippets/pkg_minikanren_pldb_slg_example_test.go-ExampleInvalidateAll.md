```go
func ExampleInvalidateAll() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache
	goal := TabledQuery(db, edge, "edge_inv", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Clear all cached answers
	InvalidateAll()

	engine := GlobalEngine()
	stats := engine.Stats()

	fmt.Printf("Cached subgoals after invalidation: %d\n", stats.CachedSubgoals)

	// Output:
	// Cached subgoals after invalidation: 0
}

```


