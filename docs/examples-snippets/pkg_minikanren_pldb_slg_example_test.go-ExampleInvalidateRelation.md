```go
func ExampleInvalidateRelation() {
	// Start with a clean cache to make the example deterministic
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache with edge_rel predicate
	goal := TabledQuery(db, edge, "edge_rel", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Invalidate only the edge_rel predicate
	InvalidateRelation("edge_rel")

	engine := GlobalEngine()
	stats := engine.Stats()

	// With fine-grained invalidation, only edge_rel is cleared
	fmt.Printf("Cache cleared: %v\n", stats.CachedSubgoals == 0)

	// Output:
	// Cache cleared: true
}

```


