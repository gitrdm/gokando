```go
func ExampleDatabase_Query_disjunction() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	// Query: Find children of alice OR bob
	child := Fresh("child")
	goal := Disj(
		db.Query(parent, NewAtom("alice"), child),
		db.Query(parent, NewAtom("bob"), child),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order due to parallel evaluation
	fmt.Printf("Found %d children\n", len(results))

	// Output:
	// Found 2 children
}

```


