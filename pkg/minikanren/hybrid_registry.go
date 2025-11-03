// Package minikanren provides hybrid integration between relational and
// finite-domain constraint solving.
//
// This file implements HybridRegistry for managing variable mappings between
// pldb relational variables and FD constraint variables. The registry maintains
// bidirectional mappings and provides automatic binding propagation.
//
// Design Philosophy:
//   - Explicit mappings: Users control which variables map to each other
//   - Bidirectional: Maps both relational→FD and FD→relational
//   - Immutable operations: All mapping operations return new registry instances
//   - Type-safe: Validates mappings at registration time
//
// The registry solves the "variable coordination problem" in hybrid systems:
// when a database query binds a relational variable, how do we propagate that
// binding to the corresponding FD variable for constraint propagation?
package minikanren

import (
	"fmt"
)

// HybridRegistry maintains bidirectional mappings between relational variables
// (pldb) and finite-domain variables (FD constraints).
//
// The registry enables automatic propagation of bindings between the two
// variable spaces, eliminating boilerplate code in hybrid queries.
//
// Usage Pattern:
//  1. Create registry with NewHybridRegistry()
//  2. Register variable pairs with MapVars(relVar, fdVar)
//  3. Execute hybrid query producing bindings
//  4. Apply bindings with AutoBind(result, store)
//
// Thread Safety: Registry instances are immutable. All operations return
// new registry instances, making them safe for concurrent use.
type HybridRegistry struct {
	// relToFD maps relational variable IDs to FD variable IDs
	relToFD map[int64]int

	// fdToRel maps FD variable IDs to relational variable IDs
	fdToRel map[int]int64
}

// NewHybridRegistry creates an empty variable mapping registry.
//
// Returns a registry with no mappings. Use MapVars() to register
// variable relationships.
//
// Example:
//
//	registry := NewHybridRegistry()
//	registry, _ = registry.MapVars(ageRelVar, ageFDVar)
//	registry, _ = registry.MapVars(nameRelVar, nameFDVar)
func NewHybridRegistry() *HybridRegistry {
	return &HybridRegistry{
		relToFD: make(map[int64]int),
		fdToRel: make(map[int]int64),
	}
}

// MapVars registers a bidirectional mapping between a relational variable
// and an FD variable.
//
// Parameters:
//   - relVar: The relational logic variable (from Fresh())
//   - fdVar: The FD constraint variable (from model.NewVariable())
//
// Returns:
//   - New registry instance with the mapping added
//   - Error if variables are nil or mapping would conflict
//
// The mapping is bidirectional: the registry can look up either direction.
// Attempting to register the same variable twice (in either space) returns
// an error to prevent ambiguous mappings.
//
// Example:
//
//	age := Fresh("age")
//	ageVar := model.NewVariable(NewBitSetDomain(100))
//	registry, err := NewHybridRegistry().MapVars(age, ageVar)
//	if err != nil {
//	    panic(err)
//	}
func (r *HybridRegistry) MapVars(relVar *Var, fdVar *FDVariable) (*HybridRegistry, error) {
	if relVar == nil {
		return r, fmt.Errorf("relational variable cannot be nil")
	}
	if fdVar == nil {
		return r, fmt.Errorf("FD variable cannot be nil")
	}

	relID := relVar.ID()
	fdID := fdVar.ID()

	// Check for conflicts
	if existingFD, exists := r.relToFD[relID]; exists {
		if existingFD != fdID {
			return r, fmt.Errorf("relational variable %d already mapped to FD variable %d", relID, existingFD)
		}
		// Same mapping already exists - return unchanged
		return r, nil
	}

	if existingRel, exists := r.fdToRel[fdID]; exists {
		if existingRel != relID {
			return r, fmt.Errorf("FD variable %d already mapped to relational variable %d", fdID, existingRel)
		}
		// Same mapping already exists - return unchanged
		return r, nil
	}

	// Create new registry with additional mapping (immutable operation)
	newRelToFD := make(map[int64]int, len(r.relToFD)+1)
	for k, v := range r.relToFD {
		newRelToFD[k] = v
	}
	newRelToFD[relID] = fdID

	newFdToRel := make(map[int]int64, len(r.fdToRel)+1)
	for k, v := range r.fdToRel {
		newFdToRel[k] = v
	}
	newFdToRel[fdID] = relID

	return &HybridRegistry{
		relToFD: newRelToFD,
		fdToRel: newFdToRel,
	}, nil
}

// GetFDVariable returns the FD variable ID mapped to the given relational
// variable, or -1 if no mapping exists.
//
// Parameters:
//   - relVar: The relational variable to look up
//
// Returns:
//   - FD variable ID if mapping exists
//   - -1 if no mapping exists or relVar is nil
//
// Example:
//
//	fdID := registry.GetFDVariable(age)
//	if fdID != -1 {
//	    // Mapping exists, use fdID
//	}
func (r *HybridRegistry) GetFDVariable(relVar *Var) int {
	if relVar == nil {
		return -1
	}
	if fdID, exists := r.relToFD[relVar.ID()]; exists {
		return fdID
	}
	return -1
}

// GetRelVariable returns the relational variable ID mapped to the given FD
// variable, or -1 if no mapping exists.
//
// Parameters:
//   - fdVar: The FD variable to look up
//
// Returns:
//   - Relational variable ID if mapping exists
//   - -1 if no mapping exists or fdVar is nil
//
// Example:
//
//	relID := registry.GetRelVariable(ageVar)
//	if relID != -1 {
//	    // Mapping exists, use relID
//	}
func (r *HybridRegistry) GetRelVariable(fdVar *FDVariable) int64 {
	if fdVar == nil {
		return -1
	}
	if relID, exists := r.fdToRel[fdVar.ID()]; exists {
		return relID
	}
	return -1
}

// AutoBind automatically transfers bindings from a query result to the
// UnifiedStore based on registered variable mappings.
//
// For each mapped variable:
//  1. Extract binding from query result
//  2. Apply binding to corresponding FD variable in store
//  3. Return updated store with all mapped bindings
//
// Parameters:
//   - result: Query result containing relational variable bindings
//   - store: UnifiedStore to update with FD variable bindings
//
// Returns:
//   - New UnifiedStore with mapped bindings applied
//   - Error if binding transfer fails
//
// This eliminates the manual mapping boilerplate:
//
//	// Without AutoBind (manual):
//	ageBinding := result.GetBinding(age.ID())
//	store, _ = store.AddBinding(int64(ageVar.ID()), ageBinding)
//	nameBinding := result.GetBinding(name.ID())
//	store, _ = store.AddBinding(int64(nameVar.ID()), nameBinding)
//
//	// With AutoBind (automatic):
//	store, _ = registry.AutoBind(result, store)
//
// Thread Safety: Safe for concurrent use. Returns new store instances.
func (r *HybridRegistry) AutoBind(result ConstraintStore, store *UnifiedStore) (*UnifiedStore, error) {
	if result == nil {
		return store, nil
	}
	if store == nil {
		return nil, fmt.Errorf("unified store cannot be nil")
	}

	// Iterate over all registered relational→FD mappings
	for relID, fdID := range r.relToFD {
		// Get binding from query result
		binding := result.GetBinding(relID)
		if binding == nil {
			// No binding for this variable - skip
			continue
		}

		// Transfer binding to FD variable in unified store
		newStore, err := store.AddBinding(int64(fdID), binding)
		if err != nil {
			return store, fmt.Errorf("failed to bind FD variable %d: %w", fdID, err)
		}
		store = newStore
	}

	return store, nil
}

// MappingCount returns the number of variable mappings in the registry.
//
// Useful for debugging and testing to verify registration succeeded.
//
// Example:
//
//	registry, _ = registry.MapVars(age, ageVar)
//	registry, _ = registry.MapVars(name, nameVar)
//	if registry.MappingCount() != 2 {
//	    panic("expected 2 mappings")
//	}
func (r *HybridRegistry) MappingCount() int {
	return len(r.relToFD)
}

// HasMapping returns true if a mapping exists for the given relational variable.
//
// Parameters:
//   - relVar: The relational variable to check
//
// Returns:
//   - true if mapping exists
//   - false if no mapping or relVar is nil
//
// Example:
//
//	if registry.HasMapping(age) {
//	    // Safe to use AutoBind for age
//	}
func (r *HybridRegistry) HasMapping(relVar *Var) bool {
	if relVar == nil {
		return false
	}
	_, exists := r.relToFD[relVar.ID()]
	return exists
}

// Clone creates a copy of the registry with the same mappings.
//
// Since registries are immutable, this returns a new instance with
// independent map storage but identical content.
//
// Useful when you need to create independent registry branches for
// different query contexts.
//
// Example:
//
//	baseRegistry := NewHybridRegistry()
//	baseRegistry, _ = baseRegistry.MapVars(age, ageVar)
//
//	// Create specialized registries for different queries
//	query1Registry, _ = baseRegistry.Clone().MapVars(name, nameVar)
//	query2Registry, _ = baseRegistry.Clone().MapVars(salary, salaryVar)
func (r *HybridRegistry) Clone() *HybridRegistry {
	newRelToFD := make(map[int64]int, len(r.relToFD))
	for k, v := range r.relToFD {
		newRelToFD[k] = v
	}

	newFdToRel := make(map[int]int64, len(r.fdToRel))
	for k, v := range r.fdToRel {
		newFdToRel[k] = v
	}

	return &HybridRegistry{
		relToFD: newRelToFD,
		fdToRel: newFdToRel,
	}
}

// String returns a human-readable representation of the registry.
//
// Shows all registered mappings in the format:
//
//	HybridRegistry{rel_id → fd_id, ...}
//
// Useful for debugging and logging.
func (r *HybridRegistry) String() string {
	if len(r.relToFD) == 0 {
		return "HybridRegistry{empty}"
	}

	result := "HybridRegistry{"
	first := true
	for relID, fdID := range r.relToFD {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%d→%d", relID, fdID)
		first = false
	}
	result += "}"
	return result
}
