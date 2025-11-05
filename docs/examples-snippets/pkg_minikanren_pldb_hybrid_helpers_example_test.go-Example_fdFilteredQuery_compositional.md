```go
func Example_fdFilteredQuery_compositional() {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))
	db, _ = db.AddFact(employee, NewAtom("charlie"), NewAtom(31))

	// FD model
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}))

	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Compose FD-filtered query with additional constraints
	name := Fresh("name")
	age := Fresh("age")

	goal := Conj(
		FDFilteredQuery(db, employee, ageVar, age, name, age),
		// Add additional constraint: name must start with 'a' or 'c'
		Disj(
			Eq(name, NewAtom("alice")),
			Eq(name, NewAtom("charlie")),
		),
	)

	results, _ := goal(ctx, adapter).Take(10)

	// Sort for deterministic output
	names := make([]string, 0)
	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		if nameAtom, ok := nameBinding.(*Atom); ok {
			if nameStr, ok := nameAtom.value.(string); ok {
				names = append(names, nameStr)
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Employees aged 25-35 with names starting with a or c: %d\n", len(names))
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}

	// Output:
	// Employees aged 25-35 with names starting with a or c: 2
	//   alice
	//   charlie
}

```


