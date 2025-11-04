```go
func ExampleUnifiedStoreAdapter_fdConstrainedQuery() {
	// Create a database of employees with ages
	employee, _ := DbRel("employee", 2, 0) // name is indexed
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(32))
	db, _ = db.AddFact(employee, NewAtom("carol"), NewAtom(45))
	db, _ = db.AddFact(employee, NewAtom("dave"), NewAtom(29))

	// Create FD model with age restricted to [25, 35]
	model := NewModel()
	ageVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(100, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}),
		"age",
	)

	// Create store with FD domain
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Define variables
	name := Fresh("name")
	age := Fresh("age")

	// Create hybrid query with manual FD filtering
	hybridQuery := func(ctx context.Context, cstore ConstraintStore) *Stream {
		dbQuery := db.Query(employee, name, age)
		dbStream := dbQuery(ctx, cstore)

		stream := NewStream()
		go func() {
			defer stream.Close()
			for {
				results, hasMore := dbStream.Take(1)
				if len(results) == 0 {
					if !hasMore {
						break
					}
					continue
				}

				result := results[0]
				ageBinding := result.GetBinding(age.ID())

				// Filter: only include ages in FD domain
				if ageAtom, ok := ageBinding.(*Atom); ok {
					if ageInt, ok := ageAtom.value.(int); ok {
						if resAdapter, ok := result.(*UnifiedStoreAdapter); ok {
							domain := resAdapter.GetDomain(ageVar.ID())
							if domain != nil && domain.Has(ageInt) {
								stream.Put(result)
							}
						}
					}
				}
			}
		}()
		return stream
	}

	// Execute query
	stream := hybridQuery(context.Background(), adapter)
	results, _ := stream.Take(10)

	// Print count (order may vary)
	fmt.Printf("Found %d employees aged 25-35\n", len(results))

	// Output:
	// Found 3 employees aged 25-35
}

```


