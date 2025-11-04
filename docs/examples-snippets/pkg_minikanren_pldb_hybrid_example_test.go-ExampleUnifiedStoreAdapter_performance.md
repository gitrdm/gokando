```go
func ExampleUnifiedStoreAdapter_performance() {
	// Create large database with 1000 people
	person, _ := DbRel("person", 3, 0, 1, 2) // all fields indexed
	db := NewDatabase()

	for i := 0; i < 1000; i++ {
		name := NewAtom(fmt.Sprintf("person%d", i))
		age := NewAtom(20 + (i % 50))
		score := NewAtom(50 + (i % 50))
		db, _ = db.AddFact(person, name, age, score)
	}

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for specific age (indexed lookup is O(1))
	name := Fresh("name")
	score := Fresh("score")

	goal := db.Query(person, name, NewAtom(30), score)
	stream := goal(context.Background(), adapter)

	// Fast retrieval even from large database
	results, _ := stream.Take(100)

	fmt.Printf("Found %d people with age 30 (from 1000 total)\n", len(results))

	// Output:
	// Found 20 people with age 30 (from 1000 total)
}

```


