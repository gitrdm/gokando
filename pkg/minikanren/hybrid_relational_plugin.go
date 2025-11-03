// Package minikanren provides plugin implementations for the hybrid solver.
// This file implements the Relational plugin that wraps the existing
// miniKanren constraint system (disequality, type constraints, etc.).
package minikanren

import (
	"fmt"
)

// RelationalPlugin wraps the miniKanren relational constraint system to work
// within the hybrid framework. It handles Constraints (from constraint_types.go)
// such as disequality, absence, and type constraints.
//
// The RelationalPlugin checks constraints against relational bindings in the
// UnifiedStore. Unlike FD propagation which prunes domains, relational constraint
// checking is typically a pass/fail test: either the constraint is satisfied,
// violated, or pending (waiting for more bindings).
//
// During propagation, the RelationalPlugin:
//  1. Extracts relational bindings from the UnifiedStore
//  2. Checks each Constraint against those bindings
//  3. Returns error if any constraint is violated
//  4. Returns original store if all constraints are satisfied or pending
//
// The relational plugin doesn't typically modify the store (no pruning),
// it just validates that current bindings don't violate constraints.
// However, if FD domains narrow variables to singletons, those singleton
// values can be promoted to relational bindings, enabling cross-solver
// propagation.
type RelationalPlugin struct {
	// No state needed - relational constraints are checked against the store
}

// NewRelationalPlugin creates a new relational constraint plugin.
func NewRelationalPlugin() *RelationalPlugin {
	return &RelationalPlugin{}
}

// Name returns the plugin identifier.
// Implements SolverPlugin.
func (rp *RelationalPlugin) Name() string {
	return "Relational"
}

// CanHandle returns true if the constraint is a relational constraint.
// Implements SolverPlugin.
func (rp *RelationalPlugin) CanHandle(constraint interface{}) bool {
	// Check if it's a Constraint (from constraint_types.go)
	_, ok := constraint.(Constraint)
	return ok
}

// Propagate checks all relational constraints in the store.
// Returns error if any constraint is violated, otherwise returns the store unchanged.
// Implements SolverPlugin.
func (rp *RelationalPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error) {
	// Get all bindings for constraint checking
	bindings := store.getAllBindings()

	// Check each constraint
	for _, c := range store.GetConstraints() {
		// Only process constraints we can handle
		constraint, ok := c.(Constraint)
		if !ok {
			continue
		}

		// Check the constraint
		result := constraint.Check(bindings)

		switch result {
		case ConstraintViolated:
			return nil, fmt.Errorf("relational constraint violated: %s", constraint.String())

		case ConstraintSatisfied:
			// Constraint is satisfied, continue

		case ConstraintPending:
			// Constraint can't be fully evaluated yet, but not violated
			// This is normal - constraint will be re-checked as more variables bind
		}
	}

	// Step 1: Check for FD singleton values that can become relational bindings
	// This enables cross-solver propagation: FD narrows domain → relational binding
	newStore, err := rp.promoteSingletons(store)
	if err != nil {
		return nil, err
	}

	// Step 2: Propagate relational bindings back to FD domains
	// This enables: relational binding x=5 → FD domain pruned to {5}
	newStore, err = rp.propagateBindingsToDomains(newStore)
	if err != nil {
		return nil, err
	}

	return newStore, nil
}

// propagateBindingsToDomains synchronizes relational bindings to FD domains.
// When a variable is bound relationally (x=5), and it has an FD domain,
// we prune the FD domain to contain only the bound value.
//
// This enables bidirectional propagation:
//   - Relational says x=5 → FD domain becomes {5}
//   - Relational says x≠3 → 3 is removed from FD domain (future enhancement)
//
// This ensures attributed variables (with both bindings and domains) remain
// consistent across solver boundaries.
func (rp *RelationalPlugin) propagateBindingsToDomains(store *UnifiedStore) (*UnifiedStore, error) {
	bindings := store.getAllBindings()
	domains := store.getAllDomains()
	newStore := store

	for varID, binding := range bindings {
		// Check if this variable also has an FD domain
		// Both bindings and domains use int64/int keys
		domain, hasDomain := domains[int(varID)]
		if !hasDomain {
			continue
		}

		// Extract the numeric value from the binding (if it's a number)
		atom, isAtom := binding.(*Atom)
		if !isAtom {
			// Binding is not an atom (could be pair, var, etc.)
			// Can't map to FD domain value - this is a conflict
			return nil, fmt.Errorf("variable %d has non-atomic binding %s with FD domain - conflict", varID, binding.String())
		}

		value, isInt := atom.Value().(int)
		if !isInt {
			// Binding is atomic but not an integer
			// Can't map to FD domain - this is a type conflict
			return nil, fmt.Errorf("variable %d bound to non-integer %v with FD domain - type mismatch", varID, atom.Value())
		}

		// Check if the bound value is in the FD domain
		if !domain.Has(value) {
			return nil, fmt.Errorf("variable %d bound to %d but FD domain doesn't contain %d - conflict", varID, value, value)
		}

		// Only prune if domain is not already a singleton with the correct value
		if domain.IsSingleton() && domain.SingletonValue() == value {
			// Already synchronized, no change needed
			continue
		}

		// Prune the FD domain to singleton {value}
		singletonDomain := NewBitSetDomainFromValues(domain.Max(), []int{value})
		var err error
		newStore, err = newStore.SetDomain(int(varID), singletonDomain)
		if err != nil {
			return nil, fmt.Errorf("failed to prune domain for bound variable %d: %w", varID, err)
		}
	}

	return newStore, nil
}

// promoteSingletons checks FD domains for singleton values and promotes them
// to relational bindings. This enables the relational solver to use information
// from FD propagation.
//
// Example: If FD propagation narrows X's domain to {5}, we can add the
// relational binding X=5, allowing relational constraints to fire.
func (rp *RelationalPlugin) promoteSingletons(store *UnifiedStore) (*UnifiedStore, error) {
	domains := store.getAllDomains()
	newStore := store

	for varID, domain := range domains {
		// Skip if already bound relationally
		if newStore.GetBinding(int64(varID)) != nil {
			continue
		}

		// Check if domain is singleton
		if domain.IsSingleton() {
			value := domain.SingletonValue()

			// Create relational binding for the singleton value
			term := NewAtom(value)
			var err error
			newStore, err = newStore.AddBinding(int64(varID), term)
			if err != nil {
				return nil, fmt.Errorf("failed to promote singleton %d=%d: %w", varID, value, err)
			}
		}
	}

	return newStore, nil
}
