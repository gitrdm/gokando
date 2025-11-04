```go
func ExampleDatabase_Query_simple() {
	parent, _ := DbRel("parent", 2, 0, 1)

	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("diana"))

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


