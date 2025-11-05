```go
func ExampleUnifiedStoreAdapter_parallelSearch() {
	// Create database
	color, _ := DbRel("color", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(color, NewAtom("apple"), NewAtom("red"))
	db, _ = db.AddFact(color, NewAtom("banana"), NewAtom("yellow"))

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Simulate parallel search: clone adapter for each branch
	branch1 := adapter.Clone().(*UnifiedStoreAdapter)
	branch2 := adapter.Clone().(*UnifiedStoreAdapter)

	// Each branch queries independently
	item := Fresh("item")

	goal1 := db.Query(color, item, NewAtom("red"))
	stream1 := goal1(context.Background(), branch1)
	results1, _ := stream1.Take(1)

	goal2 := db.Query(color, item, NewAtom("yellow"))
	stream2 := goal2(context.Background(), branch2)
	results2, _ := stream2.Take(1)

	// Print results from each independent branch
	if len(results1) > 0 {
		itemBinding := results1[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 1: %s is red\n", atom.value)
		}
	}

	if len(results2) > 0 {
		itemBinding := results2[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 2: %s is yellow\n", atom.value)
		}
	}

	// Output:
	// Branch 1: apple is red
	// Branch 2: banana is yellow
}

```


