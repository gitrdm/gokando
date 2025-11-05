```go
func ExampleHybridRegistry_multipleVariables() {
	model := NewModel()

	// Setup multiple variable pairs
	age := Fresh("age")
	salary := Fresh("salary")
	bonus := Fresh("bonus")
	yearsOfService := Fresh("years")

	ageVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))
	bonusVar := model.NewVariable(NewBitSetDomain(10000))
	yearsVar := model.NewVariable(NewBitSetDomain(50))

	// Build registry incrementally
	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(salary, salaryVar)
	registry, _ = registry.MapVars(bonus, bonusVar)
	registry, _ = registry.MapVars(yearsOfService, yearsVar)

	// Query registry state
	fmt.Printf("Total mappings: %d\n", registry.MappingCount())
	fmt.Printf("Age mapped: %t\n", registry.HasMapping(age))
	fmt.Printf("Salary mapped: %t\n", registry.HasMapping(salary))
	fmt.Printf("Bonus mapped: %t\n", registry.HasMapping(bonus))
	fmt.Printf("Years mapped: %t\n", registry.HasMapping(yearsOfService))

	// Bidirectional lookups work correctly
	ageFDID := registry.GetFDVariable(age)
	salaryRelID := registry.GetRelVariable(salaryVar)
	fmt.Printf("Age has FD mapping: %t\n", ageFDID >= 0)
	fmt.Printf("Salary has relational mapping: %t\n", salaryRelID >= 0)

	// Output:
	// Total mappings: 4
	// Age mapped: true
	// Salary mapped: true
	// Bonus mapped: true
	// Years mapped: true
	// Age has FD mapping: true
	// Salary has relational mapping: true
}

```


