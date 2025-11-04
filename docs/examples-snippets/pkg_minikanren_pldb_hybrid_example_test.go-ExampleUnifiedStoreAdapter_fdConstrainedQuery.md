```go
func ExampleUnifiedStoreAdapter_fdConstrainedQuery() {
	// Create a database of employees with ages (compact via HLAPI)
	employee, _ := DbRel("employee", 2, 0) // name is indexed
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 32},
		[]interface{}{"carol", 45},
		[]interface{}{"dave", 29},
	)

	// Create FD model with age restricted to [25, 35]
	model := NewModel()
	// ageVar := model.NewVariableWithName(
	//     NewBitSetDomainFromValues(100, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}),
	//     "age",
	// )
	ageVar := model.IntVarValues([]int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}, "age")

	// Create store with FD domain and adapter
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Define variables
	name := Fresh("name")
	age := Fresh("age")

	// Use HLAPI FDFilteredQuery to combine the DB query and FD-domain filtering
	// FDFilteredQuery(db, rel, fdVar, filterVar, queryTerms...)
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)

	// Execute query
	stream := goal(context.Background(), adapter)
	results, _ := stream.Take(10)

	// Print count (order may vary)
	fmt.Printf("Found %d employees aged 25-35\n", len(results))

	// Output:
	// Found 3 employees aged 25-35
}

```


