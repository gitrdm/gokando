// Package minikanren provides specialized reified constraints.
//
// This file implements EqualityReified, a constraint that links equality
// between two variables to a boolean variable with full bidirectional propagation.
package minikanren

import (
	"fmt"
)

// EqualityReified constrains a boolean variable to reflect equality between two variables.
//
// Given variables X, Y, and boolean B (domain {1,2} where 1=false, 2=true):
//   - B = 2 (true) ⟺ X = Y
//   - B = 1 (false) ⟺ X ≠ Y
//
// Bidirectional propagation:
//  1. X and Y become equal → remove 1 from B (set B=2)
//  2. X and Y proven unequal → remove 2 from B (set B=1)
//  3. B becomes 2 → enforce X = Y via domain intersection
//  4. B becomes 1 → remove intersection from both domains (enforce X ≠ Y)
//
// This provides proper reification semantics for equality, handling both
// "constraint must be true" and "constraint must be false" cases correctly.
//
// Implementation achieves arc-consistency through:
//   - When B=2: X.domain ← X.domain ∩ Y.domain (and vice versa)
//   - When B=1: for each value v: if v ∈ X.domain and Y.domain={v}, remove v from X
//   - Singleton detection: if X and Y are singletons, set B accordingly
//   - Disjoint detection: if X.domain ∩ Y.domain = ∅, set B=1
type EqualityReified struct {
	x       *FDVariable // First variable
	y       *FDVariable // Second variable
	boolVar *FDVariable // Boolean variable (domain {1,2})
}

// NewEqualityReified creates an equality-reified constraint.
//
// Parameters:
//   - x, y: variables whose equality is being reified
//   - boolVar: boolean variable with domain {1,2} (1=false, 2=true)
//
// Returns error if any parameter is nil.
func NewEqualityReified(x, y, boolVar *FDVariable) (*EqualityReified, error) {
	if x == nil {
		return nil, fmt.Errorf("NewEqualityReified: x cannot be nil")
	}
	if y == nil {
		return nil, fmt.Errorf("NewEqualityReified: y cannot be nil")
	}
	if boolVar == nil {
		return nil, fmt.Errorf("NewEqualityReified: boolVar cannot be nil")
	}

	return &EqualityReified{
		x:       x,
		y:       y,
		boolVar: boolVar,
	}, nil
}

// Variables returns the variables involved in this constraint.
// Implements ModelConstraint.
func (e *EqualityReified) Variables() []*FDVariable {
	return []*FDVariable{e.x, e.y, e.boolVar}
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (e *EqualityReified) Type() string {
	return "EqualityReified"
}

// String returns a human-readable representation.
// Implements ModelConstraint.
func (e *EqualityReified) String() string {
	return fmt.Sprintf("EqualityReified(X=%d, Y=%d, B=%d)", e.x.ID(), e.y.ID(), e.boolVar.ID())
}

// Propagate applies the equality-reified constraint's propagation.
// Implements PropagationConstraint.
func (e *EqualityReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("EqualityReified.Propagate: nil solver")
	}
	// Note: state can be nil (means use initial model domains)

	// Get current domains
	xDomain := solver.GetDomain(state, e.x.ID())
	if xDomain == nil || xDomain.Count() == 0 {
		return nil, fmt.Errorf("EqualityReified.Propagate: X variable %d has empty domain", e.x.ID())
	}

	yDomain := solver.GetDomain(state, e.y.ID())
	if yDomain == nil || yDomain.Count() == 0 {
		return nil, fmt.Errorf("EqualityReified.Propagate: Y variable %d has empty domain", e.y.ID())
	}

	boolDomain := solver.GetDomain(state, e.boolVar.ID())
	if boolDomain == nil || boolDomain.Count() == 0 {
		return nil, fmt.Errorf("EqualityReified.Propagate: boolean variable %d has empty domain", e.boolVar.ID())
	}

	// Validate boolean domain is subset of {1,2}
	hasOne := boolDomain.Has(1) // false
	hasTwo := boolDomain.Has(2) // true

	if boolDomain.Count() > 2 || (!hasOne && !hasTwo) {
		return nil, fmt.Errorf("EqualityReified.Propagate: boolean variable %d domain must be subset of {1,2}, got %s",
			e.boolVar.ID(), boolDomain.String())
	}

	currentState := state

	// Check if domains are disjoint (X and Y cannot be equal)
	intersection := xDomain.Intersect(yDomain)
	if intersection.Count() == 0 {
		// X and Y have no common values → they cannot be equal → B must be 1 (false)
		if hasTwo {
			newBoolDomain := boolDomain.Remove(2)
			currentState, _ = solver.SetDomain(currentState, e.boolVar.ID(), newBoolDomain)
			boolDomain = newBoolDomain
			hasTwo = false
		}
	}

	// Check if both X and Y are singletons with the same value
	if xDomain.IsSingleton() && yDomain.IsSingleton() {
		if xDomain.SingletonValue() == yDomain.SingletonValue() {
			// X = Y → B must be 2 (true)
			if hasOne {
				newBoolDomain := boolDomain.Remove(1)
				currentState, _ = solver.SetDomain(currentState, e.boolVar.ID(), newBoolDomain)
				boolDomain = newBoolDomain
				hasOne = false
			}
		} else {
			// X ≠ Y → B must be 1 (false)
			if hasTwo {
				newBoolDomain := boolDomain.Remove(2)
				currentState, _ = solver.SetDomain(currentState, e.boolVar.ID(), newBoolDomain)
				boolDomain = newBoolDomain
				hasTwo = false
			}
		}
	}

	// Update based on current boolean domain values
	// Re-check hasOne and hasTwo after updates
	boolDomain = solver.GetDomain(currentState, e.boolVar.ID())
	hasOne = boolDomain.Has(1)
	hasTwo = boolDomain.Has(2)

	// Case 1: B = 2 (true) - enforce X = Y
	if hasTwo && !hasOne {
		// Prune X and Y to their intersection
		newXDomain := xDomain.Intersect(yDomain)
		newYDomain := yDomain.Intersect(xDomain)

		if newXDomain.Count() == 0 || newYDomain.Count() == 0 {
			return nil, fmt.Errorf("EqualityReified.Propagate: B=2 requires X=Y but domains are disjoint")
		}

		if !newXDomain.Equal(xDomain) {
			currentState, _ = solver.SetDomain(currentState, e.x.ID(), newXDomain)
			xDomain = newXDomain
		}

		if !newYDomain.Equal(yDomain) {
			currentState, _ = solver.SetDomain(currentState, e.y.ID(), newYDomain)
			yDomain = newYDomain
		}
	}

	// Case 2: B = 1 (false) - enforce X ≠ Y
	if hasOne && !hasTwo {
		// If X is singleton, remove its value from Y
		if xDomain.IsSingleton() {
			xVal := xDomain.SingletonValue()
			if yDomain.Has(xVal) {
				newYDomain := yDomain.Remove(xVal)
				if newYDomain.Count() == 0 {
					return nil, fmt.Errorf("EqualityReified.Propagate: B=1 requires X≠Y but Y would be empty")
				}
				currentState, _ = solver.SetDomain(currentState, e.y.ID(), newYDomain)
				yDomain = newYDomain
			}
		}

		// If Y is singleton, remove its value from X
		if yDomain.IsSingleton() {
			yVal := yDomain.SingletonValue()
			if xDomain.Has(yVal) {
				newXDomain := xDomain.Remove(yVal)
				if newXDomain.Count() == 0 {
					return nil, fmt.Errorf("EqualityReified.Propagate: B=1 requires X≠Y but X would be empty")
				}
				currentState, _ = solver.SetDomain(currentState, e.x.ID(), newXDomain)
				xDomain = newXDomain
			}
		}
	}

	// Case 3: B is {1,2} (unknown) - no additional propagation beyond what we did above

	return currentState, nil
}
