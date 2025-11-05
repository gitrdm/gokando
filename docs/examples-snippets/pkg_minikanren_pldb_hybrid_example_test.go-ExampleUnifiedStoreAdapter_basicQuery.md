```go
func ExampleUnifiedStoreAdapter_basicQuery() {
	// Create a database of people with names and ages
	person, _ := DbRel("person", 2, 0) // name is indexed
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25))
	db, _ = db.AddFact(person, NewAtom("carol"), NewAtom(35))

	// Create UnifiedStore and adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for all people
	name := Fresh("name")
	age := Fresh("age")

	goal := db.Query(person, name, age)
	stream := goal(context.Background(), adapter)

	// Retrieve results
	results, _ := stream.Take(10)

	// Print number of results (order may vary due to map iteration)
	fmt.Printf("Found %d people\n", len(results))

	// Output:
	// Found 3 people
}

```


