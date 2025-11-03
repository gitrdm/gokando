// Package minikanren provides adapters for integrating UnifiedStore
// with the ConstraintStore interface, enabling hybrid pldb queries.
//
// The UnifiedStoreAdapter bridges between the UnifiedStore (Phase 3 hybrid solver)
// and the ConstraintStore interface used by miniKanren goals. This adapter enables
// pldb queries to work seamlessly with FD constraints and bidirectional propagation.
//
// Design rationale:
//   - UnifiedStore has methods that return (*UnifiedStore, error) for immutability
//   - ConstraintStore interface expects methods that return error for in-place modification
//   - Adapter maintains a reference to current store version and updates on mutations
//   - Thread-safe through UnifiedStore's immutability and adapter's synchronization
//
// Usage pattern:
//
//	store := NewUnifiedStore()
//	adapter := NewUnifiedStoreAdapter(store)
//
//	// Use with pldb queries
//	stream := db.Query(person, name, age)(ctx, adapter)
//
//	// Access underlying store for hybrid solver propagation
//	hybridStore := adapter.UnifiedStore()
//	propagatedStore, err := solver.Propagate(hybridStore)
//	adapter.SetUnifiedStore(propagatedStore)
package minikanren

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// UnifiedStoreAdapter wraps a UnifiedStore to implement the ConstraintStore interface.
// This enables UnifiedStore to be used with miniKanren goals, including pldb queries,
// while maintaining the hybrid solver's copy-on-write semantics.
//
// The adapter is thread-safe: multiple goroutines can safely read from an adapter,
// and mutations are protected by a mutex. However, for parallel search, each branch
// should clone the adapter to get an independent store version.
//
// Lifecycle:
//  1. Create adapter wrapping a UnifiedStore
//  2. Use adapter as ConstraintStore in goals (pldb queries, unification, etc.)
//  3. Extract UnifiedStore for hybrid propagation
//  4. Update adapter with propagated store
//  5. Clone adapter for search branching
//
// Performance notes:
//   - Adapter overhead is minimal (single pointer dereference + mutex in write path)
//   - UnifiedStore's copy-on-write means cloning is O(1)
//   - Constraint checking delegates to UnifiedStore's constraint system
type UnifiedStoreAdapter struct {
	// store holds the current version of the UnifiedStore.
	// Updated atomically on mutations.
	store *UnifiedStore

	// mu protects concurrent access to store pointer updates.
	// Read operations on UnifiedStore don't need locks (immutable data).
	// Write operations (mutations that create new store versions) do.
	mu sync.RWMutex

	// idCounter generates unique IDs for adapter instances.
	// Used for debugging and identifying store lineage.
	id string
}

var adapterIDCounter int64

// NewUnifiedStoreAdapter creates a ConstraintStore adapter wrapping the given UnifiedStore.
// The adapter takes ownership of the store reference and will update it on mutations.
//
// Example:
//
//	store := NewUnifiedStore()
//	adapter := NewUnifiedStoreAdapter(store)
//	goal := db.Query(person, Fresh("name"), Fresh("age"))
//	stream := goal(ctx, adapter)
func NewUnifiedStoreAdapter(store *UnifiedStore) *UnifiedStoreAdapter {
	id := fmt.Sprintf("adapter-%d", atomic.AddInt64(&adapterIDCounter, 1))

	return &UnifiedStoreAdapter{
		store: store,
		id:    id,
	}
}

// AddConstraint adds a constraint to the underlying UnifiedStore.
// Implements ConstraintStore interface.
//
// Note: UnifiedStore uses interface{} for constraints (not typed Constraint),
// allowing both relational constraints and FD constraints. The hybrid solver's
// plugins determine how to handle each constraint type during propagation.
//
// Returns nil always - constraint violations are detected during propagation,
// not at constraint addition time. This matches UnifiedStore's batched
// constraint checking philosophy.
func (a *UnifiedStoreAdapter) AddConstraint(constraint Constraint) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// UnifiedStore.AddConstraint returns new store (immutable update)
	a.store = a.store.AddConstraint(constraint)

	// Constraint validity is checked during propagation, not here
	return nil
}

// AddBinding binds a variable to a term in the underlying UnifiedStore.
// Implements ConstraintStore interface.
//
// Returns error if the binding would violate constraints. In the UnifiedStore model,
// binding errors typically come from the hybrid solver during propagation, but
// we return any error from AddBinding for interface compliance.
func (a *UnifiedStoreAdapter) AddBinding(varID int64, term Term) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// UnifiedStore.AddBinding returns (newStore, error)
	newStore, err := a.store.AddBinding(varID, term)
	if err != nil {
		return fmt.Errorf("binding variable %d: %w", varID, err)
	}

	a.store = newStore
	return nil
}

// GetBinding retrieves the relational binding for a variable.
// Implements ConstraintStore interface.
//
// Returns nil if the variable is unbound. Thread-safe due to UnifiedStore immutability.
func (a *UnifiedStoreAdapter) GetBinding(varID int64) Term {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetBinding(varID)
}

// GetSubstitution returns a Substitution representing all relational bindings.
// Implements ConstraintStore interface.
//
// This bridges UnifiedStore to miniKanren's substitution-based APIs.
// The substitution is a snapshot at call time; subsequent mutations won't affect it.
func (a *UnifiedStoreAdapter) GetSubstitution() *Substitution {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetSubstitution()
}

// GetConstraints returns all active constraints in the underlying store.
// Implements ConstraintStore interface.
//
// Note: Returns []Constraint but UnifiedStore stores []interface{}.
// This assumes all constraints added implement the Constraint interface.
// If non-Constraint objects are added (e.g., FD-specific constraints),
// they'll be filtered out.
func (a *UnifiedStoreAdapter) GetConstraints() []Constraint {
	a.mu.RLock()
	defer a.mu.RUnlock()

	rawConstraints := a.store.GetConstraints()
	constraints := make([]Constraint, 0, len(rawConstraints))

	for _, rc := range rawConstraints {
		if c, ok := rc.(Constraint); ok {
			constraints = append(constraints, c)
		}
		// Silently skip non-Constraint objects (e.g., FD constraints)
		// This is expected in hybrid solving
	}

	return constraints
}

// Clone creates a deep copy of the adapter with an independent UnifiedStore.
// Implements ConstraintStore interface.
//
// The cloned adapter starts with a copy of the current store (via UnifiedStore.Clone),
// enabling parallel search where each branch has its own constraint evolution.
//
// Cloning is cheap (O(1)) due to UnifiedStore's copy-on-write semantics with
// structural sharing. Most data is shared until modified.
func (a *UnifiedStoreAdapter) Clone() ConstraintStore {
	a.mu.RLock()
	defer a.mu.RUnlock()

	clonedStore := a.store.Clone()
	return NewUnifiedStoreAdapter(clonedStore)
}

// String returns a human-readable representation for debugging.
// Implements ConstraintStore interface.
func (a *UnifiedStoreAdapter) String() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return fmt.Sprintf("UnifiedStoreAdapter{id=%s, store=%s}", a.id, a.store.String())
}

// UnifiedStore returns the underlying UnifiedStore for hybrid solver operations.
// This allows extracting the store for propagation with HybridSolver.
//
// Example usage pattern:
//
//	adapter := NewUnifiedStoreAdapter(store)
//	// ... use adapter with goals/pldb ...
//	hybridStore := adapter.UnifiedStore()
//	propagated, err := hybridSolver.Propagate(hybridStore)
//	if err == nil {
//	    adapter.SetUnifiedStore(propagated)
//	}
func (a *UnifiedStoreAdapter) UnifiedStore() *UnifiedStore {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store
}

// SetUnifiedStore updates the adapter's underlying store.
// Used after hybrid solver propagation to install the propagated store.
//
// This method should be used with care: it replaces the entire store,
// so any bindings/constraints added directly to the adapter (bypassing
// the hybrid solver) will be overwritten.
//
// Typical usage:
//
//	store := adapter.UnifiedStore()
//	propagated, err := hybridSolver.Propagate(store)
//	if err != nil {
//	    // Conflict detected, backtrack
//	    return
//	}
//	adapter.SetUnifiedStore(propagated)
func (a *UnifiedStoreAdapter) SetUnifiedStore(store *UnifiedStore) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.store = store
}

// GetDomain retrieves the FD domain for a variable from the underlying UnifiedStore.
// This is not part of the ConstraintStore interface but provides access to FD domains
// for hybrid solving scenarios.
//
// Returns nil if the variable has no FD domain (relational-only variable).
func (a *UnifiedStoreAdapter) GetDomain(varID int) Domain {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.GetDomain(varID)
}

// SetDomain updates the FD domain for a variable in the underlying UnifiedStore.
// This is not part of the ConstraintStore interface but provides FD domain updates
// for hybrid solving scenarios.
//
// Returns error if the domain is empty (conflict detected).
func (a *UnifiedStoreAdapter) SetDomain(varID int, domain Domain) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	newStore, err := a.store.SetDomain(varID, domain)
	if err != nil {
		return fmt.Errorf("setting domain for variable %d: %w", varID, err)
	}

	a.store = newStore
	return nil
}

// Depth returns the depth of the underlying store in the search tree.
// Used for heuristics and debugging. Not part of ConstraintStore interface.
func (a *UnifiedStoreAdapter) Depth() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.store.Depth()
}
