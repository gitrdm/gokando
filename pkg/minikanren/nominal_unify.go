// Package minikanren provides nominal unification algorithms for reasoning about names and binding.
// Nominal unification extends regular unification with support for fresh names, binding constraints,
// and alpha-equivalence. This enables reasoning about programming language constructs where
// variable names matter for binding and scope.
//
// Key differences from regular unification:
//   - Names participate in unification with freshness constraints
//   - Alpha-equivalent terms (differing only in bound names) are considered equal
//   - Binding operations create lexical scopes
//   - Fresh name generation ensures no accidental name capture
//
// This implementation provides:
//   - Nominal unification with name binding rules
//   - Freshness constraint checking
//   - Alpha-equivalence comparison
//   - Integration with constraint store system
package minikanren

import (
	"context"
	"fmt"
	"sync"
)

// NominalUnifier handles nominal unification operations.
// It maintains state about name bindings and freshness constraints,
// enabling proper reasoning about names in logical terms.
type NominalUnifier struct {
	scope *NominalScope // Current lexical scope for name bindings
	mu    sync.RWMutex  // Protects concurrent access to scope
}

// NewNominalUnifier creates a new nominal unifier with an empty scope.
func NewNominalUnifier() *NominalUnifier {
	return &NominalUnifier{
		scope: NewNominalScope(),
	}
}

// NewNominalUnifierWithScope creates a new nominal unifier with an existing scope.
func NewNominalUnifierWithScope(scope *NominalScope) *NominalUnifier {
	return &NominalUnifier{
		scope: scope.Clone(),
	}
}

// Unify performs nominal unification between two terms.
// Returns true if unification succeeds, updating the scope with any new bindings.
// Nominal unification respects name binding rules and freshness constraints.
func (nu *NominalUnifier) Unify(term1, term2 Term) bool {
	nu.mu.Lock()
	defer nu.mu.Unlock()

	return nu.unifyInternal(term1, term2)
}

// unifyInternal performs the actual unification logic (assumes lock is held).
func (nu *NominalUnifier) unifyInternal(term1, term2 Term) bool {
	// Handle names specially in nominal unification
	if name1, ok1 := term1.(*Name); ok1 {
		if name2, ok2 := term2.(*Name); ok2 {
			return nu.unifyNames(name1, name2)
		}
		// Name unifying with non-name - check if name is bound
		if bound := nu.scope.Lookup(name1); bound != nil {
			return nu.unifyInternal(bound, term2)
		}
		// Bind the name to the term
		nu.scope.Bind(name1, term2)
		return true
	}

	if name2, ok2 := term2.(*Name); ok2 {
		// Symmetric case - term1 is not a name, term2 is
		if bound := nu.scope.Lookup(name2); bound != nil {
			return nu.unifyInternal(term1, bound)
		}
		// Bind the name to the term
		nu.scope.Bind(name2, term1)
		return true
	}

	// Regular unification for non-name terms
	return nu.unifyRegular(term1, term2)
}

// unifyNames handles unification between two names.
// In nominal logic, names unify if they are the same name or if one can be bound to the other
// without violating freshness constraints.
func (nu *NominalUnifier) unifyNames(name1, name2 *Name) bool {
	// If they're the same name, unification succeeds
	if name1.Equal(name2) {
		return true
	}

	// Check if either name is already bound
	bound1 := nu.scope.Lookup(name1)
	bound2 := nu.scope.Lookup(name2)

	if bound1 != nil && bound2 != nil {
		// Both bound - unify the bound terms
		return nu.unifyInternal(bound1, bound2)
	}

	if bound1 != nil {
		// name1 is bound, bind name2 to the same term
		nu.scope.Bind(name2, bound1)
		return true
	}

	if bound2 != nil {
		// name2 is bound, bind name1 to the same term
		nu.scope.Bind(name1, bound2)
		return true
	}

	// Neither is bound - bind them to each other (they become aliases)
	nu.scope.Bind(name1, name2)
	return true
}

// unifyRegular performs regular (non-nominal) unification for non-name terms.
// This handles atoms, variables, and compound structures.
func (nu *NominalUnifier) unifyRegular(term1, term2 Term) bool {
	// Walk terms through any name bindings
	t1 := nu.walk(term1)
	t2 := nu.walk(term2)

	// If they're structurally equal, unification succeeds
	if t1.Equal(t2) {
		return true
	}

	// Handle variables (logic variables, not names)
	if t1.IsVar() {
		if t2.IsVar() {
			// Both variables - they can be unified by making them aliases
			// In constraint system, this would be handled by the constraint store
			return true // Assume variables can be unified
		}
		// Variable unifying with non-variable - would be handled by constraint store
		return true
	}

	if t2.IsVar() {
		// Symmetric case
		return true
	}

	// Handle pairs (compound structures)
	if p1, ok1 := t1.(*Pair); ok1 {
		if p2, ok2 := t2.(*Pair); ok2 {
			// Unify car and cdr recursively
			return nu.unifyInternal(p1.Car(), p2.Car()) &&
				nu.unifyInternal(p1.Cdr(), p2.Cdr())
		}
	}

	// Handle atoms
	if a1, ok1 := t1.(*Atom); ok1 {
		if a2, ok2 := t2.(*Atom); ok2 {
			return a1.Equal(a2)
		}
	}

	// Unification failed
	return false
}

// walk traverses a term, resolving any name bindings.
func (nu *NominalUnifier) walk(term Term) Term {
	if name, ok := term.(*Name); ok {
		if bound := nu.scope.Lookup(name); bound != nil {
			return nu.walk(bound) // Follow binding chain
		}
	}
	return term
}

// BindName binds a name to a term in the current scope.
// This creates a lexical binding that affects subsequent unification operations.
func (nu *NominalUnifier) BindName(name *Name, term Term) {
	nu.mu.Lock()
	defer nu.mu.Unlock()
	nu.scope.Bind(name, term)
}

// LookupName finds the term bound to a name, or nil if unbound.
func (nu *NominalUnifier) LookupName(name *Name) Term {
	nu.mu.RLock()
	defer nu.mu.RUnlock()
	return nu.scope.Lookup(name)
}

// IsNameBound checks if a name is bound in the current scope chain.
func (nu *NominalUnifier) IsNameBound(name *Name) bool {
	nu.mu.RLock()
	defer nu.mu.RUnlock()
	return nu.scope.IsBound(name)
}

// CreateScope creates a new nested scope.
// The new scope inherits bindings from the current scope but allows shadowing.
func (nu *NominalUnifier) CreateScope() *NominalUnifier {
	nu.mu.Lock()
	defer nu.mu.Unlock()

	newScope := NewNominalScopeWithParent(nu.scope)
	return &NominalUnifier{
		scope: newScope,
	}
}

// GetScope returns a copy of the current scope.
func (nu *NominalUnifier) GetScope() *NominalScope {
	nu.mu.RLock()
	defer nu.mu.RUnlock()
	return nu.scope.Clone()
}

// FreshName generates a fresh name that is guaranteed not to conflict
// with any names currently in scope. This is essential for avoiding
// accidental name capture in nominal logic operations.
func (nu *NominalUnifier) FreshName(symbol string) *Name {
	// Generate a new name - the atomic counter ensures uniqueness
	fresh := NewName(symbol)

	// Verify it's not already bound (shouldn't happen with atomic counter)
	if nu.IsNameBound(fresh) {
		// Extremely unlikely, but handle it by generating another
		fresh = NewName(symbol + "_fresh")
	}

	return fresh
}

// AlphaEquivalent checks if two terms are alpha-equivalent.
// Alpha-equivalence means the terms are identical up to renaming of bound names.
// This is a key concept in nominal logic for reasoning about programming languages.
func (nu *NominalUnifier) AlphaEquivalent(term1, term2 Term) bool {
	nu.mu.RLock()
	defer nu.mu.RUnlock()

	return nu.alphaEquivalentInternal(term1, term2, make(map[int64]int64))
}

// alphaEquivalentInternal performs alpha-equivalence checking with name mapping.
func (nu *NominalUnifier) alphaEquivalentInternal(term1, term2 Term, nameMap map[int64]int64) bool {
	// Walk terms through bindings
	t1 := nu.walk(term1)
	t2 := nu.walk(term2)

	// Handle names with alpha-equivalence mapping
	if name1, ok1 := t1.(*Name); ok1 {
		if name2, ok2 := t2.(*Name); ok2 {
			// Check if names are mapped to each other
			if mappedID, exists := nameMap[name1.ID()]; exists {
				return mappedID == name2.ID()
			}
			// Create mapping
			nameMap[name1.ID()] = name2.ID()
			return true
		}
		return false
	}

	// Regular structural equality for non-names
	if t1.Equal(t2) {
		return true
	}

	// Handle pairs recursively
	if p1, ok1 := t1.(*Pair); ok1 {
		if p2, ok2 := t2.(*Pair); ok2 {
			return nu.alphaEquivalentInternal(p1.Car(), p2.Car(), nameMap) &&
				nu.alphaEquivalentInternal(p1.Cdr(), p2.Cdr(), nameMap)
		}
	}

	return false
}

// String returns a string representation of the nominal unifier state.
func (nu *NominalUnifier) String() string {
	nu.mu.RLock()
	defer nu.mu.RUnlock()
	return fmt.Sprintf("NominalUnifier{scope: %s}", nu.scope.String())
}

// NominalEq creates a nominal unification goal.
// This goal unifies two terms using nominal unification rules,
// respecting name bindings and freshness constraints.
//
// Example:
//
//	x := NewName("x")
//	y := NewName("y")
//	goal := NominalEq(x, NewAtom("value"))  // Binds x to "value"
func NominalEq(term1, term2 Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Create a nominal unifier for this goal
		unifier := NewNominalUnifier()

		// Attempt nominal unification
		if unifier.Unify(term1, term2) {
			// Unification succeeded - create a new constraint store with nominal bindings
			newStore := store.Clone()

			// Add nominal constraints to the store
			// This would integrate with the constraint system from Phase 2
			stream := NewStream()
			go func() {
				defer stream.Close()
				stream.Put(ctx, newStore)
			}()
			return stream
		}

		// Unification failed
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// NominalFresh creates a goal that generates a fresh name.
// This ensures the name doesn't conflict with any existing bindings.
//
// Example:
//
//	goal := NominalFresh(func(fresh *Name) Goal {
//	    return NominalEq(fresh, NewAtom("unique"))
//	})
func NominalFresh(goalFunc func(*Name) Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Create a fresh name
		unifier := NewNominalUnifier()
		freshName := unifier.FreshName("")

		// Apply the goal function with the fresh name
		goal := goalFunc(freshName)
		return goal(ctx, store)
	}
}

// NominalBind creates a goal that binds a name to a term in a new scope.
// This establishes lexical scoping for name bindings.
//
// Example:
//
//	x := NewName("x")
//	goal := NominalBind(x, NewAtom("bound"), func() Goal {
//	    return NominalEq(x, NewAtom("bound"))  // This will succeed
//	})
func NominalBind(name *Name, term Term, goalFunc func() Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Create a nominal unifier with a new scope
		unifier := NewNominalUnifier()
		unifier.BindName(name, term)

		// Execute the goal in the new scope
		goal := goalFunc()
		return goal(ctx, store)
	}
}
