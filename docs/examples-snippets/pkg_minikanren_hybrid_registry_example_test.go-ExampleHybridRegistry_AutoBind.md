```go
func ExampleHybridRegistry_AutoBind() {
	ctx := context.Background()
	model := NewModel()

	// Setup database
	employee, _ := DbRel("employee", 3, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28), NewAtom(50000))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(35), NewAtom(60000))

	// Setup FD variables
	ageVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))

	// Create registry mapping relational vars to FD vars
	name := Fresh("name")
	age := Fresh("age")
	salary := Fresh("salary")

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(salary, salaryVar)

	// Query database
	goal := db.Query(employee, name, age, salary)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(2)

	// AutoBind automatically transfers bindings from query results to FD store
	var employees []string
	for _, result := range results {
		// Single AutoBind call replaces manual binding transfer
		fdStore, _ := registry.AutoBind(result, store)

		nameBinding := result.GetBinding(name.ID())
		ageBinding := fdStore.GetBinding(int64(ageVar.ID()))
		salaryBinding := fdStore.GetBinding(int64(salaryVar.ID()))

		n := nameBinding.(*Atom).value.(string)
		a := ageBinding.(*Atom).value.(int)
		s := salaryBinding.(*Atom).value.(int)

		employees = append(employees, fmt.Sprintf("%s: age=%d salary=%d", n, a, s))
	}

	sort.Strings(employees)
	for _, emp := range employees {
		fmt.Println(emp)
	}

	// Output:
	// alice: age=28 salary=50000
	// bob: age=35 salary=60000
}

```


