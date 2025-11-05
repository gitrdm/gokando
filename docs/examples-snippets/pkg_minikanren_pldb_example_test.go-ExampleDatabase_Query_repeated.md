```go
func ExampleDatabase_Query_repeated() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "c"}, // self-loop
		[]interface{}{"d", "d"}, // self-loop
	)

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


