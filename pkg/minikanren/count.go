// Package minikanren provides global constraints for constraint programming.
//
// This file implements the Count constraint and related counting functionality
// using reification to achieve arc-consistency.
package minikanren

import (
	"fmt"
)

// Count constrains the number of variables that equal a specific value.
//
// Given:
//   - vars: array of FD variables
//   - targetValue: the value to count
//   - countVar: FD variable representing the count
//
// The constraint ensures: countVar = |{v ∈ vars : v = targetValue}|
//
// Implementation uses reification:
//  1. Create boolean variables b[i] for each vars[i]
//  2. Reify: b[i] = 2 iff vars[i] = targetValue, and b[i] = 1 otherwise
//     (BitSetDomain is 1-indexed, so we use 1=false, 2=true)
//  3. Boolean-sum: sum(b[i] == 2) = count, represented by countVar = count + 1
//
// This achieves arc-consistency through:
//   - Reified constraints prune variable domains based on boolean values
//   - Sum constraint propagates bounds on countVar
//   - Boolean domains drive further pruning on vars
//
// Example: Count([X,Y,Z], 5, N) with X,Y,Z ∈ {1..10}, N ∈ {0..3}
//   - If X=5, Y=5 → N ∈ {2,3} (at least 2 equal 5)
//   - If N=0 → X,Y,Z ≠ 5
//   - If N=3 → X=Y=Z=5
//
// Complexity: O(n) propagation per variable domain change, where n = len(vars)
type Count struct {
	vars        []*FDVariable // Variables to count over
	targetValue int           // Value to count occurrences of
	countVar    *FDVariable   // Variable representing the count

	// Internal structures for propagation
	boolVars      []*FDVariable           // Boolean variables for reification
	eqConstraints []PropagationConstraint // Equality-reified constraints
	sumConstraint PropagationConstraint   // Sum of (b[i]==2) equals countVar-1
}

// NewCount creates a Count constraint.
//
// Parameters:
//   - model: the model to add boolean variables to
//   - vars: variables to count (must not be nil or empty)
//   - targetValue: the value to count occurrences of
//   - countVar: variable to hold the count, encoded as [1..len(vars)+1]
//     where actual count = countVar - 1 (offset by +1 due to 1-indexed domains)
//
// The constructor creates auxiliary boolean variables and reified constraints
// automatically. These are added to the model.
//
// Returns error if:
//   - model is nil
//   - vars is nil or empty
//   - countVar is nil
//   - countVar's domain maximum is less than len(vars)+1
func NewCount(model *Model, vars []*FDVariable, targetValue int, countVar *FDVariable) (*Count, error) {
	if model == nil {
		return nil, fmt.Errorf("NewCount: model cannot be nil")
	}
	if len(vars) == 0 {
		return nil, fmt.Errorf("NewCount: vars cannot be empty")
	}
	if countVar == nil {
		return nil, fmt.Errorf("NewCount: countVar cannot be nil")
	}

	// Validate countVar domain upper bound supports [1..len(vars)+1]
	// We encode counts 0..n as values 1..n+1
	if countVar.Domain().MaxValue() < len(vars)+1 {
		return nil, fmt.Errorf("NewCount: countVar max domain (%d) must be >= len(vars)+1 (%d)",
			countVar.Domain().MaxValue(), len(vars)+1)
	}

	n := len(vars)

	// Create boolean variables for reification (one per input variable)
	boolVars := make([]*FDVariable, n)
	eqConstraints := make([]PropagationConstraint, n)

	// For each variable, create: boolVar[i] = (vars[i] == targetValue)
	for i, v := range vars {
		// Create boolean variable with domain {1, 2} representing {false, true}
		// BitSetDomain is 1-indexed, so we use 1=false, 2=true
		boolDomain := NewBitSetDomain(2) // Creates domain with values {1, 2}

		boolVar := model.NewVariable(boolDomain)
		boolVars[i] = boolVar

		// Create reified constraint v == targetValue without auxiliary variable
		eqReified, err := NewValueEqualsReified(v, targetValue, boolVar)
		if err != nil {
			return nil, fmt.Errorf("NewCount: failed to create equality-reified constraint: %w", err)
		}

		eqConstraints[i] = eqReified

		// Add constraint to model
		model.AddConstraint(eqReified)
	}

	// Create Boolean-sum constraint: number of b[i]==2 equals countVar-1
	sumConstraint, err := NewBoolSum(boolVars, countVar)
	if err != nil {
		return nil, fmt.Errorf("NewCount: failed to create sum constraint: %w", err)
	}

	// Add sum constraint to model
	model.AddConstraint(sumConstraint)

	// Defensive copy of vars
	varsCopy := make([]*FDVariable, len(vars))
	copy(varsCopy, vars)

	return &Count{
		vars:          varsCopy,
		targetValue:   targetValue,
		countVar:      countVar,
		boolVars:      boolVars,
		eqConstraints: eqConstraints,
		sumConstraint: sumConstraint,
	}, nil
}

// ValueEqualsReified links a variable v and a boolean b such that b=2 iff v==target.
// Domain conventions: b ∈ {1=false, 2=true}
type ValueEqualsReified struct {
	v       *FDVariable
	target  int
	boolVar *FDVariable
}

// NewValueEqualsReified creates a reified equality to a constant target.
func NewValueEqualsReified(v *FDVariable, target int, boolVar *FDVariable) (*ValueEqualsReified, error) {
	if v == nil {
		return nil, fmt.Errorf("NewValueEqualsReified: v cannot be nil")
	}
	if boolVar == nil {
		return nil, fmt.Errorf("NewValueEqualsReified: boolVar cannot be nil")
	}
	return &ValueEqualsReified{v: v, target: target, boolVar: boolVar}, nil
}

func (c *ValueEqualsReified) Variables() []*FDVariable { return []*FDVariable{c.v, c.boolVar} }
func (c *ValueEqualsReified) Type() string             { return "ValueEqualsReified" }
func (c *ValueEqualsReified) String() string {
	return fmt.Sprintf("ValueEqualsReified(v=%d == %d -> b=%d)", c.v.ID(), c.target, c.boolVar.ID())
}

// Propagate enforces b ↔ (v == target) with bidirectional pruning.
func (c *ValueEqualsReified) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("ValueEqualsReified.Propagate: nil solver")
	}
	vDom := solver.GetDomain(state, c.v.ID())
	bDom := solver.GetDomain(state, c.boolVar.ID())
	if vDom == nil || vDom.Count() == 0 {
		return nil, fmt.Errorf("ValueEqualsReified.Propagate: v has empty domain")
	}
	if bDom == nil || bDom.Count() == 0 {
		return nil, fmt.Errorf("ValueEqualsReified.Propagate: b has empty domain")
	}
	has1 := bDom.Has(1)
	has2 := bDom.Has(2)
	if bDom.Count() > 2 || (!has1 && !has2) {
		return nil, fmt.Errorf("ValueEqualsReified.Propagate: b domain must be subset of {1,2}, got %s", bDom.String())
	}

	cur := state

	// If target not in v, then b must be false
	if !vDom.Has(c.target) {
		if has2 {
			nd := bDom.Remove(2)
			cur, _ = solver.SetDomain(cur, c.boolVar.ID(), nd)
			bDom = nd
			has2 = false
		}
	}

	// If v is singleton equal to target, b must be true
	if vDom.IsSingleton() && vDom.SingletonValue() == c.target {
		if has1 {
			nd := bDom.Remove(1)
			cur, _ = solver.SetDomain(cur, c.boolVar.ID(), nd)
			bDom = nd
			has1 = false
		}
	}

	// Reflect from b to v
	has1 = bDom.Has(1)
	has2 = bDom.Has(2)
	if has2 && !has1 {
		// b = true ⇒ v = target
		if vDom.Has(c.target) {
			nd := NewBitSetDomainFromValues(vDom.MaxValue(), []int{c.target})
			if !nd.Equal(vDom) {
				cur, _ = solver.SetDomain(cur, c.v.ID(), nd)
			}
		} else {
			return nil, fmt.Errorf("ValueEqualsReified: b=2 but v cannot take target %d", c.target)
		}
	} else if has1 && !has2 {
		// b = false ⇒ v ≠ target
		if vDom.Has(c.target) {
			nd := vDom.Remove(c.target)
			if nd.Count() == 0 {
				return nil, fmt.Errorf("ValueEqualsReified: b=1 would empty v by removing %d", c.target)
			}
			cur, _ = solver.SetDomain(cur, c.v.ID(), nd)
		}
	}

	return cur, nil
}

// Variables returns all variables involved in this constraint.
// Includes the input variables, count variable, and auxiliary boolean variables.
// Implements ModelConstraint.
func (c *Count) Variables() []*FDVariable {
	// Count constraint involves: vars + countVar + boolVars
	allVars := make([]*FDVariable, 0, len(c.vars)+1+len(c.boolVars))
	allVars = append(allVars, c.vars...)
	allVars = append(allVars, c.countVar)
	allVars = append(allVars, c.boolVars...)
	return allVars
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (c *Count) Type() string {
	return "Count"
}

// String returns a human-readable representation.
// Implements ModelConstraint.
func (c *Count) String() string {
	varIDs := make([]int, len(c.vars))
	for i, v := range c.vars {
		varIDs[i] = v.ID()
	}
	return fmt.Sprintf("Count(%v, value=%d, count=%d)", varIDs, c.targetValue, c.countVar.ID())
}

// Propagate applies the Count constraint's propagation.
//
// The Count constraint itself doesn't need to do propagation because
// the reified constraints and sum constraint handle it. However, we
// implement Propagate to satisfy the PropagationConstraint interface
// and potentially add Count-specific optimizations.
//
// Implements PropagationConstraint.
func (c *Count) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	// The reified constraints and sum constraint are already in the model
	// and will be propagated by the solver's fixed-point loop.
	// We don't need to do anything here.
	//
	// However, we can add a sanity check or optimizations if needed.
	return state, nil
}

// BoundsSum is a simple sum constraint with bounds propagation.
//
// Constrains: sum(vars) = total
//
// Bounds propagation:
//   - total.min >= sum(vars[i].min)
//   - total.max <= sum(vars[i].max)
//   - For each var[i]: var[i].min >= total.min - sum(vars[j!=i].max)
//   - For each var[i]: var[i].max <= total.max - sum(vars[j!=i].min)
//
// This is a simplified version sufficient for counting with 0/1 variables.
// A full Sum constraint would support coefficients and inequalities.
type BoundsSum struct {
	vars  []*FDVariable
	total *FDVariable
}

// NewBoundsSum creates a bounds-propagating sum constraint.
//
// Parameters:
//   - vars: variables to sum (must not be nil or empty)
//   - total: variable representing the sum
//
// Returns error if vars is nil/empty or total is nil.
func NewBoundsSum(vars []*FDVariable, total *FDVariable) (*BoundsSum, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("NewBoundsSum: vars cannot be empty")
	}
	if total == nil {
		return nil, fmt.Errorf("NewBoundsSum: total cannot be nil")
	}

	varsCopy := make([]*FDVariable, len(vars))
	copy(varsCopy, vars)

	return &BoundsSum{
		vars:  varsCopy,
		total: total,
	}, nil
}

// Variables returns the variables involved in this constraint.
// Implements ModelConstraint.
func (b *BoundsSum) Variables() []*FDVariable {
	allVars := make([]*FDVariable, 0, len(b.vars)+1)
	allVars = append(allVars, b.vars...)
	allVars = append(allVars, b.total)
	return allVars
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (b *BoundsSum) Type() string {
	return "BoundsSum"
}

// String returns a human-readable representation.
// Implements ModelConstraint.
func (b *BoundsSum) String() string {
	varIDs := make([]int, len(b.vars))
	for i, v := range b.vars {
		varIDs[i] = v.ID()
	}
	return fmt.Sprintf("BoundsSum(%v, total=%d)", varIDs, b.total.ID())
}

// Propagate applies bounds propagation for sum constraint.
// Implements PropagationConstraint.
func (b *BoundsSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("BoundsSum.Propagate: nil solver")
	}
	// Note: state can be nil (means use initial model domains)

	// Get current domains
	varDomains := make([]Domain, len(b.vars))
	for i, v := range b.vars {
		varDomains[i] = solver.GetDomain(state, v.ID())
		if varDomains[i] == nil || varDomains[i].Count() == 0 {
			return nil, fmt.Errorf("BoundsSum.Propagate: variable %d has empty domain", v.ID())
		}
	}

	totalDomain := solver.GetDomain(state, b.total.ID())
	if totalDomain == nil || totalDomain.Count() == 0 {
		return nil, fmt.Errorf("BoundsSum.Propagate: total variable %d has empty domain", b.total.ID())
	}

	currentState := state

	// Compute sum bounds from variables
	sumMin := 0
	sumMax := 0
	for _, dom := range varDomains {
		sumMin += dom.Min()
		sumMax += dom.Max()
	}

	// Prune total domain
	newTotalDomain := totalDomain.RemoveBelow(sumMin).RemoveAbove(sumMax)
	if newTotalDomain.Count() == 0 {
		return nil, fmt.Errorf("BoundsSum.Propagate: total bounds [%d,%d] incompatible with sum bounds [%d,%d]",
			totalDomain.Min(), totalDomain.Max(), sumMin, sumMax)
	}

	if !newTotalDomain.Equal(totalDomain) {
		currentState, _ = solver.SetDomain(currentState, b.total.ID(), newTotalDomain)
		totalDomain = newTotalDomain
	}

	// Prune variable domains based on total bounds
	totalMin := totalDomain.Min()
	totalMax := totalDomain.Max()

	for i, v := range b.vars {
		// Compute sum of other variables
		otherMin := 0
		otherMax := 0
		for j, dom := range varDomains {
			if i != j {
				otherMin += dom.Min()
				otherMax += dom.Max()
			}
		}

		// Bounds for var[i]
		// var[i] >= totalMin - otherMax
		// var[i] <= totalMax - otherMin
		varMin := totalMin - otherMax
		varMax := totalMax - otherMin

		// Prune var[i] domain
		newVarDomain := varDomains[i].RemoveBelow(varMin).RemoveAbove(varMax)
		if newVarDomain.Count() == 0 {
			return nil, fmt.Errorf("BoundsSum.Propagate: variable %d domain empty after bounds propagation", v.ID())
		}

		if !newVarDomain.Equal(varDomains[i]) {
			currentState, _ = solver.SetDomain(currentState, v.ID(), newVarDomain)
			varDomains[i] = newVarDomain
		}
	}

	return currentState, nil
}

// BoolSum constrains the number of boolean variables that are true.
//
// Encoding:
//   - Each boolean variable has domain subset of {1=false, 2=true}
//   - total represents the count of trues, encoded as total = count + 1
//     so total ranges in [1..n+1] for n booleans
//
// Propagation:
//   - Let lb = sum of per-var minimum contributions (1 if var must be true, else 0)
//   - Let ub = sum of per-var maximum contributions (1 if var may be true, else 0)
//   - Prune total to [lb+1, ub+1]
//   - For each var, using otherLb = lb - varMin and otherUb = ub - varMax:
//   - If (total.min-1) > otherUb  => var must be true (set to {2})
//   - If (total.max-1) < otherLb  => var must be false (set to {1})
//
// This achieves bounds consistency for boolean sums and is sufficient for Count.
type BoolSum struct {
	vars  []*FDVariable
	total *FDVariable // domain [1..n+1], representing count+1
}

// NewBoolSum creates a BoolSum constraint over boolean variables {1,2} and a total in [1..n+1].
func NewBoolSum(vars []*FDVariable, total *FDVariable) (*BoolSum, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("NewBoolSum: vars cannot be empty")
	}
	if total == nil {
		return nil, fmt.Errorf("NewBoolSum: total cannot be nil")
	}
	// defensive copy
	vs := make([]*FDVariable, len(vars))
	copy(vs, vars)
	return &BoolSum{vars: vs, total: total}, nil
}

// Variables returns all variables in the BoolSum constraint.
func (b *BoolSum) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(b.vars)+1)
	out = append(out, b.vars...)
	out = append(out, b.total)
	return out
}

// Type returns the constraint type identifier.
func (b *BoolSum) Type() string { return "BoolSum" }

// String returns a human-readable representation.
func (b *BoolSum) String() string {
	ids := make([]int, len(b.vars))
	for i, v := range b.vars {
		ids[i] = v.ID()
	}
	return fmt.Sprintf("BoolSum(%v, total=%d)", ids, b.total.ID())
}

// Propagate enforces bounds consistency on the sum of boolean vars.
func (b *BoolSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("BoolSum.Propagate: nil solver")
	}

	// Read domains
	boolDoms := make([]Domain, len(b.vars))
	for i, v := range b.vars {
		d := solver.GetDomain(state, v.ID())
		if d == nil || d.Count() == 0 {
			return nil, fmt.Errorf("BoolSum.Propagate: boolean var %d has empty domain", v.ID())
		}
		// Validate subset of {1,2}
		has1 := d.Has(1)
		has2 := d.Has(2)
		if d.Count() > 2 || (!has1 && !has2) {
			return nil, fmt.Errorf("BoolSum.Propagate: boolean var %d domain must be subset of {1,2}, got %s", v.ID(), d.String())
		}
		boolDoms[i] = d
	}
	totalDom := solver.GetDomain(state, b.total.ID())
	if totalDom == nil || totalDom.Count() == 0 {
		return nil, fmt.Errorf("BoolSum.Propagate: total var %d has empty domain", b.total.ID())
	}

	cur := state

	// Compute lb/ub for actual count (0..n)
	lb := 0
	ub := 0
	// Also collect per-var min/max for reuse
	varMins := make([]int, len(b.vars))
	varMaxs := make([]int, len(b.vars))
	for i, d := range boolDoms {
		has1 := d.Has(1)
		has2 := d.Has(2)
		varMin := 0
		varMax := 0
		switch {
		case has2 && !has1:
			// {2}
			varMin, varMax = 1, 1
		case has2 && has1:
			// {1,2}
			varMin, varMax = 0, 1
		case !has2 && has1:
			// {1}
			varMin, varMax = 0, 0
		default:
			return nil, fmt.Errorf("BoolSum.Propagate: boolean var has invalid domain %s", d.String())
		}
		varMins[i] = varMin
		varMaxs[i] = varMax
		lb += varMin
		ub += varMax
	}

	// Prune total to [lb+1, ub+1]
	newTotal := totalDom.RemoveBelow(lb + 1).RemoveAbove(ub + 1)
	if newTotal.Count() == 0 {
		return nil, fmt.Errorf("BoolSum.Propagate: total domain empty after pruning to [%d,%d] (had %s)", lb+1, ub+1, totalDom.String())
	}
	if !newTotal.Equal(totalDom) {
		cur, _ = solver.SetDomain(cur, b.total.ID(), newTotal)
		totalDom = newTotal
	}

	// Translate total bounds to actual count bounds
	cmin := totalDom.Min() - 1
	cmax := totalDom.Max() - 1

	// For each boolean var, deduce forced truth values using
	// newMin = max(varMin, cmin - otherUb)
	// newMax = min(varMax, cmax - otherLb)
	for i, v := range b.vars {
		otherLb := lb - varMins[i]
		otherUb := ub - varMaxs[i]

		newMin := varMins[i]
		if t := cmin - otherUb; t > newMin {
			newMin = t
		}
		newMax := varMaxs[i]
		if t := cmax - otherLb; t < newMax {
			newMax = t
		}

		d := boolDoms[i]
		// Infeasible
		if newMin > newMax {
			return nil, fmt.Errorf("BoolSum.Propagate: infeasible bounds for var %d", v.ID())
		}
		// Force true
		if newMin == 1 {
			if !d.Has(2) {
				return nil, fmt.Errorf("BoolSum.Propagate: var %d must be true, but domain %s lacks 2", v.ID(), d.String())
			}
			nd := d.Remove(1)
			if !nd.Equal(d) {
				cur, _ = solver.SetDomain(cur, v.ID(), nd)
				boolDoms[i] = nd
			}
			continue
		}
		// Forbid true
		if newMax == 0 {
			if !d.Has(1) {
				return nil, fmt.Errorf("BoolSum.Propagate: var %d must be false, but domain %s lacks 1", v.ID(), d.String())
			}
			nd := d.Remove(2)
			if !nd.Equal(d) {
				cur, _ = solver.SetDomain(cur, v.ID(), nd)
				boolDoms[i] = nd
			}
			continue
		}
	}

	return cur, nil
}
