package minikanren

import (
	"context"
	"fmt"
	"sort"
)

// Example_fdFilteredQuery demonstrates how to use FDFilteredQuery to combine
// database queries with finite-domain constraints. This is the recommended
// pattern for hybrid relational-FD integration.
//
// The example shows a hiring scenario where we need to find employees within
// a specific age range using FD constraints to filter database results.
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

// Example_fdFilteredQuery_multipleConstraints demonstrates combining multiple
// FD-filtered queries to enforce constraints across multiple database relations.
//
// This pattern is useful for complex queries that need to coordinate constraints
// from multiple data sources.
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

// Example_mapQueryResult demonstrates using MapQueryResult to extract database
// values and bind them to FD variables for constraint propagation.
//
// This pattern is useful when you need to query a fact from the database and
// then use that value in FD constraint solving.
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

// Example_fdFilteredQuery_withArithmetic demonstrates combining FDFilteredQuery
// with arithmetic constraints to solve complex hybrid problems.
//
// This example shows a bonus calculation where bonuses are constrained by both
// database facts and arithmetic relationships.
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

// Example_fdFilteredQuery_compositional demonstrates the compositional design
// of FDFilteredQuery - it's a Goal that composes with other Goals naturally.
//
// This shows how hybrid queries integrate seamlessly with the existing miniKanren
// operators like Conj, Disj, and custom goal constructors.
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
