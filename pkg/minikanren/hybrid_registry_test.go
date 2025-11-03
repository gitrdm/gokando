package minikanren

import (
	"context"
	"testing"
)

// TestHybridRegistry_MapVars tests basic variable mapping registration.
func TestHybridRegistry_MapVars(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	newRegistry, err := registry.MapVars(age, ageVar)

	if err != nil {
		t.Fatalf("MapVars failed: %v", err)
	}

	if newRegistry.MappingCount() != 1 {
		t.Errorf("Expected 1 mapping, got %d", newRegistry.MappingCount())
	}

	// Original registry should be unchanged (immutability)
	if registry.MappingCount() != 0 {
		t.Error("Original registry was modified")
	}
}

// TestHybridRegistry_BidirectionalMapping tests that mappings work in both directions.
func TestHybridRegistry_BidirectionalMapping(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry, _ := NewHybridRegistry().MapVars(age, ageVar)

	// Test relational → FD lookup
	fdID := registry.GetFDVariable(age)
	if fdID != ageVar.ID() {
		t.Errorf("Expected FD ID %d, got %d", ageVar.ID(), fdID)
	}

	// Test FD → relational lookup
	relID := registry.GetRelVariable(ageVar)
	if relID != age.ID() {
		t.Errorf("Expected relational ID %d, got %d", age.ID(), relID)
	}
}

// TestHybridRegistry_MultipleMappings tests registering multiple variable pairs.
func TestHybridRegistry_MultipleMappings(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	name := Fresh("name")
	salary := Fresh("salary")

	ageVar := model.NewVariable(NewBitSetDomain(100))
	nameVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(name, nameVar)
	registry, _ = registry.MapVars(salary, salaryVar)

	if registry.MappingCount() != 3 {
		t.Errorf("Expected 3 mappings, got %d", registry.MappingCount())
	}

	// Verify all mappings work
	if registry.GetFDVariable(age) != ageVar.ID() {
		t.Error("Age mapping incorrect")
	}
	if registry.GetFDVariable(name) != nameVar.ID() {
		t.Error("Name mapping incorrect")
	}
	if registry.GetFDVariable(salary) != salaryVar.ID() {
		t.Error("Salary mapping incorrect")
	}
}

// TestHybridRegistry_ConflictDetection tests that conflicting mappings are rejected.
func TestHybridRegistry_ConflictDetection(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar1 := model.NewVariable(NewBitSetDomain(100))
	ageVar2 := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar1)

	// Attempt to map same relational variable to different FD variable
	_, err := registry.MapVars(age, ageVar2)
	if err == nil {
		t.Error("Expected error when mapping same relational var to different FD var")
	}
}

// TestHybridRegistry_ConflictDetectionReverse tests FD→relational conflict detection.
func TestHybridRegistry_ConflictDetectionReverse(t *testing.T) {
	model := NewModel()
	age1 := Fresh("age1")
	age2 := Fresh("age2")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age1, ageVar)

	// Attempt to map different relational variable to same FD variable
	_, err := registry.MapVars(age2, ageVar)
	if err == nil {
		t.Error("Expected error when mapping different relational var to same FD var")
	}
}

// TestHybridRegistry_IdempotentMapping tests that re-registering same mapping is allowed.
func TestHybridRegistry_IdempotentMapping(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, err1 := registry.MapVars(age, ageVar)
	registry, err2 := registry.MapVars(age, ageVar) // Same mapping again

	if err1 != nil || err2 != nil {
		t.Errorf("Re-registering same mapping should succeed: err1=%v, err2=%v", err1, err2)
	}

	if registry.MappingCount() != 1 {
		t.Errorf("Expected 1 mapping, got %d", registry.MappingCount())
	}
}

// TestHybridRegistry_AutoBind tests automatic binding transfer from query results.
func TestHybridRegistry_AutoBind(t *testing.T) {
	ctx := context.Background()
	model := NewModel()

	// Setup database
	employee, _ := DbRel("employee", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28))

	// Setup registry
	age := Fresh("age")
	name := Fresh("name")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)

	// Execute query
	goal := db.Query(employee, name, age)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(1)

	if len(results) == 0 {
		t.Fatal("No query results")
	}

	// AutoBind should transfer age binding to ageVar
	newStore, err := registry.AutoBind(results[0], store)
	if err != nil {
		t.Fatalf("AutoBind failed: %v", err)
	}

	// Verify binding was transferred
	binding := newStore.GetBinding(int64(ageVar.ID()))
	if binding == nil {
		t.Fatal("AutoBind did not transfer binding")
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

// TestHybridRegistry_AutoBindMultiple tests AutoBind with multiple mapped variables.
func TestHybridRegistry_AutoBindMultiple(t *testing.T) {
	ctx := context.Background()
	model := NewModel()

	// Setup database with 3-column relation
	employee, _ := DbRel("employee", 3, 0)
	db := NewDatabase()
	db, _ = db.AddFact(employee, NewAtom("alice"), NewAtom(28), NewAtom(50000))

	// Setup registry with two mappings
	name := Fresh("name")
	age := Fresh("age")
	salary := Fresh("salary")

	ageVar := model.NewVariable(NewBitSetDomain(100))
	salaryVar := model.NewVariable(NewBitSetDomain(100000))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)
	registry, _ = registry.MapVars(salary, salaryVar)

	// Execute query
	goal := db.Query(employee, name, age, salary)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	results, _ := goal(ctx, adapter).Take(1)

	if len(results) == 0 {
		t.Fatal("No query results")
	}

	// AutoBind should transfer both mappings
	newStore, err := registry.AutoBind(results[0], store)
	if err != nil {
		t.Fatalf("AutoBind failed: %v", err)
	}

	// Verify age binding
	ageBinding := newStore.GetBinding(int64(ageVar.ID()))
	if ageBinding == nil {
		t.Error("Age binding not transferred")
	}

	// Verify salary binding
	salaryBinding := newStore.GetBinding(int64(salaryVar.ID()))
	if salaryBinding == nil {
		t.Error("Salary binding not transferred")
	}
}

// TestHybridRegistry_AutoBindNoBindings tests AutoBind with unbound variables.
func TestHybridRegistry_AutoBindNoBindings(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)

	// Create store with no bindings
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// AutoBind with no bindings should succeed without changes
	newStore, err := registry.AutoBind(adapter, store)
	if err != nil {
		t.Errorf("AutoBind with no bindings failed: %v", err)
	}

	// Store should be unchanged
	if newStore != store {
		t.Error("Expected same store when no bindings present")
	}
}

// TestHybridRegistry_HasMapping tests the HasMapping predicate.
func TestHybridRegistry_HasMapping(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	name := Fresh("name")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()
	registry, _ = registry.MapVars(age, ageVar)

	if !registry.HasMapping(age) {
		t.Error("Expected HasMapping(age) to be true")
	}

	if registry.HasMapping(name) {
		t.Error("Expected HasMapping(name) to be false")
	}

	if registry.HasMapping(nil) {
		t.Error("Expected HasMapping(nil) to be false")
	}
}

// TestHybridRegistry_GetFDVariableNotFound tests lookup of unmapped variable.
func TestHybridRegistry_GetFDVariableNotFound(t *testing.T) {
	age := Fresh("age")
	registry := NewHybridRegistry()

	fdID := registry.GetFDVariable(age)
	if fdID != -1 {
		t.Errorf("Expected -1 for unmapped variable, got %d", fdID)
	}

	nilID := registry.GetFDVariable(nil)
	if nilID != -1 {
		t.Errorf("Expected -1 for nil variable, got %d", nilID)
	}
}

// TestHybridRegistry_GetRelVariableNotFound tests reverse lookup of unmapped variable.
func TestHybridRegistry_GetRelVariableNotFound(t *testing.T) {
	model := NewModel()
	ageVar := model.NewVariable(NewBitSetDomain(100))
	registry := NewHybridRegistry()

	relID := registry.GetRelVariable(ageVar)
	if relID != -1 {
		t.Errorf("Expected -1 for unmapped variable, got %d", relID)
	}

	nilID := registry.GetRelVariable(nil)
	if nilID != -1 {
		t.Errorf("Expected -1 for nil variable, got %d", nilID)
	}
}

// TestHybridRegistry_Clone tests registry cloning.
func TestHybridRegistry_Clone(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	original := NewHybridRegistry()
	original, _ = original.MapVars(age, ageVar)

	clone := original.Clone()

	// Clone should have same mappings
	if clone.MappingCount() != original.MappingCount() {
		t.Error("Clone has different mapping count")
	}

	if clone.GetFDVariable(age) != original.GetFDVariable(age) {
		t.Error("Clone has different mapping")
	}

	// Modifying clone should not affect original
	name := Fresh("name")
	nameVar := model.NewVariable(NewBitSetDomain(100))
	clone, _ = clone.MapVars(name, nameVar)

	if original.MappingCount() == clone.MappingCount() {
		t.Error("Modifying clone affected original")
	}
}

// TestHybridRegistry_NilInputs tests error handling for nil inputs.
func TestHybridRegistry_NilInputs(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	registry := NewHybridRegistry()

	// Test nil relational variable
	_, err := registry.MapVars(nil, ageVar)
	if err == nil {
		t.Error("Expected error for nil relational variable")
	}

	// Test nil FD variable
	_, err = registry.MapVars(age, nil)
	if err == nil {
		t.Error("Expected error for nil FD variable")
	}

	// Test AutoBind with nil store
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)
	_, err = registry.AutoBind(adapter, nil)
	if err == nil {
		t.Error("Expected error for nil unified store")
	}

	// Test AutoBind with nil result (should succeed)
	_, err = registry.AutoBind(nil, store)
	if err != nil {
		t.Errorf("AutoBind with nil result should succeed: %v", err)
	}
}

// TestHybridRegistry_String tests the String representation.
func TestHybridRegistry_String(t *testing.T) {
	// Empty registry
	empty := NewHybridRegistry()
	if empty.String() != "HybridRegistry{empty}" {
		t.Errorf("Empty registry string incorrect: %s", empty.String())
	}

	// Registry with mappings
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))
	registry, _ := empty.MapVars(age, ageVar)

	str := registry.String()
	if len(str) == 0 {
		t.Error("Registry string should not be empty")
	}
	// Should contain the mapping
	// Format is "HybridRegistry{relID→fdID}"
	// Just verify it's not empty and contains expected elements
	if str == "HybridRegistry{empty}" {
		t.Error("Registry with mappings should not show as empty")
	}
}

// TestHybridRegistry_Immutability tests that registry operations are immutable.
func TestHybridRegistry_Immutability(t *testing.T) {
	model := NewModel()
	age := Fresh("age")
	name := Fresh("name")
	ageVar := model.NewVariable(NewBitSetDomain(100))
	nameVar := model.NewVariable(NewBitSetDomain(100))

	r1 := NewHybridRegistry()
	r2, _ := r1.MapVars(age, ageVar)
	r3, _ := r2.MapVars(name, nameVar)

	// Each operation should create a new instance
	if r1.MappingCount() != 0 {
		t.Error("r1 should have 0 mappings")
	}
	if r2.MappingCount() != 1 {
		t.Error("r2 should have 1 mapping")
	}
	if r3.MappingCount() != 2 {
		t.Error("r3 should have 2 mappings")
	}
}
