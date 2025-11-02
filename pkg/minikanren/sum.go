// Package minikanren: global constraints - LinearSum (bounds propagation)
//
// LinearSum enforces an equality between a weighted sum of FD variables and
// an FD "total" variable using bounds-consistent propagation. This is a
// production-ready constraint for modeling many arithmetic relations
// (e.g., resource limits, digit column sums without carries) while preserving
// the solver's immutable, lock-free semantics.
//
// Design
// - Variables: x[0..n-1] with domains over positive integers (1..Max)
// - Coefficients: non-negative integers a[i] (to keep results in positive range)
// - Total: t with domain over positive integers (1..Max)
// - Relation: sum(i) a[i]*x[i] = t
//
// Propagation (bounds consistency):
//   - Prune t to [SumMin..SumMax], where
//     SumMin = Σ a[i]*min(x[i])
//     SumMax = Σ a[i]*max(x[i])
//   - For each x[k] with a[k] > 0, derive admissible interval via:
//     a[k]*x[k] ∈ [t.min - OtherMax, t.max - OtherMin]
//     where OtherMin (resp. OtherMax) is the contribution range of all j≠k.
//     Convert that to bounds on x[k] using ceil/floor division and prune domain.
//
// Notes
//   - Coefficients must be non-negative; this matches common modeling and
//     keeps all intermediate sums positive within bitset domains.
//   - If any variable or total is empty, the solver will detect via domain checks
//     and return an error (inconsistency).
//   - This constraint is intentionally bounds-only (interval reasoning). It is
//     fast and safe; value-level pruning would require heavier algorithms.
package minikanren

import (
	"fmt"
)

// LinearSum is a bounds-consistent weighted sum constraint: Σ a[i]*x[i] = t
type LinearSum struct {
	vars   []*FDVariable
	coeffs []int
	total  *FDVariable
}

// NewLinearSum constructs a new LinearSum constraint.
//
// Contract:
// - len(vars) > 0, len(vars) == len(coeffs)
// - coeffs[i] >= 0 for all i
// - total != nil
func NewLinearSum(vars []*FDVariable, coeffs []int, total *FDVariable) (*LinearSum, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("LinearSum: vars cannot be empty")
	}
	if len(vars) != len(coeffs) {
		return nil, fmt.Errorf("LinearSum: len(vars) != len(coeffs)")
	}
	if total == nil {
		return nil, fmt.Errorf("LinearSum: total cannot be nil")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("LinearSum: vars[%d] is nil", i)
		}
	}
	for i, c := range coeffs {
		if c < 0 {
			return nil, fmt.Errorf("LinearSum: coeffs[%d] is negative (%d)", i, c)
		}
	}
	// Defensive copies
	vcopy := make([]*FDVariable, len(vars))
	copy(vcopy, vars)
	ccopy := make([]int, len(coeffs))
	copy(ccopy, coeffs)

	return &LinearSum{vars: vcopy, coeffs: ccopy, total: total}, nil
}

// Variables implements ModelConstraint.
func (s *LinearSum) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(s.vars)+1)
	out = append(out, s.vars...)
	out = append(out, s.total)
	return out
}

// Type implements ModelConstraint.
func (s *LinearSum) Type() string { return "LinearSum" }

// String implements ModelConstraint.
func (s *LinearSum) String() string {
	return fmt.Sprintf("LinearSum(sum=%d terms -> total=%d)", len(s.vars), s.total.ID())
}

// Propagate applies bounds-consistent pruning.
// Implements PropagationConstraint.
func (s *LinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("LinearSum.Propagate: nil solver")
	}

	n := len(s.vars)
	// Fetch domains and validate
	xdom := make([]Domain, n)
	for i, v := range s.vars {
		d := solver.GetDomain(state, v.ID())
		if d == nil {
			return nil, fmt.Errorf("LinearSum: variable %d has nil domain", v.ID())
		}
		if d.Count() == 0 {
			return nil, fmt.Errorf("LinearSum: variable %d has empty domain", v.ID())
		}
		xdom[i] = d
	}
	tdom := solver.GetDomain(state, s.total.ID())
	if tdom == nil {
		return nil, fmt.Errorf("LinearSum: total variable %d has nil domain", s.total.ID())
	}
	if tdom.Count() == 0 {
		return nil, fmt.Errorf("LinearSum: total variable %d has empty domain", s.total.ID())
	}

	// Compute SumMin, SumMax based on current bounds and non-negative coeffs
	sumMin, sumMax := 0, 0
	for i := 0; i < n; i++ {
		c := s.coeffs[i]
		if c == 0 {
			continue
		}
		sumMin += c * xdom[i].Min()
		sumMax += c * xdom[i].Max()
	}

	// Prune total domain to [sumMin..sumMax]
	changed := false
	if tdom.Min() < sumMin {
		tdom = tdom.RemoveBelow(sumMin)
		changed = true
	}
	if tdom.Max() > sumMax {
		tdom = tdom.RemoveAbove(sumMax)
		changed = true
	}
	if tdom.Count() == 0 {
		return nil, fmt.Errorf("LinearSum: total domain became empty after pruning")
	}
	if changed {
		state, _ = solver.SetDomain(state, s.total.ID(), tdom)
	}

	// Precompute otherMin/otherMax partial sums for efficiency
	// Using arrays of prefix/suffix contributions
	otherMinPrefix := make([]int, n+1)
	otherMaxPrefix := make([]int, n+1)
	for i := 0; i < n; i++ {
		c := s.coeffs[i]
		otherMinPrefix[i+1] = otherMinPrefix[i] + c*xdom[i].Min()
		otherMaxPrefix[i+1] = otherMaxPrefix[i] + c*xdom[i].Max()
	}
	// Iterate variables and tighten bounds
	for i := 0; i < n; i++ {
		c := s.coeffs[i]
		if c == 0 {
			// Variable does not affect the sum; skip pruning x[i]
			continue
		}
		// Contribution of others
		otherMin := otherMinPrefix[n] - c*xdom[i].Min()
		otherMax := otherMaxPrefix[n] - c*xdom[i].Max()

		// Admissible contribution for this var based on t range
		tMin := tdom.Min()
		tMax := tdom.Max()

		// a[i]*x[i] ∈ [tMin - otherMax, tMax - otherMin]
		contribMin := tMin - otherMax
		contribMax := tMax - otherMin
		if contribMax < 0 {
			return nil, fmt.Errorf("LinearSum: negative admissible contribution for var %d", s.vars[i].ID())
		}

		// Bounds for x[i]
		// Since c>0, use ceil/floor division
		xiMin := ceilDiv(contribMin, c)
		xiMax := floorDiv(contribMax, c)

		// Intersect x[i] with [xiMin..xiMax]
		d := xdom[i]
		if d.Min() < xiMin {
			d = d.RemoveBelow(xiMin)
		}
		if d.Max() > xiMax {
			d = d.RemoveAbove(xiMax)
		}
		if d.Count() == 0 {
			return nil, fmt.Errorf("LinearSum: variable %d domain became empty", s.vars[i].ID())
		}
		if !d.Equal(xdom[i]) {
			state, _ = solver.SetDomain(state, s.vars[i].ID(), d)
			xdom[i] = d // keep local copy in sync for subsequent iterations
		}
	}

	return state, nil
}

// ceilDiv returns ceil(a/b) for integers with b>0.
func ceilDiv(a, b int) int {
	if b <= 0 {
		panic("ceilDiv: non-positive divisor")
	}
	if a >= 0 {
		return (a + b - 1) / b
	}
	// For negative a, integer division truncates toward zero; adjust manually
	return a / b // a is negative, b>0 → result floors toward zero which is ceil
}

// floorDiv returns floor(a/b) for integers with b>0.
func floorDiv(a, b int) int {
	if b <= 0 {
		panic("floorDiv: non-positive divisor")
	}
	if a >= 0 {
		return a / b
	}
	// For negative a, floor is a/b - 1 if not divisible exactly
	if a%b == 0 {
		return a / b
	}
	return a/b - 1
}
