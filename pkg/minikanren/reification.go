// Package minikanren provides reification support for constraint programming.
//
// Reification allows the truth value of a constraint to be reflected as a boolean
// variable using 1-indexed domains: {1 = false, 2 = true}. This enables:
//   - Conditional constraints: "if X > 5 then Y = 10"
//   - Counting: "count how many variables equal a value"
//   - Soft constraints: "maximize constraints satisfied"
//   - Logical combinations: AND, OR, NOT over constraints
//
// Reification is bidirectional:
//   - Constraint → Boolean: When constraint becomes true/false, prune boolean domain
//   - Boolean → Constraint: When boolean is bound, enforce or disable constraint
//
// The reification architecture follows these principles:
//   - ReifiedConstraint wraps any PropagationConstraint
//   - Boolean variable must have domain subset of {1,2} (1=false, 2=true)
//   - Maintains copy-on-write semantics for parallel search
//   - Integrates seamlessly with existing constraint propagation
package minikanren

import (
	"fmt"
)

// ReifiedConstraint wraps a PropagationConstraint with a boolean variable.
//
// The boolean variable's value reflects the constraint's satisfaction:
//   - 2 (true): constraint must be satisfied
//   - 1 (false): constraint must be violated
//   - {1,2}: constraint satisfaction is unknown
//
// Note: We use 1=false, 2=true because BitSetDomain is 1-indexed (values >= 1).
//
// Bidirectional propagation:
//  1. When constraint is determined satisfied → set boolean to 2
//  2. When constraint is determined violated → set boolean to 1
//  3. When boolean = 2 → enforce constraint (propagate normally)
//  4. When boolean = 1 → ensure constraint is violated (complex, often via search)
//
// For simplicity, this implementation focuses on cases 1–3. Case 4 (forcing
// a constraint to be false) is challenging and often requires specialized
// negation logic per constraint type. We handle it by:
//   - If boolean is bound to 1 (false), we skip constraint propagation
//   - The search will naturally find assignments that violate the constraint
//
// This is sound but may be weaker than full constraint negation. For many
// use cases (including Count built via equality reification), this is sufficient.
type ReifiedConstraint struct {
	constraint PropagationConstraint // Underlying constraint to reify
	boolVar    *FDVariable           // Boolean variable (domain must be {0,1})
}

// NewReifiedConstraint creates a reified constraint.
//
// Parameters:
//   - constraint: the constraint to reify (must not be nil)
//   - boolVar: boolean variable with domain subset of {1,2} reflecting truth value
//     (1=false, 2=true, per BitSetDomain 1-indexing)
//
// Returns error if:
//   - constraint is nil
//   - boolVar is nil
//   - boolVar's domain is not a subset of {1,2}
func NewReifiedConstraint(constraint PropagationConstraint, boolVar *FDVariable) (*ReifiedConstraint, error) {
	if constraint == nil {
		return nil, fmt.Errorf("NewReifiedConstraint: constraint cannot be nil")
	}
	if boolVar == nil {
		return nil, fmt.Errorf("NewReifiedConstraint: boolVar cannot be nil")
	}

	// Note: We don't validate the bool domain here; propagation checks it

	return &ReifiedConstraint{
		constraint: constraint,
		boolVar:    boolVar,
	}, nil
}

// Variables returns all variables involved in this reified constraint.
// Includes both the constraint's variables and the boolean variable.
// Implements ModelConstraint.
func (r *ReifiedConstraint) Variables() []*FDVariable {
	constraintVars := r.constraint.Variables()
	// Boolean variable must be included
	allVars := make([]*FDVariable, 0, len(constraintVars)+1)
	allVars = append(allVars, constraintVars...)

	// Only add boolVar if not already in constraint variables
	found := false
	for _, v := range constraintVars {
		if v.ID() == r.boolVar.ID() {
			found = true
			break
		}
	}
	if !found {
		allVars = append(allVars, r.boolVar)
	}

	return allVars
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (r *ReifiedConstraint) Type() string {
	return "Reified(" + r.constraint.Type() + ")"
}

// String returns a human-readable representation.
// Implements ModelConstraint.
func (r *ReifiedConstraint) String() string {
	return fmt.Sprintf("Reified(%s, bool=%d)", r.constraint.String(), r.boolVar.ID())
}

// Propagate applies reification logic with bidirectional propagation.
//
// Algorithm:
//
//  1. Check boolean variable's domain:
//     - If bound to 1: propagate underlying constraint normally
//     - If bound to 0: constraint is disabled (we don't enforce violation)
//     - If {0,1}: attempt propagation and check if constraint is determined
//
//  2. If boolean is not yet bound and we propagate:
//     - Try propagating the constraint
//     - If constraint leads to failure → set boolean to 0
//     - If constraint is trivially satisfied → set boolean to 1
//     - Otherwise, boolean remains {0,1}
//
// Implements PropagationConstraint.
func (r *ReifiedConstraint) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("ReifiedConstraint.Propagate: nil solver")
	}
	// Note: state can be nil (means use initial model domains)

	// Get current boolean domain
	boolDomain := solver.GetDomain(state, r.boolVar.ID())
	if boolDomain == nil {
		return nil, fmt.Errorf("ReifiedConstraint.Propagate: boolean variable %d has no domain", r.boolVar.ID())
	}

	// Validate boolean domain is subset of {0,1}
	if boolDomain.Count() == 0 {
		return nil, fmt.Errorf("ReifiedConstraint.Propagate: boolean variable %d has empty domain", r.boolVar.ID())
	}

	// Check what values are in boolean domain
	hasOne := boolDomain.Has(1) // false
	hasTwo := boolDomain.Has(2) // true

	// Boolean domain must only contain 1 and/or 2
	if boolDomain.Count() > 2 || (!hasOne && !hasTwo) {
		// Domain contains values other than 1,2 or has invalid values
		return nil, fmt.Errorf("ReifiedConstraint.Propagate: boolean variable %d domain must be subset of {1,2}, got %s",
			r.boolVar.ID(), boolDomain.String())
	}

	// Case 1: Boolean is bound to 2 (true) - enforce constraint
	if hasTwo && !hasOne {
		// Propagate underlying constraint normally
		newState, err := r.constraint.Propagate(solver, state)
		if err != nil {
			// Constraint propagation failed, but boolean says it must be true
			// This is a genuine conflict
			return nil, err
		}
		return newState, nil
	}

	// Case 2: Boolean is bound to 1 (false) - enforce negation when possible
	if hasOne && !hasTwo {
		newState, err := r.enforceNegation(solver, state)
		if err != nil {
			return nil, err
		}
		return newState, nil
	}

	// Case 3: Boolean is {1,2} - unknown
	// Try propagating to see if constraint becomes determined

	// First, check if constraint is trivially satisfied
	// We do this by checking if all variables are bound and satisfy the constraint
	isTrivial, satisfied, err := r.isConstraintDetermined(solver, state)

	if err != nil {
		return nil, err
	}

	if isTrivial {
		// Constraint is determined
		if satisfied {
			// Set boolean to 2 (true)
			newDomain := boolDomain.Remove(1)
			newState, _ := solver.SetDomain(state, r.boolVar.ID(), newDomain)
			return newState, nil
		} else {
			// Set boolean to 1 (false)
			newDomain := boolDomain.Remove(2)
			newState, _ := solver.SetDomain(state, r.boolVar.ID(), newDomain)
			return newState, nil
		}
	}

	// Try propagating the constraint speculatively, but do not commit prunings
	// when the boolean is unknown. We only use the result to detect impossibility
	// (and thus set B=1). This avoids biasing domains toward the true branch.
	if _, err := r.constraint.Propagate(solver, state); err != nil {
		// Propagation failed - constraint cannot be satisfied under current domains
		// Set boolean to 1 (false) and keep original variable domains unchanged.
		newDomain := boolDomain.Remove(2)
		finalState, _ := solver.SetDomain(state, r.boolVar.ID(), newDomain)
		return finalState, nil
	}

	// Neither determined nor impossible: leave domains unchanged with B in {1,2}
	return state, nil
}

// isConstraintDetermined checks if the constraint's satisfaction is determined.
//
// Returns:
//   - isDetermined: true if we can definitively say if constraint is sat/unsat
//   - isSatisfied: if isDetermined, whether constraint is satisfied
//   - error: if check fails
//
// A constraint is determined as satisfied if:
//   - All its variables are bound (singleton domains)
//   - The values satisfy the constraint
//
// A constraint is determined as unsatisfied if:
//   - Propagating it leads to an empty domain (checked by caller)
//
// For constraints where variables are not all bound, we conservatively
// return isDetermined=false.
func (r *ReifiedConstraint) isConstraintDetermined(solver *Solver, state *SolverState) (bool, bool, error) {
	// Check if all constraint variables are bound
	vars := r.constraint.Variables()
	allBound := true

	for _, v := range vars {
		if v.ID() == r.boolVar.ID() {
			// Skip the boolean variable itself
			continue
		}

		domain := solver.GetDomain(state, v.ID())
		if domain == nil {
			return false, false, fmt.Errorf("variable %d has no domain", v.ID())
		}

		if !domain.IsSingleton() {
			allBound = false
			break
		}
	}

	if !allBound {
		// Not all variables bound, can't determine satisfaction
		return false, false, nil
	}

	// All variables are bound - check if constraint is satisfied
	// We do this by trying to propagate and seeing if we get a conflict

	// For constraints like AllDifferent, Arithmetic, Inequality, we can check directly
	// For now, use a general approach: if propagation doesn't fail, it's satisfied

	_, err := r.constraint.Propagate(solver, state)
	if err != nil {
		// Propagation failed with all variables bound = constraint violated
		return true, false, nil
	}

	// Propagation succeeded with all variables bound = constraint satisfied
	return true, true, nil
}

// BoolVar returns the boolean variable associated with this reified constraint.
// Useful for accessing the truth value during or after solving.
func (r *ReifiedConstraint) BoolVar() *FDVariable {
	return r.boolVar
}

// Constraint returns the underlying constraint being reified.
// Useful for introspection and debugging.
func (r *ReifiedConstraint) Constraint() PropagationConstraint {
	return r.constraint
}

// enforceNegation applies the logical negation of the underlying constraint
// as much as is practical without introducing new variables. The intent is to
// prevent solutions that would make the constraint true when the boolean is 1.
//
// Strategy by constraint type:
//   - Arithmetic (dst = src + k):
//   - If both bound and satisfy equality → conflict
//   - If one side bound → remove the matching value from the other
//   - Inequality:
//   - LessThan    false → enforce X ≥ Y via bounds
//   - LessEqual   false → enforce X > Y via bounds
//   - GreaterThan false → enforce X ≤ Y via bounds
//   - GreaterEqual false→ enforce X < Y via bounds
//   - NotEqual    false → enforce X = Y by intersecting domains
//   - AllDifferent:
//   - If all vars bound and all distinct → conflict (since NOT AllDifferent must hold)
//   - Otherwise, no pruning (would require disjunction of equalities)
func (r *ReifiedConstraint) enforceNegation(solver *Solver, state *SolverState) (*SolverState, error) {
	switch c := r.constraint.(type) {
	case *Arithmetic:
		srcDom := solver.GetDomain(state, c.src.ID())
		dstDom := solver.GetDomain(state, c.dst.ID())
		if srcDom == nil || dstDom == nil {
			return nil, fmt.Errorf("Reified(Arithmetic).Negation: nil domain")
		}
		// If both bound, forbid equality
		if srcDom.IsSingleton() && dstDom.IsSingleton() {
			sVal := srcDom.SingletonValue()
			dVal := dstDom.SingletonValue()
			if dVal == sVal+c.offset {
				return nil, fmt.Errorf("Reified(Arithmetic).Negation: equality holds but B=1")
			}
			return state, nil
		}
		// If one side singleton, remove matching value from the other
		cur := state
		if srcDom.IsSingleton() {
			sVal := srcDom.SingletonValue()
			forbid := sVal + c.offset
			if dstDom.Has(forbid) {
				nd := dstDom.Remove(forbid)
				if nd.Count() == 0 {
					return nil, fmt.Errorf("Reified(Arithmetic).Negation: dst empty after removing %d", forbid)
				}
				cur, _ = solver.SetDomain(cur, c.dst.ID(), nd)
				dstDom = nd
			}
		}
		if dstDom.IsSingleton() {
			dVal := dstDom.SingletonValue()
			forbid := dVal - c.offset
			sDom := solver.GetDomain(state, c.src.ID())
			if sDom.Has(forbid) {
				nd := sDom.Remove(forbid)
				if nd.Count() == 0 {
					return nil, fmt.Errorf("Reified(Arithmetic).Negation: src empty after removing %d", forbid)
				}
				cur, _ = solver.SetDomain(cur, c.src.ID(), nd)
			}
		}
		return cur, nil

	case *Inequality:
		xDom := solver.GetDomain(state, c.x.ID())
		yDom := solver.GetDomain(state, c.y.ID())
		if xDom == nil || yDom == nil {
			return nil, fmt.Errorf("Reified(Inequality).Negation: nil domain")
		}
		// Delegate to complementary bounds/equality logic
		switch c.kind {
		case LessThan:
			// Enforce X ≥ Y
			return c.propGE(solver, state, xDom, yDom)
		case LessEqual:
			// Enforce X > Y
			return c.propGT(solver, state, xDom, yDom)
		case GreaterThan:
			// Enforce X ≤ Y
			return c.propLE(solver, state, xDom, yDom)
		case GreaterEqual:
			// Enforce X < Y
			return c.propLT(solver, state, xDom, yDom)
		case NotEqual:
			// Enforce equality by intersecting domains
			inter := xDom.Intersect(yDom)
			if inter.Count() == 0 {
				return nil, fmt.Errorf("Reified(Inequality≠).Negation: no common value for equality")
			}
			cur := state
			if !inter.Equal(xDom) {
				cur, _ = solver.SetDomain(cur, c.x.ID(), inter)
			}
			// y to match x's new domain
			xNew := solver.GetDomain(cur, c.x.ID())
			inter2 := yDom.Intersect(xNew)
			if inter2.Count() == 0 {
				return nil, fmt.Errorf("Reified(Inequality≠).Negation: Y empty after equality enforcement")
			}
			if !inter2.Equal(yDom) {
				cur, _ = solver.SetDomain(cur, c.y.ID(), inter2)
			}
			return cur, nil
		default:
			return state, nil
		}

	case *AllDifferent:
		// If all variables are bound and all different, conflict under negation
		vars := c.variables
		allBound := true
		values := make(map[int]bool, len(vars))
		for _, v := range vars {
			d := solver.GetDomain(state, v.ID())
			if d == nil || d.Count() == 0 {
				return nil, fmt.Errorf("Reified(AllDifferent).Negation: nil/empty domain")
			}
			if !d.IsSingleton() {
				allBound = false
				continue
			}
			val := d.SingletonValue()
			if values[val] {
				// Duplicate found → NOT AllDifferent satisfied
				return state, nil
			}
			values[val] = true
		}
		if allBound {
			// All bound and all distinct → conflict with B=1
			return nil, fmt.Errorf("Reified(AllDifferent).Negation: all distinct but B=1")
		}
		// No strong pruning without introducing disjunctions
		return state, nil
	default:
		// Unknown constraint type: no pruning, but will detect conflicts once bound
		return state, nil
	}
}
