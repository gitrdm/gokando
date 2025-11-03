// Package minikanren: global constraint - ElementValues (table element)
//
// ElementValues enforces: result = values[index]
// - index: finite-domain variable whose values are 1-based indices into 'values'
// - values: fixed slice of positive integers (acts like a constant array)
// - result: finite-domain variable that must equal the value referenced by 'index'
//
// Propagation (arc-consistent over the fixed table):
// 1) Index bounds pruning to valid range [1..len(values)].
// 2) From index to result: result ∈ { values[i] | i ∈ indexDomain }.
// 3) From result to index: index ∈ { i | values[i] ∈ resultDomain }.
//
// Notes
// - We allow duplicate entries in 'values'. The constraint naturally handles it.
// - All domains are immutable; SetDomain returns a new state preserving copy-on-write semantics.
// - If any domain becomes empty, propagation signals inconsistency via error.
package minikanren

import "fmt"

// ElementValues is a constraint linking an index variable, a constant table of values,
// and a result variable such that result = values[index].
type ElementValues struct {
	index  *FDVariable
	values []int
	result *FDVariable
}

// NewElementValues constructs a new ElementValues constraint.
//
// Contract:
// - index != nil, result != nil
// - len(values) > 0
func NewElementValues(index *FDVariable, values []int, result *FDVariable) (*ElementValues, error) {
	if index == nil {
		return nil, fmt.Errorf("ElementValues: index cannot be nil")
	}
	if result == nil {
		return nil, fmt.Errorf("ElementValues: result cannot be nil")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("ElementValues: values cannot be empty")
	}
	vcopy := make([]int, len(values))
	copy(vcopy, values)
	return &ElementValues{index: index, values: vcopy, result: result}, nil
}

// Variables returns the involved variables. Implements ModelConstraint.
func (e *ElementValues) Variables() []*FDVariable { return []*FDVariable{e.index, e.result} }

// Type returns the constraint identifier. Implements ModelConstraint.
func (e *ElementValues) Type() string { return "ElementValues" }

// String returns a human-readable description. Implements ModelConstraint.
func (e *ElementValues) String() string {
	return fmt.Sprintf("ElementValues(result=%d = values[index=%d], n=%d)", e.result.ID(), e.index.ID(), len(e.values))
}

// Propagate enforces result = values[index] bidirectionally.
// Implements PropagationConstraint.
func (e *ElementValues) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("ElementValues.Propagate: nil solver")
	}

	idxDom := solver.GetDomain(state, e.index.ID())
	resDom := solver.GetDomain(state, e.result.ID())
	if idxDom == nil || idxDom.Count() == 0 {
		return nil, fmt.Errorf("ElementValues: index has empty domain")
	}
	if resDom == nil || resDom.Count() == 0 {
		return nil, fmt.Errorf("ElementValues: result has empty domain")
	}

	n := len(e.values)
	cur := state

	// 1) Clamp index domain to [1..n]
	changed := false
	if idxDom.Min() < 1 {
		idxDom = idxDom.RemoveBelow(1)
		changed = true
	}
	if idxDom.Max() > n {
		idxDom = idxDom.RemoveAbove(n)
		changed = true
	}
	if idxDom.Count() == 0 {
		return nil, fmt.Errorf("ElementValues: index domain empty after clamping to [1..%d]", n)
	}
	if changed {
		cur, _ = solver.SetDomain(cur, e.index.ID(), idxDom)
	}

	// 2) From index to result: allowed result values are those referenced by any admissible index
	allowedResVals := make([]int, 0, idxDom.Count())
	idxDom.IterateValues(func(i int) {
		if i >= 1 && i <= n {
			allowedResVals = append(allowedResVals, e.values[i-1])
		}
	})
	if len(allowedResVals) == 0 {
		return nil, fmt.Errorf("ElementValues: no result values supported by current index domain")
	}
	allowedResDom := NewBitSetDomainFromValues(resDom.MaxValue(), allowedResVals)
	resFiltered := resDom.Intersect(allowedResDom)
	if resFiltered.Count() == 0 {
		return nil, fmt.Errorf("ElementValues: result domain inconsistent with index domain")
	}
	if !resFiltered.Equal(resDom) {
		cur, _ = solver.SetDomain(cur, e.result.ID(), resFiltered)
		resDom = resFiltered
	}

	// 3) From result to index: keep only indices whose mapped value is in result domain
	allowedIdx := make([]int, 0, idxDom.Count())
	idxDom.IterateValues(func(i int) {
		if i >= 1 && i <= n {
			v := e.values[i-1]
			if resDom.Has(v) {
				allowedIdx = append(allowedIdx, i)
			}
		}
	})
	if len(allowedIdx) == 0 {
		return nil, fmt.Errorf("ElementValues: index domain has no value compatible with result domain")
	}
	idxFiltered := NewBitSetDomainFromValues(idxDom.MaxValue(), allowedIdx)
	if idxFiltered.Count() == 0 {
		return nil, fmt.Errorf("ElementValues: index domain emptied unexpectedly")
	}
	if !idxFiltered.Equal(idxDom) {
		cur, _ = solver.SetDomain(cur, e.index.ID(), idxFiltered)
	}

	return cur, nil
}
