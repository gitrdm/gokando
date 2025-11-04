package minikanren

import (
	"context"
	"testing"
)

// TestFDFilteredQuery_BasicFiltering verifies that FDFilteredQuery correctly
// filters database results based on FD domain membership. This is the core
// functionality that enables hybrid integration.
func TestFDFilteredQuery_BasicFiltering(t *testing.T) {
	ctx := context.Background()

	// Setup database with employee ages
	employee, err := DbRel("employee", 2, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, err = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}
	db, err = db.AddFact(employee, NewAtom("bob"), NewAtom(42))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}
	db, err = db.AddFact(employee, NewAtom("charlie"), NewAtom(31))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}
	db, err = db.AddFact(employee, NewAtom("diana"), NewAtom(19))
	if err != nil {
		t.Fatalf("Failed to add fact: %v", err)
	}

	// Setup FD model with age constraint [25, 35]
	model := NewModel()
	ageValues := make([]int, 0, 11)
	for i := 25; i <= 35; i++ {
		ageValues = append(ageValues, i)
	}
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, ageValues))

	// Setup hybrid store
	store := NewUnifiedStore()
	store, err = store.SetDomain(ageVar.ID(), ageVar.Domain())
	if err != nil {
		t.Fatalf("Failed to set domain: %v", err)
	}
	adapter := NewUnifiedStoreAdapter(store)

	// Execute FD-filtered query
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)
	results, hasMore := goal(ctx, adapter).Take(10)

	// Verify results
	if hasMore {
		t.Error("Expected all results in one batch")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify alice (28) and charlie (31) are in results
	foundAlice := false
	foundCharlie := false

	for _, result := range results {
		nameBinding := result.GetBinding(name.ID())
		ageBinding := result.GetBinding(age.ID())

		if nameAtom, ok := nameBinding.(*Atom); ok {
			nameStr, ok := nameAtom.value.(string)
			if !ok {
				t.Error("Name binding is not a string")
				continue
			}

			if ageAtom, ok := ageBinding.(*Atom); ok {
				ageInt, ok := ageAtom.value.(int)
				if !ok {
					t.Error("Age binding is not an integer")
					continue
				}

				// Verify age is in domain
				if ageInt < 25 || ageInt > 35 {
					t.Errorf("Result age %d not in domain [25, 35]", ageInt)
				}

				switch nameStr {
				case "alice":
					if ageInt != 28 {
						t.Errorf("Alice age expected 28, got %d", ageInt)
					}
					foundAlice = true
				case "charlie":
					if ageInt != 31 {
						t.Errorf("Charlie age expected 31, got %d", ageInt)
					}
					foundCharlie = true
				case "bob", "diana":
					t.Errorf("Unexpected result: %s should be filtered out", nameStr)
				default:
					t.Errorf("Unknown name: %s", nameStr)
				}
			}
		}
	}

	if !foundAlice {
		t.Error("Alice not found in results")
	}
	if !foundCharlie {
		t.Error("Charlie not found in results")
	}
}

// TestFDFilteredQuery_EmptyDomain verifies that an empty FD domain filters
// out all results. This tests the edge case of unsatisfiable constraints.
func TestFDFilteredQuery_EmptyDomain(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, err := DbRel("employee", 2, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))

	// Setup FD model with empty domain (impossible constraint)
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{}))

	// Setup hybrid store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Execute FD-filtered query
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name)
	results, _ := goal(ctx, adapter).Take(10)

	// Verify no results
	if len(results) != 0 {
		t.Errorf("Expected 0 results with empty domain, got %d", len(results))
	}
}

// TestFDFilteredQuery_NoDomainPassthrough verifies that when no FD domain
// is set, all results pass through unfiltered. This ensures backward
// compatibility with pure relational queries.
func TestFDFilteredQuery_NoDomainPassthrough(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, err := DbRel("employee", 2, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))

	// Create FD variable but DON'T set domain in store
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 35}))

	// Setup store WITHOUT setting domain
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Execute query
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)
	results, _ := goal(ctx, adapter).Take(10)

	// Verify all results pass through
	if len(results) != 2 {
		t.Errorf("Expected 2 results without domain, got %d", len(results))
	}
}

// TestFDFilteredQuery_NonIntegerBindings verifies that non-integer bindings
// pass through without filtering. This ensures the helper works with mixed
// data types in databases.
func TestFDFilteredQuery_NonIntegerBindings(t *testing.T) {
	ctx := context.Background()

	// Setup database with string "ages" (not integers)
	employee, err := DbRel("employee", 2, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom("twenty-eight"))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom("forty-two"))

	// Setup FD model
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25, 35}))

	// Setup hybrid store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Execute query
	name := Fresh("name")
	age := Fresh("age")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)
	results, _ := goal(ctx, adapter).Take(10)

	// Non-integer bindings should pass through
	if len(results) != 2 {
		t.Errorf("Expected 2 results with non-integer bindings, got %d", len(results))
	}
}

// TestFDFilteredQuery_MultipleVariables verifies that FDFilteredQuery works
// correctly when the query has multiple variables, filtering only the
// specified one.
func TestFDFilteredQuery_MultipleVariables(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, err := DbRel("employee", 3, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28), NewAtom("engineer"))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42), NewAtom("manager"))
	db, _ = db.AddFact(employee, NewAtom("charlie"), NewAtom(31), NewAtom("engineer"))

	// Setup FD model constraining age
	model := NewModel()
	ageValues := []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, ageValues))

	// Setup hybrid store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Query with multiple variables
	name := Fresh("name")
	age := Fresh("age")
	role := Fresh("role")
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age, role)
	results, _ := goal(ctx, adapter).Take(10)

	// Verify filtering on age only
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify all results have age in domain
	for _, result := range results {
		ageBinding := result.GetBinding(age.ID())
		if ageAtom, ok := ageBinding.(*Atom); ok {
			if ageInt, ok := ageAtom.value.(int); ok {
				if ageInt < 25 || ageInt > 35 {
					t.Errorf("Age %d not in domain", ageInt)
				}
			}
		}
	}
}

// TestMapQueryResult_BasicMapping verifies that MapQueryResult correctly
// transfers a binding from a query result to an FD variable.
func TestMapQueryResult_BasicMapping(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, err := DbRel("employee", 2, 0)
	if err != nil {
		t.Fatalf("Failed to create relation: %v", err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))

	// Query for alice's age
	age := Fresh("age")
	goal := db.Query(employee, NewAtom("alice"), age)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(1)

	if len(results) == 0 {
		t.Fatal("No results from query")
	}

	// Create FD variable
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}))

	// Map query result to FD variable
	store, err = MapQueryResult(results[0], age, ageVar, store)
	if err != nil {
		t.Fatalf("MapQueryResult failed: %v", err)
	}

	// Verify binding was transferred
	binding := store.GetBinding(int64(ageVar.ID()))
	if binding == nil {
		t.Fatal("Binding not transferred to FD variable")
	}

	if ageAtom, ok := binding.(*Atom); ok {
		if ageInt, ok := ageAtom.value.(int); ok {
			if ageInt != 28 {
				t.Errorf("Expected age 28, got %d", ageInt)
			}
		} else {
			t.Error("Binding is not an integer")
		}
	} else {
		t.Error("Binding is not an atom")
	}
}

// TestMapQueryResult_NoBinding verifies that MapQueryResult returns unchanged
// store when the variable has no binding in the result.
func TestMapQueryResult_NoBinding(t *testing.T) {
	// Create empty result
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Create unbound variable
	age := Fresh("age")

	// Create FD variable
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{20, 30}))

	// Attempt mapping with no binding
	newStore, err := MapQueryResult(adapter, age, ageVar, store)
	if err != nil {
		t.Fatalf("MapQueryResult failed: %v", err)
	}

	// Verify store unchanged
	if newStore != store {
		t.Error("Store should be unchanged when variable not bound")
	}

	// Verify no binding for FD variable
	binding := newStore.GetBinding(int64(ageVar.ID()))
	if binding != nil {
		t.Error("FD variable should not have binding")
	}
}

// TestMapQueryResult_NilInputs verifies that MapQueryResult handles nil
// inputs gracefully without panicking.
func TestMapQueryResult_NilInputs(t *testing.T) {
	store := NewUnifiedStore()
	age := Fresh("age")
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{20}))

	// Test nil result
	newStore, err := MapQueryResult(nil, age, ageVar, store)
	if err != nil {
		t.Errorf("MapQueryResult with nil result failed: %v", err)
	}
	if newStore != store {
		t.Error("Store should be unchanged with nil result")
	}

	// Test nil variable
	newStore, err = MapQueryResult(NewUnifiedStoreAdapter(store), nil, ageVar, store)
	if err != nil {
		t.Errorf("MapQueryResult with nil var failed: %v", err)
	}
	if newStore != store {
		t.Error("Store should be unchanged with nil var")
	}

	// Test nil FD variable
	newStore, err = MapQueryResult(NewUnifiedStoreAdapter(store), age, nil, store)
	if err != nil {
		t.Errorf("MapQueryResult with nil FD var failed: %v", err)
	}
	if newStore != store {
		t.Error("Store should be unchanged with nil FD var")
	}

	// Test nil store
	newStore, err = MapQueryResult(NewUnifiedStoreAdapter(store), age, ageVar, nil)
	if err != nil {
		t.Errorf("MapQueryResult with nil store failed: %v", err)
	}
	if newStore != nil {
		t.Error("Store should be nil when input is nil")
	}
}

// TestHybridConj_CombinesConstraints verifies that HybridConj correctly
// combines multiple FD-filtered queries with conjunction semantics.
func TestHybridConj_CombinesConstraints(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))

	salary, _ := DbRel("salary", 2, 0)
	db, _ = db.AddFact(salary, NewAtom("alice"), NewAtom(50000))
	db, _ = db.AddFact(salary, NewAtom("bob"), NewAtom(60000))

	// Setup FD constraints
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{28}))
	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000}))

	// Setup store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	store, _ = store.SetDomain(salaryVar.ID(), salaryVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Create hybrid queries
	name := Fresh("name")
	age := Fresh("age")
	sal := Fresh("salary")

	ageGoal := FDFilteredQuery(db, employee, ageVar, age, name, age)
	salaryGoal := FDFilteredQuery(db, salary, salaryVar, sal, name, sal)

	// Combine with conjunction
	combined := HybridConj(ageGoal, salaryGoal)
	results, _ := combined(ctx, adapter).Take(10)

	// Both constraints must be satisfied - only alice
	if len(results) != 1 {
		t.Errorf("Expected 1 result from conjunction, got %d", len(results))
	}

	if len(results) > 0 {
		nameBinding := results[0].GetBinding(name.ID())
		if nameAtom, ok := nameBinding.(*Atom); ok {
			if nameStr, ok := nameAtom.value.(string); ok {
				if nameStr != "alice" {
					t.Errorf("Expected alice, got %s", nameStr)
				}
			}
		}
	}
}

// TestHybridDisj_AcceptsEither verifies that HybridDisj correctly combines
// multiple FD-filtered queries with disjunction semantics.
func TestHybridDisj_AcceptsEither(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))
	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))
	db, _ = db.AddFact(employee, NewAtom("charlie"), NewAtom(35))

	// Setup two non-overlapping age constraints
	model := NewModel()
	youngVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{28}))
	seniorVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{42}))

	// Setup store with both domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(youngVar.ID(), youngVar.Domain())
	store, _ = store.SetDomain(seniorVar.ID(), seniorVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Create two queries
	name1 := Fresh("name1")
	age1 := Fresh("age1")
	name2 := Fresh("name2")
	age2 := Fresh("age2")

	youngGoal := FDFilteredQuery(db, employee, youngVar, age1, name1, age1)
	seniorGoal := FDFilteredQuery(db, employee, seniorVar, age2, name2, age2)

	// Combine with disjunction
	combined := HybridDisj(youngGoal, seniorGoal)
	results, _ := combined(ctx, adapter).Take(10)

	// Either constraint can be satisfied - alice OR bob
	if len(results) != 2 {
		t.Errorf("Expected 2 results from disjunction, got %d", len(results))
	}

	// Charlie (35) should not appear in either result set
	for _, result := range results {
		// Check name1
		if nameBinding := result.GetBinding(name1.ID()); nameBinding != nil {
			if nameAtom, ok := nameBinding.(*Atom); ok {
				if nameStr, ok := nameAtom.value.(string); ok {
					if nameStr == "charlie" {
						t.Error("Charlie should not appear in results")
					}
				}
			}
		}
		// Check name2
		if nameBinding := result.GetBinding(name2.ID()); nameBinding != nil {
			if nameAtom, ok := nameBinding.(*Atom); ok {
				if nameStr, ok := nameAtom.value.(string); ok {
					if nameStr == "charlie" {
						t.Error("Charlie should not appear in results")
					}
				}
			}
		}
	}
}

// TestFDFilteredQuery_ConcurrentExecution verifies that FDFilteredQuery is
// safe for concurrent execution, which is critical for parallel search.
func TestFDFilteredQuery_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	for i := 0; i < 100; i++ {
		db, _ = db.AddFact(employee, NewAtom(i), NewAtom(i))
	}

	// Setup FD model
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomainFromValues(100, []int{25, 50, 75}))

	// Setup store
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Execute same query concurrently from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			name := Fresh("name")
			age := Fresh("age")
			goal := FDFilteredQuery(db, employee, ageVar, age, name, age)
			results, _ := goal(ctx, adapter).Take(100)

			// Each execution should get same 3 results
			if len(results) != 3 {
				t.Errorf("Expected 3 results, got %d", len(results))
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
