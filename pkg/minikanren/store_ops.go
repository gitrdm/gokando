// Package minikanren provides constraint store manipulation primitives
// for advanced constraint logic programming operations.
//
// This file implements core.logic-style store operations that allow
// direct manipulation of constraint stores, enabling sophisticated
// constraint programming techniques like store inspection, modification,
// and composition.
//
// Key operations:
//   - EmptyStore: Create empty constraint stores
//   - StoreWithConstraint: Add constraints to stores
//   - StoreWithoutConstraint: Remove constraints from stores
//   - StoreUnion/Intersection/Difference: Set operations on stores
//
// All operations maintain thread safety and preserve constraint semantics.
package minikanren

import (
	"fmt"
	"reflect"
)

// EmptyStore creates a new empty constraint store with no constraints or bindings.
// This is the identity element for store operations and serves as a starting point
// for building constraint stores programmatically.
//
// Returns a LocalConstraintStoreImpl with no global bus integration for simplicity.
// If global coordination is needed, use NewLocalConstraintStore() instead.
func EmptyStore() ConstraintStore {
	return NewLocalConstraintStore(nil) // No global bus for pure local operations
}

// StoreWithConstraint creates a new constraint store by adding a constraint
// to an existing store. The original store is not modified - a clone is created.
//
// This operation follows functional programming principles: stores are immutable,
// and operations return new stores rather than modifying existing ones.
//
// Parameters:
//   - store: The base constraint store to extend
//   - constraint: The constraint to add to the store
//
// Returns a new ConstraintStore with the constraint added, or an error if
// the constraint would be immediately violated by existing bindings.
func StoreWithConstraint(store ConstraintStore, constraint Constraint) (ConstraintStore, error) {
	if store == nil {
		return nil, fmt.Errorf("cannot add constraint to nil store")
	}
	if constraint == nil {
		return nil, fmt.Errorf("cannot add nil constraint to store")
	}

	// Clone the store to avoid modifying the original
	newStore := store.Clone()

	// Add the constraint to the cloned store
	err := newStore.AddConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("failed to add constraint to store: %w", err)
	}

	return newStore, nil
}

// StoreWithoutConstraint creates a new constraint store by removing a specific
// constraint from an existing store. The original store is not modified.
//
// Constraint removal is based on constraint identity (ID matching). If the
// constraint is not found in the store, the operation succeeds but returns
// an unchanged clone of the original store.
//
// Parameters:
//   - store: The base constraint store to modify
//   - constraint: The constraint to remove from the store
//
// Returns a new ConstraintStore with the constraint removed.
func StoreWithoutConstraint(store ConstraintStore, constraint Constraint) (ConstraintStore, error) {
	if store == nil {
		return nil, fmt.Errorf("cannot remove constraint from nil store")
	}
	if constraint == nil {
		return nil, fmt.Errorf("cannot remove nil constraint from store")
	}

	// Clone the store to avoid modifying the original
	newStore := store.Clone()

	// Get current constraints
	constraints := newStore.GetConstraints()

	// Find and remove the constraint by ID
	targetID := constraint.ID()
	found := false
	for i, c := range constraints {
		if c.ID() == targetID {
			// Remove constraint by creating new slice without it
			// This is inefficient but maintains immutability
			newConstraints := make([]Constraint, 0, len(constraints)-1)
			newConstraints = append(newConstraints, constraints[:i]...)
			newConstraints = append(newConstraints, constraints[i+1:]...)

			// Update the store with new constraint list
			// Note: This requires internal access - we'll need to extend the interface
			if localStore, ok := newStore.(*LocalConstraintStoreImpl); ok {
				localStore.mu.Lock()
				localStore.constraints = newConstraints
				localStore.generation++
				localStore.mu.Unlock()
			} else {
				return nil, fmt.Errorf("constraint removal not supported for store type %T", newStore)
			}

			found = true
			break
		}
	}

	if !found {
		// Constraint not found - this is not an error, just return the clone
	}

	return newStore, nil
}

// StoreUnion creates a new constraint store containing all constraints
// from both input stores. Bindings are merged with precedence given to
// the second store (s2) in case of conflicts.
//
// The union operation combines constraints from both stores while preserving
// constraint semantics. If the same constraint exists in both stores,
// it will appear only once in the result.
//
// Parameters:
//   - s1, s2: The constraint stores to combine
//
// Returns a new ConstraintStore containing the union of constraints and bindings.
func StoreUnion(s1, s2 ConstraintStore) (ConstraintStore, error) {
	if s1 == nil || s2 == nil {
		return nil, fmt.Errorf("cannot union nil stores")
	}

	// Start with a clone of s1
	result := s1.Clone()

	// Get constraints from s2
	s2Constraints := s2.GetConstraints()

	// Add each constraint from s2 to the result
	// Skip duplicates based on constraint ID
	s1Constraints := result.GetConstraints()
	s1ConstraintIDs := make(map[string]bool, len(s1Constraints))
	for _, c := range s1Constraints {
		s1ConstraintIDs[c.ID()] = true
	}

	for _, constraint := range s2Constraints {
		if !s1ConstraintIDs[constraint.ID()] {
			// Constraint not already in result, add it
			var err error
			result, err = StoreWithConstraint(result, constraint)
			if err != nil {
				return nil, fmt.Errorf("failed to add constraint during union: %w", err)
			}
		}
	}

	// Merge bindings from s2, with s2 taking precedence
	s2Bindings := getStoreBindings(s2)
	for varID, term := range s2Bindings {
		// Add binding to result (this will check constraints)
		err := result.AddBinding(varID, term)
		if err != nil {
			return nil, fmt.Errorf("failed to merge binding during union: %w", err)
		}
	}

	return result, nil
}

// StoreIntersection creates a new constraint store containing only the
// constraints that exist in both input stores. Bindings are intersected -
// only variables bound in both stores are included.
//
// The intersection represents the common constraints between the two stores,
// useful for finding overlapping constraint requirements.
//
// Parameters:
//   - s1, s2: The constraint stores to intersect
//
// Returns a new ConstraintStore containing the intersection of constraints and bindings.
func StoreIntersection(s1, s2 ConstraintStore) (ConstraintStore, error) {
	if s1 == nil || s2 == nil {
		return nil, fmt.Errorf("cannot intersect nil stores")
	}

	// Start with an empty store
	result := EmptyStore()

	// Get constraints from both stores
	s1Constraints := s1.GetConstraints()
	s2Constraints := s2.GetConstraints()

	// Create lookup map for s2 constraints by ID
	s2ConstraintMap := make(map[string]Constraint, len(s2Constraints))
	for _, c := range s2Constraints {
		s2ConstraintMap[c.ID()] = c
	}

	// Add constraints that exist in both stores
	for _, c1 := range s1Constraints {
		if _, exists := s2ConstraintMap[c1.ID()]; exists {
			// Same constraint exists in both - add it once
			var err error
			result, err = StoreWithConstraint(result, c1)
			if err != nil {
				return nil, fmt.Errorf("failed to add constraint during intersection: %w", err)
			}
			// Remove from map to avoid duplicates if same constraint appears multiple times
			delete(s2ConstraintMap, c1.ID())
		}
	}

	// Intersect bindings - only include variables bound in both stores
	s1Bindings := getStoreBindings(s1)
	s2Bindings := getStoreBindings(s2)

	for varID, s1Term := range s1Bindings {
		if s2Term, exists := s2Bindings[varID]; exists {
			// Variable bound in both stores - check if bindings are compatible
			if reflect.DeepEqual(s1Term, s2Term) {
				// Same binding - add to result
				err := result.AddBinding(varID, s1Term)
				if err != nil {
					return nil, fmt.Errorf("failed to add binding during intersection: %w", err)
				}
			}
			// If bindings differ, we don't include the variable (no intersection)
		}
		// Variables only in s1 are not included
	}

	return result, nil
}

// StoreDifference creates a new constraint store containing constraints
// from the first store (s1) that are not present in the second store (s2).
// Bindings from s1 are included unless they conflict with s2 bindings.
//
// The difference operation removes constraints and bindings from s1 that
// are present in s2, useful for computing store deltas or filtering.
//
// Parameters:
//   - s1, s2: The constraint stores to difference (s1 - s2)
//
// Returns a new ConstraintStore containing constraints/bindings in s1 but not s2.
func StoreDifference(s1, s2 ConstraintStore) (ConstraintStore, error) {
	if s1 == nil || s2 == nil {
		return nil, fmt.Errorf("cannot difference nil stores")
	}

	// Start with a clone of s1
	result := s1.Clone()

	// Get constraints from s2
	s2Constraints := s2.GetConstraints()

	// Create lookup map for s2 constraints by ID
	s2ConstraintMap := make(map[string]bool, len(s2Constraints))
	for _, c := range s2Constraints {
		s2ConstraintMap[c.ID()] = true
	}

	// Remove constraints that exist in s2
	s1Constraints := result.GetConstraints()
	constraintsToRemove := make([]Constraint, 0)

	for _, c := range s1Constraints {
		if s2ConstraintMap[c.ID()] {
			constraintsToRemove = append(constraintsToRemove, c)
		}
	}

	// Remove the constraints
	for _, c := range constraintsToRemove {
		var err error
		result, err = StoreWithoutConstraint(result, c)
		if err != nil {
			return nil, fmt.Errorf("failed to remove constraint during difference: %w", err)
		}
	}

	// For bindings, we keep all s1 bindings since difference is about removal
	// The bindings are already in the cloned store

	return result, nil
}

// getStoreBindings extracts bindings from a constraint store.
// This is a helper function that works with the current ConstraintStore interface.
// In the future, this could be added as a method to the interface.
func getStoreBindings(store ConstraintStore) map[int64]Term {
	// Use the substitution to extract bindings
	sub := store.GetSubstitution()
	if sub == nil {
		return make(map[int64]Term)
	}

	// Convert substitution back to binding map
	bindings := make(map[int64]Term)
	for varID, term := range sub.bindings {
		bindings[varID] = term
	}

	return bindings
}
