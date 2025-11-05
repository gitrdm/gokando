```go
func ExampleDatabase_Query_simple() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"alice", "charlie"},
		[]interface{}{"bob", "diana"},
	)

	// Query: Who are alice's children?
	child := Fresh("child")
	goal := db.Query(parent, NewAtom("alice"), child)

	// Execute the query
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order
	fmt.Printf("Alice has %d children\n", len(results))

	// Output:
	// Alice has 2 children
}

```


