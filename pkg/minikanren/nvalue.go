// Package minikanren provides NValue-style global constraints.
//
// DistinctCount (aka NValue) constrains the number of distinct values taken
// by a list of variables. This file provides a composition-based, production
// implementation using existing, well-tested primitives (reification and
// BoolSum) to achieve safe bounds-consistent propagation without bespoke
// graph algorithms.
//
// Design overview
// ----------------
// Given variables X[1..n] with discrete domains, let U be the union of values
// present in their domains. For each value v in U, we create:
//   - Booleans b_iv reifying (X_i == v)
//   - A total T_v that counts how many X_i equal v via BoolSum(b_iv, T_v)
//     where T_v encodes count+1 in [1..n+1]
//   - A boolean used_v that is true iff some variable takes value v.
//     We implement used_v ↔ (T_v >= 2), which is equivalent to T_v ≠ 1.
//     To avoid introducing a general inequality reifier, we use a small gadget:
//   - Reify (T_v == 1) into b_zero_v
//   - Enforce XOR(used_v, b_zero_v) via BoolSum([used_v, b_zero_v], total={2})
//
// Finally, the number of distinct values equals the number of used_v that are
// true. We connect that with a BoolSum over all used_v to a caller-provided
// variable DPlus1 that encodes distinctCount+1.
//
// With this composition, standard propagation flows through the existing
// constraints and achieves sound bounds-consistent pruning. For example,
// when DPlus1 is fixed to 2 (distinctCount=1) and one X_i becomes bound to
// value a, all other values w≠a get used_w=false, which forces all b_jw=false
// and removes w from other X_j domains. This matches the typical AtMostNValues=1
// behavior without bespoke code paths.
package minikanren

import "fmt"

// DistinctCount composes internal reified equalities and boolean sums
// to count distinct values among vars. The distinct count is exposed as
// a variable DPlus1 with the standard encoding: distinctCount = DPlus1 - 1.
type DistinctCount struct {
	vars           []*FDVariable
	dPlus1         *FDVariable
	values         []int                     // union of candidate values
	usedBools      []*FDVariable             // used_v booleans per value v
	tTotals        []*FDVariable             // T_v totals (count+1) per value v
	zeroBools      []*FDVariable             // b_zero_v reifying T_v == 1
	eqReified      [][]PropagationConstraint // b_iv reifications
	perValSums     []PropagationConstraint   // BoolSum(b_iv, T_v)
	xorConstraints []PropagationConstraint   // XOR(used_v, b_zero_v) via BoolSum
	totalSum       PropagationConstraint     // BoolSum(used_v, dPlus1)
}

// NewDistinctCount builds the distinct-count composition and posts the
// internal constraints to the provided model.
//
// Parameters:
//   - model: the model to host auxiliary variables and constraints
//   - vars: non-empty slice of FD variables
//   - dPlus1: FD variable encoding distinctCount+1 in [1..len(U)+1]
//
// Returns a DistinctCount aggregate that participates as a ModelConstraint
// (for introspection consistency). Propagation is performed by the internal
// constraints; DistinctCount.Propagate is a no-op.
func NewDistinctCount(model *Model, vars []*FDVariable, dPlus1 *FDVariable) (*DistinctCount, error) {
	if model == nil {
		return nil, fmt.Errorf("NewDistinctCount: model cannot be nil")
	}
	if len(vars) == 0 {
		return nil, fmt.Errorf("NewDistinctCount: vars cannot be empty")
	}
	if dPlus1 == nil {
		return nil, fmt.Errorf("NewDistinctCount: dPlus1 cannot be nil")
	}

	// Compute union of candidate values and determine n
	n := len(vars)
	valueSet := make(map[int]struct{})
	for i, v := range vars {
		if v == nil || v.Domain() == nil || v.Domain().Count() == 0 {
			return nil, fmt.Errorf("NewDistinctCount: invalid variable at index %d", i)
		}
		v.Domain().IterateValues(func(val int) { valueSet[val] = struct{}{} })
	}
	if len(valueSet) == 0 {
		return nil, fmt.Errorf("NewDistinctCount: union of values is empty")
	}
	values := make([]int, 0, len(valueSet))
	for val := range valueSet {
		values = append(values, val)
	}

	// Prepare containers
	usedBools := make([]*FDVariable, len(values))
	tTotals := make([]*FDVariable, len(values))
	zeroBools := make([]*FDVariable, len(values))
	eqReified := make([][]PropagationConstraint, len(values))
	perValSums := make([]PropagationConstraint, len(values))
	xorConstraints := make([]PropagationConstraint, len(values))

	// Domain helpers
	boolDom := NewBitSetDomain(2)      // {1:false, 2:true}
	totalDom := NewBitSetDomain(n + 1) // [1..n+1] for per-value counts

	// Build, per value
	for vi, val := range values {
		// b_iv reifications and per-value total T_v
		b_iv := make([]*FDVariable, n)
		eqs := make([]PropagationConstraint, n)
		for i, x := range vars {
			b := model.NewVariable(boolDom)
			b_iv[i] = b
			eq, err := NewValueEqualsReified(x, val, b)
			if err != nil {
				return nil, fmt.Errorf("NewDistinctCount: reify X[%d]==%d: %w", i, val, err)
			}
			eqs[i] = eq
			model.AddConstraint(eq)
		}
		t := model.NewVariable(totalDom)
		sum, err := NewBoolSum(b_iv, t)
		if err != nil {
			return nil, fmt.Errorf("NewDistinctCount: BoolSum per value %d: %w", val, err)
		}
		model.AddConstraint(sum)

		// used_v ↔ (T_v ≠ 1) via XOR(used_v, b_zero_v) and reify T_v==1
		used := model.NewVariable(boolDom)
		bZero := model.NewVariable(boolDom)
		reifZero, err := NewValueEqualsReified(t, 1, bZero)
		if err != nil {
			return nil, fmt.Errorf("NewDistinctCount: reify T_v==1: %w", err)
		}
		model.AddConstraint(reifZero)
		// XOR as BoolSum([used,bZero]) = 1 true ⇒ total=2
		totalXor := model.NewVariable(NewBitSetDomainFromValues(3, []int{2}))
		xorSum, err := NewBoolSum([]*FDVariable{used, bZero}, totalXor)
		if err != nil {
			return nil, fmt.Errorf("NewDistinctCount: XOR BoolSum: %w", err)
		}
		model.AddConstraint(xorSum)

		// Collect
		usedBools[vi] = used
		zeroBools[vi] = bZero
		tTotals[vi] = t
		eqReified[vi] = eqs
		perValSums[vi] = sum
		xorConstraints[vi] = xorSum
	}

	// Sum of used_bools equals distinctCount (encoded by dPlus1)
	totalUsed, err := NewBoolSum(usedBools, dPlus1)
	if err != nil {
		return nil, fmt.Errorf("NewDistinctCount: total used BoolSum: %w", err)
	}
	model.AddConstraint(totalUsed)

	// Defensive copy of vars
	vv := make([]*FDVariable, len(vars))
	copy(vv, vars)

	c := &DistinctCount{
		vars:           vv,
		dPlus1:         dPlus1,
		values:         values,
		usedBools:      usedBools,
		tTotals:        tTotals,
		zeroBools:      zeroBools,
		eqReified:      eqReified,
		perValSums:     perValSums,
		xorConstraints: xorConstraints,
		totalSum:       totalUsed,
	}
	// Register as a constraint for introspection; propagation is handled by parts
	model.AddConstraint(c)
	return c, nil
}

// Variables returns all public-facing variables (vars + dPlus1).
func (c *DistinctCount) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(c.vars)+1)
	out = append(out, c.vars...)
	out = append(out, c.dPlus1)
	return out
}

func (c *DistinctCount) Type() string { return "DistinctCount" }
func (c *DistinctCount) String() string {
	return fmt.Sprintf("DistinctCount(|vars|=%d, |U|=%d)", len(c.vars), len(c.values))
}

// Propagate is a no-op: all pruning is performed by internal constraints.
func (c *DistinctCount) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}

// NewNValue creates an exact NValue: number of distinct values equals N
// where the encoding is NPlus1 = N + 1.
func NewNValue(model *Model, vars []*FDVariable, nPlus1 *FDVariable) (*DistinctCount, error) {
	return NewDistinctCount(model, vars, nPlus1)
}

// NewAtMostNValues enforces that the number of distinct values is ≤ N.
// The caller should provide limitPlus1 with domain [1..N+1]; the BoolSum over
// used_bools ties the count to limitPlus1, so the ≤ is enforced via the upper
// bound on limitPlus1.
func NewAtMostNValues(model *Model, vars []*FDVariable, limitPlus1 *FDVariable) (*DistinctCount, error) {
	return NewDistinctCount(model, vars, limitPlus1)
}

// NewAtLeastNValues enforces that the number of distinct values is ≥ N.
// Provide minPlus1 with domain [N+1..|U|+1]; the BoolSum over used_bools
// ties the count to minPlus1, so the ≥ is enforced via the lower bound on
// minPlus1.
func NewAtLeastNValues(model *Model, vars []*FDVariable, minPlus1 *FDVariable) (*DistinctCount, error) {
	return NewDistinctCount(model, vars, minPlus1)
}
