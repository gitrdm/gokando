package minikanren

import (
	"context"
	"fmt"
	"sort"
)

// ExampleNewHybridRegistry demonstrates basic registry usage for mapping variables.
func ExampleNewHybridRegistry() {
	// Create a registry for tracking relational↔FD variable mappings
	registry := NewHybridRegistry()

	// Setup variables
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	// Register the mapping
	registry, _ = registry.MapVars(age, ageVar)

	// Query the mapping
	fdID := registry.GetFDVariable(age)
	fmt.Printf("Has mapping: %t\n", fdID >= 0)
	fmt.Printf("Registry has %d mapping(s)\n", registry.MappingCount())

	// Output:
	// Has mapping: true
	// Registry has 1 mapping(s)
}

// ExampleHybridRegistry_AutoBind demonstrates automatic binding transfer,
// eliminating manual mapping boilerplate.
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

// ExampleHybridRegistry_multipleVariables demonstrates managing complex mappings
// with many variables across relational and FD spaces.
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
