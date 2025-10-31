// Package minikanren provides comprehensive tests for constraint store operations
// and debugging utilities introduced in Phase 10.
//
// This test file covers:
//   - Store manipulation primitives (EmptyStore, StoreWithConstraint, etc.)
//   - Store inspection utilities (StoreVariables, StoreDomains, etc.)
//   - Edge cases, error conditions, and thread safety
//
// All tests use production constraint implementations for validation.
package minikanren

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestEmptyStore(t *testing.T) {
	store := EmptyStore()
	if store == nil {
		t.Fatal("EmptyStore returned nil")
	}

	// Check that it's empty
	constraints := store.GetConstraints()
	if len(constraints) != 0 {
		t.Errorf("EmptyStore should have no constraints, got %d", len(constraints))
	}

	sub := store.GetSubstitution()
	if sub.Size() != 0 {
		t.Errorf("EmptyStore should have no bindings, got %d", sub.Size())
	}
}

func TestStoreWithConstraint(t *testing.T) {
	store := EmptyStore()
	v := Fresh("x")
	constraint := NewDisequalityConstraint(v, NewAtom("forbidden"))

	result, err := StoreWithConstraint(store, constraint)
	if err != nil {
		t.Fatalf("StoreWithConstraint failed: %v", err)
	}

	// Check that constraint was added
	constraints := result.GetConstraints()
	if len(constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(constraints))
	}

	if !strings.Contains(constraints[0].ID(), "neq") {
		t.Errorf("Expected constraint ID to contain 'neq', got '%s'", constraints[0].ID())
	}

	// Check that original store is unchanged
	originalConstraints := store.GetConstraints()
	if len(originalConstraints) != 0 {
		t.Errorf("Original store should be unchanged, but has %d constraints", len(originalConstraints))
	}
}

func TestStoreWithConstraintErrors(t *testing.T) {
	// Test nil store
	_, err := StoreWithConstraint(nil, NewDisequalityConstraint(Fresh("x"), NewAtom("test")))
	if err == nil {
		t.Error("Expected error for nil store")
	}

	// Test nil constraint
	store := EmptyStore()
	_, err = StoreWithConstraint(store, nil)
	if err == nil {
		t.Error("Expected error for nil constraint")
	}

	// Test constraint that immediately violates (create a constraint that would violate)
	store = EmptyStore()
	v := Fresh("x")
	// Bind x to "test" then create constraint that x != "test"
	err = store.AddBinding(v.id, NewAtom("test"))
	if err != nil {
		t.Fatalf("Failed to bind variable: %v", err)
	}
	constraint := NewDisequalityConstraint(v, NewAtom("test")) // This should violate

	_, err = StoreWithConstraint(store, constraint)
	if err == nil {
		t.Error("Expected error for immediately violating constraint")
	}
}

func TestStoreWithoutConstraint(t *testing.T) {
	// Create store with a constraint
	store := EmptyStore()
	constraint := NewTypeConstraint(Fresh("x"), SymbolType)

	storeWithConstraint, err := StoreWithConstraint(store, constraint)
	if err != nil {
		t.Fatalf("Failed to add constraint: %v", err)
	}

	// Remove the constraint
	result, err := StoreWithoutConstraint(storeWithConstraint, constraint)
	if err != nil {
		t.Fatalf("StoreWithoutConstraint failed: %v", err)
	}

	// Check that constraint was removed
	constraints := result.GetConstraints()
	if len(constraints) != 0 {
		t.Errorf("Expected 0 constraints after removal, got %d", len(constraints))
	}
}

func TestStoreWithoutConstraintErrors(t *testing.T) {
	store := EmptyStore()

	// Test nil store
	_, err := StoreWithoutConstraint(nil, NewDisequalityConstraint(Fresh("x"), NewAtom("test")))
	if err == nil {
		t.Error("Expected error for nil store")
	}

	// Test nil constraint
	_, err = StoreWithoutConstraint(store, nil)
	if err == nil {
		t.Error("Expected error for nil constraint")
	}

	// Test removing non-existent constraint (should succeed)
	constraint := NewAbsenceConstraint(NewAtom("missing"), Fresh("x"))
	result, err := StoreWithoutConstraint(store, constraint)
	if err != nil {
		t.Errorf("Removing non-existent constraint should succeed, got error: %v", err)
	}

	constraints := result.GetConstraints()
	if len(constraints) != 0 {
		t.Errorf("Store should still be empty after removing non-existent constraint")
	}
}

func TestStoreUnion(t *testing.T) {
	// Create two stores with different constraints
	store1 := EmptyStore()
	store2 := EmptyStore()

	v1 := Fresh("x")
	v2 := Fresh("y")

	constraint1 := NewDisequalityConstraint(v1, NewAtom("forbidden1"))
	constraint2 := NewTypeConstraint(v2, NumberType)

	store1, _ = StoreWithConstraint(store1, constraint1)
	store2, _ = StoreWithConstraint(store2, constraint2)

	// Union the stores
	result, err := StoreUnion(store1, store2)
	if err != nil {
		t.Fatalf("StoreUnion failed: %v", err)
	}

	// Check that both constraints are present
	constraints := result.GetConstraints()
	if len(constraints) != 2 {
		t.Errorf("Expected 2 constraints in union, got %d", len(constraints))
	}

	// Check that both constraints are present (order may vary)
	constraintIDs := make(map[string]bool)
	for _, c := range constraints {
		constraintIDs[c.ID()] = true
	}

	foundNeq := false
	foundType := false
	for _, c := range constraints {
		if strings.Contains(c.ID(), "neq") {
			foundNeq = true
		}
		if strings.Contains(c.ID(), "type") || strings.Contains(c.ID(), "numbero") {
			foundType = true
		}
	}

	if !foundNeq || !foundType {
		t.Errorf("Union missing expected constraint types: found neq=%v, type=%v", foundNeq, foundType)
	}
}

func TestStoreUnionErrors(t *testing.T) {
	// Test nil stores
	_, err := StoreUnion(nil, EmptyStore())
	if err == nil {
		t.Error("Expected error for nil first store")
	}

	_, err = StoreUnion(EmptyStore(), nil)
	if err == nil {
		t.Error("Expected error for nil second store")
	}
}

func TestStoreIntersection(t *testing.T) {
	// Create stores with some overlapping constraints
	store1 := EmptyStore()
	store2 := EmptyStore()

	constraint1 := NewAbsenceConstraint(NewAtom("bad"), Fresh("shared"))
	constraint2 := NewTypeConstraint(Fresh("unique1"), SymbolType)
	constraint3 := NewDisequalityConstraint(Fresh("unique2"), NewAtom("forbidden"))

	store1, _ = StoreWithConstraint(store1, constraint1)
	store1, _ = StoreWithConstraint(store1, constraint2)

	store2, _ = StoreWithConstraint(store2, constraint1)
	store2, _ = StoreWithConstraint(store2, constraint3)

	// Intersect the stores
	result, err := StoreIntersection(store1, store2)
	if err != nil {
		t.Fatalf("StoreIntersection failed: %v", err)
	}

	// Check that only the shared constraint is present
	constraints := result.GetConstraints()
	if len(constraints) != 1 {
		t.Errorf("Expected 1 constraint in intersection, got %d", len(constraints))
	}

	if !strings.Contains(constraints[0].ID(), "absento") {
		t.Errorf("Expected absence constraint in intersection, got '%s'", constraints[0].ID())
	}
}

func TestStoreDifference(t *testing.T) {
	// Create stores for difference operation
	store1 := EmptyStore()
	store2 := EmptyStore()

	constraint1 := NewAbsenceConstraint(NewAtom("shared"), Fresh("x"))
	constraint2 := NewTypeConstraint(Fresh("unique"), NumberType)

	store1, _ = StoreWithConstraint(store1, constraint1)
	store1, _ = StoreWithConstraint(store1, constraint2)

	store2, _ = StoreWithConstraint(store2, constraint1)

	// Difference: store1 - store2
	result, err := StoreDifference(store1, store2)
	if err != nil {
		t.Fatalf("StoreDifference failed: %v", err)
	}

	// Check that only the unique constraint remains
	constraints := result.GetConstraints()
	if len(constraints) != 1 {
		t.Errorf("Expected 1 constraint in difference, got %d", len(constraints))
	}

	if !strings.Contains(constraints[0].ID(), "numbero") {
		t.Errorf("Expected type constraint in difference, got '%s'", constraints[0].ID())
	}
}

func TestStoreVariables(t *testing.T) {
	store := EmptyStore()

	// Empty store should have no variables
	vars := StoreVariables(store)
	if len(vars) != 0 {
		t.Errorf("Empty store should have no variables, got %d", len(vars))
	}

	// Add constraint with variables
	v1 := Fresh("x")
	v2 := Fresh("y")
	constraint := NewDisequalityConstraint(v1, v2)

	store, _ = StoreWithConstraint(store, constraint)
	vars = StoreVariables(store)

	if len(vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(vars))
	}

	// Check that variables are returned in sorted order
	if vars[0].id >= vars[1].id {
		t.Error("Variables should be sorted by ID")
	}
}

func TestStoreDomains(t *testing.T) {
	store := EmptyStore()

	// Add FD domain constraint
	v := Fresh("x")
	domain := NewBitSetFromInterval(1, 10) // Domain 1-10
	constraint := NewFDDomainConstraint(v, domain)

	store, _ = StoreWithConstraint(store, constraint)

	domains := StoreDomains(store)

	// Should have domain info for the variable
	if len(domains) != 1 {
		t.Errorf("Expected 1 domain entry, got %d", len(domains))
	}

	domainStr, exists := domains[v.id]
	if !exists {
		t.Errorf("Expected domain for variable %d", v.id)
	}

	if domainStr == "" {
		t.Errorf("Expected non-empty domain string, got empty string")
	}
}

func TestStoreValidate(t *testing.T) {
	store := EmptyStore()

	// Valid empty store
	errors := StoreValidate(store)
	if len(errors) != 0 {
		t.Errorf("Empty store should be valid, got errors: %v", errors)
	}

	// Test nil store
	errors = StoreValidate(nil)
	if len(errors) != 1 {
		t.Errorf("Nil store should return 1 error, got %d", len(errors))
	}

	if !strings.Contains(errors[0].Error(), "store is nil") {
		t.Errorf("Expected 'store is nil' error, got '%s'", errors[0].Error())
	}

	// Add a satisfied constraint
	constraint := NewTypeConstraint(Fresh("x"), SymbolType)

	store, _ = StoreWithConstraint(store, constraint)
	errors = StoreValidate(store)
	if len(errors) != 0 {
		t.Errorf("Store with satisfied constraint should be valid, got errors: %v", errors)
	}
}

func TestStoreToString(t *testing.T) {
	store := EmptyStore()

	str := StoreToString(store)
	if str == "" {
		t.Error("StoreToString should not return empty string")
	}

	if !strings.Contains(str, "Constraint Store") {
		t.Errorf("StoreToString should contain 'Constraint Store', got: %s", str)
	}

	// Add constraint and check output
	v := Fresh("x")
	constraint := NewDisequalityConstraint(v, NewAtom("test"))

	store, _ = StoreWithConstraint(store, constraint)
	str = StoreToString(store)

	if !strings.Contains(str, "≠") && !strings.Contains(str, "neq") {
		t.Errorf("StoreToString should contain constraint info, got: %s", str)
	}
}

func TestStoreSummary(t *testing.T) {
	store := EmptyStore()

	summary := StoreSummary(store)
	if summary == "" {
		t.Error("StoreSummary should not return empty string")
	}

	if !strings.Contains(summary, "constraints") {
		t.Errorf("StoreSummary should contain 'constraints', got: %s", summary)
	}
}

// Test thread safety of store operations
func TestStoreOperationsThreadSafety(t *testing.T) {
	store := EmptyStore()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Run multiple goroutines performing store operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Create unique constraint for this operation
				constraint := NewAbsenceConstraint(NewAtom(fmt.Sprintf("bad-%d-%d", id, j)), Fresh(fmt.Sprintf("var-%d-%d", id, j)))

				// Perform various operations
				_, err := StoreWithConstraint(store, constraint)
				if err != nil {
					t.Errorf("StoreWithConstraint failed in goroutine %d: %v", id, err)
				}

				StoreVariables(store)
				StoreDomains(store)
				StoreValidate(store)
				StoreToString(store)
				StoreSummary(store)
			}
		}(i)
	}

	wg.Wait()
}

// Test edge cases and error conditions
func TestStoreOperationsEdgeCases(t *testing.T) {
	// Test with nil inputs
	if StoreVariables(nil) != nil {
		t.Error("StoreVariables(nil) should return nil")
	}

	if StoreDomains(nil) != nil {
		t.Error("StoreDomains(nil) should return nil")
	}

	// StoreValidate(nil) returns an error slice, not nil
	errors := StoreValidate(nil)
	if len(errors) == 0 {
		t.Error("StoreValidate(nil) should return errors")
	}

	if StoreToString(nil) != "Store: <nil>" {
		t.Errorf("StoreToString(nil) should return 'Store: <nil>', got '%s'", StoreToString(nil))
	}

	if StoreSummary(nil) != "Store: <nil>" {
		t.Errorf("StoreSummary(nil) should return 'Store: <nil>', got '%s'", StoreSummary(nil))
	}
}
