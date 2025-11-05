```go
func Example_fdFilteredQuery_withArithmetic() {
	ctx := context.Background()

	// Setup salary database
	salary, _ := DbRel("salary", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(salary, NewAtom("alice"), NewAtom(50000))
	db, _ = db.AddFact(salary, NewAtom("bob"), NewAtom(80000))

	// FD constraints: salary must be in range, bonus = salary / 10
	model := NewModel()
	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000, 60000, 70000}))
	bonusVar := model.NewVariable(NewBitSetDomainFromValues(10000, []int{5000, 6000, 7000}))

	// Add arithmetic constraint: bonus * 10 = salary (scaled by 10 to avoid division)
	ls, _ := NewLinearSum([]*FDVariable{bonusVar}, []int{10}, salaryVar)
	model.AddConstraint(ls)

	// Propagate arithmetic constraints to get pruned domains
	solver := NewSolver(model)
	// Call Solve once to trigger propagation, then read domains from base state
	solver.Solve(ctx, 1)

	// Initialize store with propagated domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(salaryVar.ID(), solver.GetDomain(nil, salaryVar.ID()))
	store, _ = store.SetDomain(bonusVar.ID(), solver.GetDomain(nil, bonusVar.ID()))
	adapter := NewUnifiedStoreAdapter(store)

	// Query with FD filtering
	name := Fresh("name")
	sal := Fresh("salary")
	goal := FDFilteredQuery(db, salary, salaryVar, sal, name, sal)

	results, _ := goal(ctx, adapter).Take(10)

	fmt.Printf("Employees with valid salary/bonus combinations: %d\n", len(results))
	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		salBinding := result.GetBinding(sal.ID())

		if nameAtom, ok := nameBinding.(*Atom); ok {
			if salAtom, ok := salBinding.(*Atom); ok {
				if salInt, ok := salAtom.value.(int); ok {
					bonus := salInt / 10
					fmt.Printf("  %s: salary %d, bonus %d\n", nameAtom.value, salInt, bonus)
				}
			}
		}
	}

	// Output:
	// Employees with valid salary/bonus combinations: 1
	//   alice: salary 50000, bonus 5000
}

```


