```go
func ExampleTabledRelation_symmetricGraph() {
	friend, _ := DbRel("friend", 2, 0, 1)
	db := NewDatabase()
	// Add symmetric friendships
	db, _ = db.AddFact(friend, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(friend, NewAtom("bob"), NewAtom("alice"))

	friendPred := TabledRelation(db, friend, "friend")

	x := Fresh("x")
	// Who is friends with Alice?
	goal := Conj(
		friendPred(x, NewAtom("alice")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(x.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s is friend with alice\n", atom.String())
		}
	}

	// Output:
	// bob is friend with alice
}

```


