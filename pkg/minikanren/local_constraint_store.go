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
//   - Pool-friendly: Designed for efficient reuse through object pooling
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
//   - Local constraints: Checked against bindings for immediate violations
//   - Local bindings: Variable-to-term mappings for this context
//
// When constraints or bindings are added, the store first checks all local
// constraints for immediate violations, then coordinates with the global
// bus if necessary for cross-store constraints.
type LocalConstraintStoreImpl struct {
	// id uniquely identifies this store instance
	id string

	// constraints holds the local constraints for this store
	constraints []Constraint

	// bindings holds variable-to-term mappings
	bindings map[int64]Term

	// globalBus coordinates cross-store constraints (optional)
	globalBus *GlobalConstraintBus

	// generation tracks the number of modifications for efficient cloning
	generation int64

	// mu protects concurrent access to store state
	mu sync.RWMutex
}

// StoreRef provides a lightweight reference to a constraint store for zero-copy streaming.
// Instead of passing full store instances through channels, we pass StoreRefs that
// can be resolved to actual stores when needed. This enables efficient streaming
// where stores are only copied when they actually diverge.
type StoreRef struct {
	storeID  string
	resolver StoreResolver
}

// StoreResolver provides a way to resolve StoreRefs back to actual stores.
// This allows the streaming system to lazily resolve stores only when needed.
type StoreResolver interface {
	ResolveStore(ref StoreRef) ConstraintStore
}

// NewStoreRef creates a new StoreRef for the given store.
func NewStoreRef(store ConstraintStore) StoreRef {
	if localStore, ok := store.(*LocalConstraintStoreImpl); ok {
		return StoreRef{
			storeID:  localStore.id,
			resolver: nil, // Will be set by the streaming system
		}
	}
	// For other store types, create a ref that directly holds the store
	return StoreRef{
		storeID:  "direct",
		resolver: &DirectStoreResolver{store: store},
	}
}

// Resolve returns the actual store from this reference.
// If the resolver is set, it delegates to the resolver; otherwise
// returns a direct reference.
func (sr StoreRef) Resolve() ConstraintStore {
	if sr.resolver != nil {
		return sr.resolver.ResolveStore(sr)
	}
	// This shouldn't happen in normal operation
	panic("StoreRef has no resolver")
}

// DirectStoreResolver directly holds a store reference for simple cases.
type DirectStoreResolver struct {
	store ConstraintStore
}

// ResolveStore returns the directly held store.
func (dsr *DirectStoreResolver) ResolveStore(ref StoreRef) ConstraintStore {
	return dsr.store
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
		id:          id,
		constraints: make([]Constraint, 0),
		bindings:    make(map[int64]Term),
		globalBus:   globalBus,
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
		select {
		case <-lcs.globalBus.shutdownCh:
			// Bus is shutting down, don't send event
		default:
			event := ConstraintEvent{
				Type:      VariableBound,
				StoreID:   lcs.id,
				VarID:     varID,
				Term:      term,
				Timestamp: atomic.AddInt64(&lcs.globalBus.eventCounter, 1),
			}

			// Non-blocking send to avoid deadlocks
			select {
			case lcs.globalBus.events <- event:
			default:
				// Event queue full - log but don't fail the binding
				// In production, this might warrant more sophisticated handling
			}
		}
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

// Clone creates a copy of the constraint store for parallel execution.
// Creates a deep copy of constraints and bindings for thread safety.
//
// Cloning is used frequently in parallel execution contexts where each
// goal needs its own copy of the constraint state.
// Implements the ConstraintStore interface.
func (lcs *LocalConstraintStoreImpl) Clone() ConstraintStore {
	lcs.mu.RLock()
	defer lcs.mu.RUnlock()

	// Create new store with unique ID
	cloneID := fmt.Sprintf("%s-clone-%d", lcs.id, lcs.generation)

	// Deep copy constraints
	constraints := make([]Constraint, len(lcs.constraints))
	for i, constraint := range lcs.constraints {
		constraints[i] = constraint.Clone()
	}

	// Deep copy bindings
	bindings := make(map[int64]Term, len(lcs.bindings))
	for id, term := range lcs.bindings {
		bindings[id] = term.Clone()
	}

	clone := &LocalConstraintStoreImpl{
		id:          cloneID,
		constraints: constraints,
		bindings:    bindings,
		globalBus:   lcs.globalBus,
		generation:  0, // Reset generation counter for clone
	}

	// Register clone with global bus if available
	if lcs.globalBus != nil {
		// Check if bus is still active before registering
		lcs.globalBus.RegisterStore(clone)

		// Notify about cloning event only if bus is still active
		// Use a non-blocking approach to avoid issues with shutdown
		select {
		case <-lcs.globalBus.shutdownCh:
			// Bus is shutting down, don't send event
		default:
			event := ConstraintEvent{
				Type:      StoreCloned,
				StoreID:   cloneID,
				Timestamp: atomic.AddInt64(&lcs.globalBus.eventCounter, 1),
			}

			select {
			case lcs.globalBus.events <- event:
			default:
				// Event queue full or bus shutting down - continue anyway
			}
		}
	}

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

	// Clear references to help garbage collection
	lcs.globalBus = nil
	lcs.constraints = nil
	lcs.bindings = nil
}

// Reset prepares the constraint store for reuse in a buffer pool.
// This method clears all state while keeping the store structure intact
// for efficient reuse in high-throughput streaming scenarios.
func (lcs *LocalConstraintStoreImpl) Reset() {
	lcs.mu.Lock()
	defer lcs.mu.Unlock()

	// Clear all constraints and bindings while preserving capacity
	lcs.constraints = lcs.constraints[:0] // Reuse slice capacity
	lcs.bindings = make(map[int64]Term)   // Reset bindings map
	lcs.generation = 0

	// Note: We keep the same ID and globalBus reference for reuse
	// The global bus is reset separately if needed
}
