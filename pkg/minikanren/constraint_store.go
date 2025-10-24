// Package minikanren provides constraint system infrastructure for order-independent
// constraint logic programming. This file defines the core interfaces and types
// for managing constraints in a hybrid local/global architecture.
//
// The constraint system uses a two-tier approach:
//   - Local constraints: managed within individual goal contexts for fast checking
//   - Global constraints: coordinated across contexts when constraints span multiple stores
//
// This design provides order-independent constraint semantics while maintaining
// high performance for the common case of locally-scoped constraints.
package minikanren

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ConstraintResult represents the outcome of evaluating a constraint.
// Constraints can be satisfied (no violation), violated (goal should fail),
// or pending (waiting for more variable bindings).
type ConstraintResult int

const (
	// ConstraintSatisfied indicates the constraint is currently satisfied
	// and does not prevent the goal from succeeding.
	ConstraintSatisfied ConstraintResult = iota

	// ConstraintViolated indicates the constraint has been violated
	// and the goal should fail immediately.
	ConstraintViolated

	// ConstraintPending indicates the constraint cannot be fully evaluated
	// yet due to unbound variables, but is not currently violated.
	ConstraintPending
)

// String returns a human-readable representation of the constraint result.
func (cr ConstraintResult) String() string {
	switch cr {
	case ConstraintSatisfied:
		return "satisfied"
	case ConstraintViolated:
		return "violated"
	case ConstraintPending:
		return "pending"
	default:
		return "unknown"
	}
}

// Constraint represents a logical constraint that can be checked against
// variable bindings. Constraints are the core abstraction that enables
// order-independent constraint logic programming.
//
// Constraints must be thread-safe as they may be checked concurrently
// during parallel goal evaluation.
type Constraint interface {
	// ID returns a unique identifier for this constraint instance.
	// Used for tracking and debugging constraint violations.
	ID() string

	// IsLocal returns true if this constraint can be evaluated purely
	// within a local constraint store, false if it requires global coordination.
	// Local constraints have better performance characteristics.
	IsLocal() bool

	// Variables returns the logic variables that this constraint depends on.
	// Used to determine when the constraint needs to be re-evaluated.
	Variables() []*Var

	// Check evaluates the constraint against the current variable bindings
	// in the given constraint store. Must be thread-safe.
	Check(bindings map[int64]Term) ConstraintResult

	// String returns a human-readable representation of the constraint
	// for debugging and error reporting.
	String() string

	// Clone creates a deep copy of the constraint for use in parallel
	// execution contexts where constraint stores may be forked.
	Clone() Constraint
}

// ConstraintStore represents a collection of constraints and variable bindings.
// This interface abstracts over both local and global constraint storage.
type ConstraintStore interface {
	// AddConstraint adds a new constraint to the store.
	// Returns an error if the constraint immediately violates existing bindings.
	AddConstraint(constraint Constraint) error

	// AddBinding attempts to bind a variable to a term.
	// Returns an error if the binding would violate any constraints.
	AddBinding(varID int64, term Term) error

	// GetBinding retrieves the current binding for a variable.
	// Returns nil if the variable is unbound.
	GetBinding(varID int64) Term

	// GetSubstitution returns a substitution representing all current bindings.
	GetSubstitution() *Substitution

	// GetConstraints returns all constraints currently in the store.
	GetConstraints() []Constraint

	// Clone creates a deep copy of the constraint store for parallel execution.
	Clone() ConstraintStore

	// String returns a human-readable representation for debugging.
	String() string
}

// ConstraintEvent represents a notification about constraint-related activities.
// Used for coordinating between local stores and the global constraint bus.
type ConstraintEvent struct {
	// Type indicates the kind of event (constraint added, variable bound, etc.)
	Type ConstraintEventType

	// StoreID identifies which local constraint store generated this event
	StoreID string

	// VarID is the variable ID involved in the event (for binding events)
	VarID int64

	// Term is the term being bound to the variable (for binding events)
	Term Term

	// Constraint is the constraint involved in the event (for constraint events)
	Constraint Constraint

	// Timestamp helps with debugging and event ordering
	Timestamp int64
}

// ConstraintEventType categorizes different kinds of constraint events
// for efficient processing by the global constraint bus.
type ConstraintEventType int

const (
	// ConstraintAdded indicates a new constraint was added to a local store
	ConstraintAdded ConstraintEventType = iota

	// VariableBound indicates a variable was bound to a term
	VariableBound

	// ConstraintViolationDetected indicates a constraint violation was detected
	ConstraintViolationDetected

	// StoreCloned indicates a constraint store was cloned (for parallel execution)
	StoreCloned
)

// String returns a human-readable representation of the constraint event type.
func (cet ConstraintEventType) String() string {
	switch cet {
	case ConstraintAdded:
		return "constraint-added"
	case VariableBound:
		return "variable-bound"
	case ConstraintViolationDetected:
		return "constraint-violated"
	case StoreCloned:
		return "store-cloned"
	default:
		return "unknown-event"
	}
}

// LocalConstraintStore interface defines the operations needed by
// the GlobalConstraintBus to coordinate with local stores.
type LocalConstraintStore interface {
	ID() string
	getAllBindings() map[int64]Term
}

// GlobalConstraintBus coordinates constraint checking across multiple
// local constraint stores. It handles cross-store constraints and provides
// a coordination point for complex constraint interactions.
//
// The bus is designed to minimize coordination overhead - most constraints
// should be local and not require global coordination.
type GlobalConstraintBus struct {
	// crossStoreConstraints holds constraints that span multiple stores
	crossStoreConstraints map[string]Constraint

	// storeRegistry tracks all active local constraint stores
	storeRegistry map[string]LocalConstraintStore

	// events is the channel for constraint events requiring global coordination
	events chan ConstraintEvent

	// eventCounter provides unique timestamps for events
	eventCounter int64

	// mu protects concurrent access to bus state
	mu sync.RWMutex

	// shutdown indicates if the bus is shutting down
	shutdown bool

	// shutdownCh is closed when the bus shuts down
	shutdownCh chan struct{}
}

// NewGlobalConstraintBus creates a new global constraint bus for coordinating
// constraint checking across multiple local stores.
func NewGlobalConstraintBus() *GlobalConstraintBus {
	bus := &GlobalConstraintBus{
		crossStoreConstraints: make(map[string]Constraint),
		storeRegistry:         make(map[string]LocalConstraintStore),
		events:                make(chan ConstraintEvent, 1000), // Buffered for performance
		shutdownCh:            make(chan struct{}),
	}

	// Start the event processing goroutine
	go bus.processEvents()

	return bus
}

// RegisterStore adds a local constraint store to the global registry.
// This enables the bus to coordinate constraints across the store.
func (gcb *GlobalConstraintBus) RegisterStore(store LocalConstraintStore) error {
	gcb.mu.Lock()
	defer gcb.mu.Unlock()

	if gcb.shutdown {
		return fmt.Errorf("constraint bus is shutdown")
	}

	gcb.storeRegistry[store.ID()] = store
	return nil
}

// UnregisterStore removes a local constraint store from the global registry.
// Should be called when a store is no longer needed to prevent memory leaks.
func (gcb *GlobalConstraintBus) UnregisterStore(storeID string) {
	gcb.mu.Lock()
	defer gcb.mu.Unlock()

	delete(gcb.storeRegistry, storeID)
}

// AddCrossStoreConstraint registers a constraint that requires global coordination.
// Such constraints are checked whenever any relevant variable is bound in any store.
func (gcb *GlobalConstraintBus) AddCrossStoreConstraint(constraint Constraint) error {
	gcb.mu.Lock()
	defer gcb.mu.Unlock()

	if gcb.shutdown {
		return fmt.Errorf("constraint bus is shutdown")
	}

	gcb.crossStoreConstraints[constraint.ID()] = constraint

	// Notify all stores about the new cross-store constraint
	event := ConstraintEvent{
		Type:       ConstraintAdded,
		Constraint: constraint,
		Timestamp:  atomic.AddInt64(&gcb.eventCounter, 1),
	}

	select {
	case gcb.events <- event:
		return nil
	default:
		return fmt.Errorf("constraint bus event queue full")
	}
}

// CoordinateBinding attempts to bind a variable across all relevant stores.
// This is used when a binding might affect cross-store constraints.
func (gcb *GlobalConstraintBus) CoordinateBinding(varID int64, term Term, originStoreID string) error {
	gcb.mu.RLock()
	defer gcb.mu.RUnlock()

	if gcb.shutdown {
		return fmt.Errorf("constraint bus is shutdown")
	}

	// Check if any cross-store constraints would be violated by this binding
	for _, constraint := range gcb.crossStoreConstraints {
		for _, constraintVar := range constraint.Variables() {
			if constraintVar.id == varID {
				// This binding affects a cross-store constraint
				// We need to check it against the combined state of all stores
				if !gcb.wouldBindingViolateConstraint(constraint, varID, term) {
					continue
				}
				return fmt.Errorf("binding would violate cross-store constraint %s", constraint.ID())
			}
		}
	}

	return nil
}

// Shutdown gracefully shuts down the global constraint bus.
// Should be called when constraint processing is complete.
func (gcb *GlobalConstraintBus) Shutdown() {
	gcb.mu.Lock()
	defer gcb.mu.Unlock()

	if gcb.shutdown {
		return
	}

	gcb.shutdown = true
	close(gcb.shutdownCh)
	close(gcb.events)
}

// processEvents handles constraint events in a dedicated goroutine.
// This provides asynchronous processing of cross-store constraint coordination.
func (gcb *GlobalConstraintBus) processEvents() {
	for {
		select {
		case event, ok := <-gcb.events:
			if !ok {
				// Events channel closed, shutdown
				return
			}

			// Process the event based on its type
			switch event.Type {
			case ConstraintAdded:
				gcb.handleConstraintAdded(event)
			case VariableBound:
				gcb.handleVariableBound(event)
			case ConstraintViolationDetected:
				gcb.handleConstraintViolated(event)
			case StoreCloned:
				gcb.handleStoreCloned(event)
			}

		case <-gcb.shutdownCh:
			return
		}
	}
}

// wouldBindingViolateConstraint checks if a proposed variable binding
// would violate a cross-store constraint by examining the combined state
// of all registered stores.
func (gcb *GlobalConstraintBus) wouldBindingViolateConstraint(constraint Constraint, varID int64, term Term) bool {
	// Create a combined binding map from all stores
	combinedBindings := make(map[int64]Term)

	// Collect bindings from all registered stores
	for _, store := range gcb.storeRegistry {
		storeBindings := store.getAllBindings()
		for id, binding := range storeBindings {
			combinedBindings[id] = binding
		}
	}

	// Add the proposed binding
	combinedBindings[varID] = term

	// Check the constraint against the combined bindings
	result := constraint.Check(combinedBindings)
	return result == ConstraintViolated
}

// handleConstraintAdded processes events when new constraints are added.
func (gcb *GlobalConstraintBus) handleConstraintAdded(event ConstraintEvent) {
	// Currently just logging - could extend for more sophisticated handling
	// In a production system, this might trigger constraint propagation
}

// handleVariableBound processes events when variables are bound.
func (gcb *GlobalConstraintBus) handleVariableBound(event ConstraintEvent) {
	// Check if any cross-store constraints are affected
	gcb.mu.RLock()
	defer gcb.mu.RUnlock()

	for _, constraint := range gcb.crossStoreConstraints {
		for _, constraintVar := range constraint.Variables() {
			if constraintVar.id == event.VarID {
				// This binding affects a cross-store constraint
				// In a full implementation, we might need to propagate this
				// to other stores or trigger additional constraint checking
				break
			}
		}
	}
}

// handleConstraintViolated processes constraint violation events.
func (gcb *GlobalConstraintBus) handleConstraintViolated(event ConstraintEvent) {
	// In a production system, this might trigger rollback or error reporting
}

// handleStoreCloned processes store cloning events for parallel execution.
func (gcb *GlobalConstraintBus) handleStoreCloned(event ConstraintEvent) {
	// Track cloned stores for proper resource management
}
