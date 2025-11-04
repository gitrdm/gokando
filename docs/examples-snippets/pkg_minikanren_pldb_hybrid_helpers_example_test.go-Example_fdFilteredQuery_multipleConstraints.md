```go
func Example_fdFilteredQuery_multipleConstraints() {
	ctx := context.Background()

	// Setup employee and salary databases
	employee, _ := DbRel("employee", 2, 0)
	salary, _ := DbRel("salary", 2, 0)
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 42},
		[]interface{}{"charlie", 31},
	)
	db = db.MustAddFacts(salary,
		[]interface{}{"alice", 50000},
		[]interface{}{"bob", 80000},
		[]interface{}{"charlie", 45000},
	)

	// FD constraints: age 25-35, salary 40k-60k
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}))
	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{40000, 45000, 50000, 55000, 60000}))

	// Initialize store with both domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	store, _ = store.SetDomain(salaryVar.ID(), salaryVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Create two FD-filtered queries
	name := Fresh("name")
	age := Fresh("age")
	sal := Fresh("salary")

	ageQuery := FDFilteredQuery(db, employee, ageVar, age, name, age)
	salaryQuery := FDFilteredQuery(db, salary, salaryVar, sal, name, sal)

	// Combine with conjunction - both constraints must hold
	goal := HybridConj(ageQuery, salaryQuery)

	// Execute
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

	fmt.Printf("Employees meeting both criteria: %d\n", len(names))
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}

	// Output:
	// Employees meeting both criteria: 2
	//   alice
	//   charlie
}

```


