// Package minikanren provides Min/Max-of-array global constraints.
//
// These constraints link a result variable R to the minimum or maximum
// value among a list of FD variables X[1..n]. They implement safe, bounds-
// consistent propagation without over-pruning:
//   - Min(vars, R):
//     R ∈ [min_i Min(Xi) .. min_i Max(Xi)]
//     and for all i: Xi ≥ R (i.e., prune Xi below R.min)
//   - Max(vars, R):
//     R ∈ [max_i Min(Xi) .. max_i Max(Xi)]
//     and for all i: Xi ≤ R (i.e., prune Xi above R.max)
//
// This propagation is sound and inexpensive (O(n)) per call. Stronger
// propagation (e.g., identifying unique carriers of the current extremum)
// could prune more but is intentionally avoided here to keep the behavior
// simple, predictable, and integration-friendly with the solver's fixed-point loop.
package minikanren

import "fmt"

// MinOfArray enforces R = min(vars) with bounds-consistent pruning.
type MinOfArray struct {
	vars []*FDVariable
	r    *FDVariable
}

// NewMin creates a MinOfArray constraint with result variable r.
//
// Contract:
//   - vars: non-empty slice; each variable must have a positive domain (1..MaxValue)
//   - r: non-nil result variable with a positive domain
func NewMin(vars []*FDVariable, r *FDVariable) (PropagationConstraint, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("Min: vars must be non-empty")
	}
	if r == nil {
		return nil, fmt.Errorf("Min: result variable r must not be nil")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("Min: nil variable at index %d", i)
		}
		if d := v.Domain(); d == nil || d.MaxValue() <= 0 {
			return nil, fmt.Errorf("Min: variable %d has invalid domain", i)
		}
	}
	if r.Domain() == nil || r.Domain().MaxValue() <= 0 {
		return nil, fmt.Errorf("Min: result variable has invalid domain")
	}
	vv := make([]*FDVariable, len(vars))
	copy(vv, vars)
	return &MinOfArray{vars: vv, r: r}, nil
}

func (m *MinOfArray) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(m.vars)+1)
	out = append(out, m.vars...)
	out = append(out, m.r)
	return out
}

func (m *MinOfArray) Type() string { return "MinOfArray" }

func (m *MinOfArray) String() string {
	return fmt.Sprintf("MinOfArray(|vars|=%d)", len(m.vars))
}

// Propagate clamps r to feasible [min_i Min(Xi) .. min_i Max(Xi)] and enforces Xi >= r.min.
func (m *MinOfArray) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Min.Propagate: nil solver")
	}
	n := len(m.vars)
	if n == 0 {
		return state, nil
	}

	dx := make([]Domain, n)
	for i := 0; i < n; i++ {
		dx[i] = solver.GetDomain(state, m.vars[i].ID())
		if dx[i] == nil || dx[i].Count() == 0 {
			return nil, fmt.Errorf("Min: invalid or empty domain at index %d", i)
		}
	}
	dr := solver.GetDomain(state, m.r.ID())
	if dr == nil || dr.Count() == 0 {
		return nil, fmt.Errorf("Min: invalid or empty result domain")
	}

	// Compute bounds for R
	a := dx[0].Min() // min of mins
	b := dx[0].Max() // min of maxes
	for i := 1; i < n; i++ {
		if v := dx[i].Min(); v < a {
			a = v
		}
		if v := dx[i].Max(); v < b {
			b = v
		}
	}
	if a > b {
		return nil, fmt.Errorf("Min: infeasible bounds a=%d > b=%d", a, b)
	}

	// Prune R to [a..b]
	newDr := dr
	if dr.Min() < a {
		newDr = newDr.RemoveBelow(a)
	}
	if newDr.Count() == 0 {
		return nil, fmt.Errorf("Min: pruning R below %d empties domain", a)
	}
	if dr.Max() > b {
		newDr = newDr.RemoveAbove(b)
	}
	if newDr.Count() == 0 {
		return nil, fmt.Errorf("Min: pruning R above %d empties domain", b)
	}

	newState := state
	if !newDr.Equal(dr) {
		var changed bool
		newState, changed = solver.SetDomain(newState, m.r.ID(), newDr)
		if changed {
			dr = newDr
		}
	}

	// Enforce Xi >= R.min for all i
	rMin := dr.Min()
	for i := 0; i < n; i++ {
		nd := dx[i]
		if nd.Min() < rMin {
			nd = nd.RemoveBelow(rMin)
			if nd.Count() == 0 {
				return nil, fmt.Errorf("Min: pruning Xi at %d below %d empties domain", i, rMin)
			}
			if !nd.Equal(dx[i]) {
				var changed bool
				newState, changed = solver.SetDomain(newState, m.vars[i].ID(), nd)
				if changed {
					dx[i] = nd
				}
			}
		}
	}

	return newState, nil
}

// MaxOfArray enforces R = max(vars) with bounds-consistent pruning.
type MaxOfArray struct {
	vars []*FDVariable
	r    *FDVariable
}

// NewMax creates a MaxOfArray constraint with result variable r.
//
// Contract:
//   - vars: non-empty slice; each variable must have a positive domain (1..MaxValue)
//   - r: non-nil result variable with a positive domain
func NewMax(vars []*FDVariable, r *FDVariable) (PropagationConstraint, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("Max: vars must be non-empty")
	}
	if r == nil {
		return nil, fmt.Errorf("Max: result variable r must not be nil")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("Max: nil variable at index %d", i)
		}
		if d := v.Domain(); d == nil || d.MaxValue() <= 0 {
			return nil, fmt.Errorf("Max: variable %d has invalid domain", i)
		}
	}
	if r.Domain() == nil || r.Domain().MaxValue() <= 0 {
		return nil, fmt.Errorf("Max: result variable has invalid domain")
	}
	vv := make([]*FDVariable, len(vars))
	copy(vv, vars)
	return &MaxOfArray{vars: vv, r: r}, nil
}

func (m *MaxOfArray) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(m.vars)+1)
	out = append(out, m.vars...)
	out = append(out, m.r)
	return out
}

func (m *MaxOfArray) Type() string { return "MaxOfArray" }

func (m *MaxOfArray) String() string {
	return fmt.Sprintf("MaxOfArray(|vars|=%d)", len(m.vars))
}

// Propagate clamps r to feasible [max_i Min(Xi) .. max_i Max(Xi)] and enforces Xi <= r.max.
func (m *MaxOfArray) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Max.Propagate: nil solver")
	}
	n := len(m.vars)
	if n == 0 {
		return state, nil
	}

	dx := make([]Domain, n)
	for i := 0; i < n; i++ {
		dx[i] = solver.GetDomain(state, m.vars[i].ID())
		if dx[i] == nil || dx[i].Count() == 0 {
			return nil, fmt.Errorf("Max: invalid or empty domain at index %d", i)
		}
	}
	dr := solver.GetDomain(state, m.r.ID())
	if dr == nil || dr.Count() == 0 {
		return nil, fmt.Errorf("Max: invalid or empty result domain")
	}

	// Compute bounds for R
	a := dx[0].Min() // max of mins
	b := dx[0].Max() // max of maxes
	for i := 1; i < n; i++ {
		if v := dx[i].Min(); v > a {
			a = v
		}
		if v := dx[i].Max(); v > b {
			b = v
		}
	}
	if a > b {
		return nil, fmt.Errorf("Max: infeasible bounds a=%d > b=%d", a, b)
	}

	// Prune R to [a..b]
	newDr := dr
	if dr.Min() < a {
		newDr = newDr.RemoveBelow(a)
	}
	if newDr.Count() == 0 {
		return nil, fmt.Errorf("Max: pruning R below %d empties domain", a)
	}
	if dr.Max() > b {
		newDr = newDr.RemoveAbove(b)
	}
	if newDr.Count() == 0 {
		return nil, fmt.Errorf("Max: pruning R above %d empties domain", b)
	}

	newState := state
	if !newDr.Equal(dr) {
		var changed bool
		newState, changed = solver.SetDomain(newState, m.r.ID(), newDr)
		if changed {
			dr = newDr
		}
	}

	// Enforce Xi <= R.max for all i
	rMax := dr.Max()
	for i := 0; i < n; i++ {
		nd := dx[i]
		if nd.Max() > rMax {
			nd = nd.RemoveAbove(rMax)
			if nd.Count() == 0 {
				return nil, fmt.Errorf("Max: pruning Xi at %d above %d empties domain", i, rMax)
			}
			if !nd.Equal(dx[i]) {
				var changed bool
				newState, changed = solver.SetDomain(newState, m.vars[i].ID(), nd)
				if changed {
					dx[i] = nd
				}
			}
		}
	}

	return newState, nil
}
