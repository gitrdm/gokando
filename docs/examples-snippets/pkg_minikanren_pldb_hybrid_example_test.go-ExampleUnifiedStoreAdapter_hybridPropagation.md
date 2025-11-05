```go
func ExampleUnifiedStoreAdapter_hybridPropagation() {
	// Create database of people with ages
	person, _ := DbRel("person", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))

	// Create FD model with age variable (domain 0-100)
	model := NewModel()
	ageValues := make([]int, 101)
	for i := range ageValues {
		ageValues[i] = i
	}
	// ageVar := model.NewVariableWithName(NewBitSetDomainFromValues(101, ageValues), "age")
	ageVar := model.IntVarValues(ageValues, "age")

	// Create HybridSolver and a UnifiedStore populated from the model.
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}
	adapter := NewUnifiedStoreAdapter(store)

	// Query for alice's age
	age := Fresh("age")
	goal := db.Query(person, NewAtom("alice"), age)
	stream := goal(context.Background(), adapter)

	results, _ := stream.Take(1)
	if len(results) > 0 {
		resultAdapter := results[0].(*UnifiedStoreAdapter)

		// Link logical variable to FD variable
		resultStore := resultAdapter.UnifiedStore()
		ageBinding := resultAdapter.GetBinding(age.ID())
		if ageAtom, ok := ageBinding.(*Atom); ok {
			if ageInt, ok := ageAtom.value.(int); ok {
				// Bind FD variable to the same value
				resultStore, _ = resultStore.AddBinding(int64(ageVar.ID()), NewAtom(ageInt))
				resultAdapter.SetUnifiedStore(resultStore)

				// Run propagation
				propagated, err := solver.Propagate(resultAdapter.UnifiedStore())
				if err == nil {
					// FD domain should now be singleton {30}
					ageDomain := propagated.GetDomain(ageVar.ID())
					if ageDomain.IsSingleton() {
						fmt.Printf("FD domain pruned to: {%d}\n", ageDomain.SingletonValue())
					}
				}
			}
		}
	}

	// Output:
	// FD domain pruned to: {30}
}

```


