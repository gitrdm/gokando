```go
func Example_mapQueryResult() {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))

	// Query alice's age
	age := Fresh("age")
	goal := db.Query(employee, NewAtom("alice"), age)

	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(1)

	// Create FD variable to receive the age
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}))

	// Map the query result to the FD variable (convenience helper)
	store, _ = MapQueryResult(results[0], age, ageVar, store)

	// Now ageVar is bound to alice's age
	binding := store.GetBinding(int64(ageVar.ID()))
	if ageAtom, ok := binding.(*Atom); ok {
		fmt.Printf("Alice's age: %d\n", ageAtom.value)
	}

	// Output:
	// Alice's age: 28
}

```


