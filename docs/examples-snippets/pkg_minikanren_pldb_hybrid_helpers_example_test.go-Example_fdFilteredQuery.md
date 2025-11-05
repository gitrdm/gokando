```go
func Example_fdFilteredQuery() {
	ctx := context.Background()

	// 1. Setup database with employee records (compact via HLAPI)
	employee, _ := DbRel("employee", 2, 0)
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 42},
		[]interface{}{"charlie", 31},
		[]interface{}{"diana", 19},
	)

	// 2. Setup FD constraint for eligible age range [25, 35]
	model := NewModel()
	eligibleAges := make([]int, 0)
	for age := 25; age <= 35; age++ {
		eligibleAges = append(eligibleAges, age)
	}
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, eligibleAges))

	// 3. Initialize hybrid store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// 4. Create FD-filtered query (ONE LINE vs 50 lines manual)
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)

	// 5. Execute and display results
	results, _ := goal(ctx, adapter).Take(10)

	// Collect and sort results for deterministic output
	type empRecord struct {
		name string
		age  int
	}
	employees := make([]empRecord, 0)

	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		ageBinding := result.GetBinding(age.ID())

		if nameAtom, ok := nameBinding.(*Atom); ok {
			if ageAtom, ok := ageBinding.(*Atom); ok {
				if nameStr, ok := nameAtom.value.(string); ok {
					if ageInt, ok := ageAtom.value.(int); ok {
						employees = append(employees, empRecord{nameStr, ageInt})
					}
				}
			}
		}
	}

	sort.Slice(employees, func(i, j int) bool {
		return employees[i].name < employees[j].name
	})

	fmt.Printf("Eligible employees (age 25-35): %d\n", len(employees))
	for _, emp := range employees {
		fmt.Printf("  %s: age %d\n", emp.name, emp.age)
	}

	// Output:
	// Eligible employees (age 25-35): 2
	//   alice: age 28
	//   charlie: age 31
}

```


