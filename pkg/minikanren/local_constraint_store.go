// Package minikanren provides the LocalConstraintStore implementation for
// managing constraints and variable bindings within individual goal contexts.
//
// The LocalConstraintStore is the core component of the hybrid constraint system,
// providing fast local constraint checking while coordinating with the global
// constraint bus when necessary for cross-store constraints.
//
// Key design principles:
//   - Fast path: Local constraint checking without coordination overhead
//   - Slow path: Global coordination only when cross-store constraints are involved
//   - Thread-safe: Safe for concurrent access and parallel goal evaluation
//   - Efficient cloning: Optimized for parallel execution where stores are frequently copied
package minikanren

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// LocalConstraintStoreImpl provides a concrete implementation of LocalConstraintStore
// for managing constraints and variable bindings within a single goal context.
//
// The store maintains two separate collections:
//   - Local constraints: Checked quickly without global coordination
//   - Local bindings: Variable-to-term mappings for this context
//
// When constraints or bindings are added, the store first checks all local
// constraints for immediate violations, then coordinates with the global
// bus if necessary for cross-store constraints.
type LocalConstraintStoreImpl struct {
	// id uniquely identifies this store instance
	id string

	// constraints holds all local constraints for this store
	constraints []Constraint

	// bindings maps variable IDs to their bound terms
	bindings map[int64]Term

	// globalBus coordinates cross-store constraints (optional)
	globalBus *GlobalConstraintBus

	// generation tracks the number of modifications for efficient cloning
	generation int64

	// mu protects concurrent access to store state
	mu sync.RWMutex
}

// storeCounter provides unique IDs for store instances
var storeCounter int64

// NewLocalConstraintStore creates a new local constraint store with
// optional global constraint bus integration.
//
// If globalBus is nil, the store operates in local-only mode with
// no cross-store constraint coordination. This is suitable for
// simple use cases where all constraints are local.
func NewLocalConstraintStore(globalBus *GlobalConstraintBus) *LocalConstraintStoreImpl {
	id := fmt.Sprintf("store-%d", atomic.AddInt64(&storeCounter, 1))

	store := &LocalConstraintStoreImpl{
		id:        id,
		bindings:  make(map[int64]Term),
		globalBus: globalBus,
	}

	// Register with global bus if available
	if globalBus != nil {
		globalBus.RegisterStore(store)
	}

	return store
}

// ID returns the unique identifier for this constraint store.
// Implements the LocalConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) ID() string {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()
	return lcs.id
}

// AddConstraint adds a new constraint to the store and checks it
// against current bindings for immediate violations.
//
// The constraint is first checked locally for immediate violations.
// If the constraint is not local (requires global coordination),
// it is also registered with the global constraint bus.
//
// Returns an error if the constraint is immediately violated.
func (lcs *LocalConstraintStoreImpl) AddConstraint(constraint Constraint) error {
	lcs.mu.Lock()
	defer lcs.mu.Unlock()

	// Check if the constraint is immediately violated by current bindings
	result := constraint.Check(lcs.bindings)
	if result == ConstraintViolated {
		return fmt.Errorf("constraint %s immediately violated by current bindings", constraint.ID())
	}

	// Add to local constraints list
	lcs.constraints = append(lcs.constraints, constraint)
	lcs.generation++

	// If constraint requires global coordination, register with bus
	if !constraint.IsLocal() && lcs.globalBus != nil {
		err := lcs.globalBus.AddCrossStoreConstraint(constraint)
		if err != nil {
			// Remove from local constraints if global registration failed
			lcs.constraints = lcs.constraints[:len(lcs.constraints)-1]
			lcs.generation++
			return fmt.Errorf("failed to register cross-store constraint: %w", err)
		}
	}

	return nil
}

// AddBinding attempts to bind a variable to a term, checking all
// relevant constraints for violations.
//
// The binding process follows these steps:
//  1. Check all local constraints against the proposed binding
//  2. If any local constraint is violated, reject the binding
//  3. If the binding affects cross-store constraints, coordinate with global bus
//  4. If all checks pass, add the binding to the store
//
// Returns an error if the binding would violate any constraint.
func (lcs *LocalConstraintStoreImpl) AddBinding(varID int64, term Term) error {
	lcs.mu.Lock()
	defer lcs.mu.Unlock()

	// Create a temporary binding map to test constraints
	testBindings := make(map[int64]Term, len(lcs.bindings)+1)
	for id, binding := range lcs.bindings {
		testBindings[id] = binding
	}
	testBindings[varID] = term

	// Check all local constraints against the proposed binding
	for _, constraint := range lcs.constraints {
		result := constraint.Check(testBindings)
		if result == ConstraintViolated {
			return fmt.Errorf("binding var_%d = %s would violate constraint %s",
				varID, term.String(), constraint.ID())
		}
	}

	// Check cross-store constraints if global bus is available
	if lcs.globalBus != nil {
		err := lcs.globalBus.CoordinateBinding(varID, term, lcs.id)
		if err != nil {
			return fmt.Errorf("cross-store constraint violation: %w", err)
		}
	}

	// All constraints satisfied - add the binding
	lcs.bindings[varID] = term
	lcs.generation++

	// Notify global bus about the new binding
	if lcs.globalBus != nil {
		event := ConstraintEvent{
			Type:      VariableBound,
			StoreID:   lcs.id,
			VarID:     varID,
			Term:      term,
			Timestamp: atomic.AddInt64(&lcs.globalBus.eventCounter, 1),
		}
		// Non-blocking send; safe even if bus is shutting down
		_ = lcs.globalBus.trySend(event)
	}

	return nil
}

// GetBinding retrieves the current binding for a variable.
// Returns nil if the variable is unbound.
// Implements the ConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) GetBinding(varID int64) Term {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	return lcs.bindings[varID] // Returns nil if not found
}

// getAllBindings returns a copy of all current bindings.
// Used by the global constraint bus for cross-store constraint checking.
// Implements the LocalConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) getAllBindings() map[int64]Term {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	// Return a copy to avoid concurrent modification issues
	bindings := make(map[int64]Term, len(lcs.bindings))
	for id, term := range lcs.bindings {
		bindings[id] = term
	}
	return bindings
}

// GetSubstitution returns a substitution representing all current bindings.
// This bridges between the constraint store system and the existing
// miniKanren substitution-based APIs.
// Implements the ConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) GetSubstitution() *Substitution {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	sub := NewSubstitution()

	// Convert bindings to substitution format
	for varID, term := range lcs.bindings {
		// Create a temporary variable with the correct ID for the substitution
		tempVar := &Var{id: varID}
		sub = sub.Bind(tempVar, term)
	}

	return sub
}

// Clone creates a deep copy of the constraint store for parallel execution.
// The clone shares no mutable state with the original store, making it
// safe for concurrent use in parallel goal evaluation.
//
// Cloning is optimized for performance as it's used frequently in
// parallel execution contexts. The clone initially shares constraint
// references with the original but will copy-on-write if modified.
// Implements the ConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) Clone() ConstraintStore {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	// Create new store with unique ID
	cloneID := fmt.Sprintf("%s-clone-%d", lcs.id, lcs.generation)

	clone := &LocalConstraintStoreImpl{
		id:         cloneID,
		globalBus:  lcs.globalBus, // Share reference to global bus
		generation: 0,             // Reset generation counter for clone
	}

	// Deep copy bindings
	clone.bindings = make(map[int64]Term, len(lcs.bindings))
	for id, term := range lcs.bindings {
		clone.bindings[id] = term.Clone()
	}

	// Deep copy constraints
	clone.constraints = make([]Constraint, len(lcs.constraints))
	for i, constraint := range lcs.constraints {
		clone.constraints[i] = constraint.Clone()
	}

	// Do NOT register clones with the global bus. Clones are ephemeral evaluation
	// contexts and registering each clone causes unbounded growth in the bus registry
	// with no corresponding shutdown, leading to goroutine leaks and timeouts in tests.
	// Cross-store coordination (if required) should be handled at higher levels.

	return clone
}

// String returns a human-readable representation of the constraint store
// for debugging and error reporting.
// Implements the ConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) String() string {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	return fmt.Sprintf("LocalConstraintStore{id: %s, constraints: %d, bindings: %d, generation: %d}",
		lcs.id, len(lcs.constraints), len(lcs.bindings), lcs.generation)
}

// GetConstraints returns a copy of all constraints in the store.
// Used for debugging and testing purposes.
func (lcs *LocalConstraintStoreImpl) GetConstraints() []Constraint {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	// Return a copy to prevent external modification
	constraints := make([]Constraint, len(lcs.constraints))
	copy(constraints, lcs.constraints)
	return constraints
}

// IsEmpty returns true if the store has no constraints or bindings.
// Useful for optimization and testing.
func (lcs *LocalConstraintStoreImpl) IsEmpty() bool {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	return len(lcs.constraints) == 0 && len(lcs.bindings) == 0
}

// Generation returns the current generation number of the store.
// The generation increments with each modification, enabling
// efficient change detection and caching strategies.
func (lcs *LocalConstraintStoreImpl) Generation() int64 {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	return lcs.generation
}

// Shutdown cleanly shuts down the store and unregisters it from
// the global constraint bus. Should be called when the store
// is no longer needed to prevent memory leaks.
func (lcs *LocalConstraintStoreImpl) Shutdown() {
	lcs.mu.Lock()
	defer lcs.mu.Unlock()

	if lcs.globalBus != nil {
		lcs.globalBus.UnregisterStore(lcs.id)
	}

	// Clear internal state to help garbage collection
	lcs.constraints = nil
	lcs.bindings = nil
	lcs.globalBus = nil
}
