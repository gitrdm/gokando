// Package minikanren: global constraints - LinearSum (bounds propagation)
//
// LinearSum enforces an equality between a weighted sum of FD variables and
// an FD "total" variable using bounds-consistent propagation. This is a
// production-ready constraint for modeling many arithmetic relations
// (e.g., resource limits, cost-benefit models, profit maximization) while
// preserving the solver's immutable, lock-free semantics.
//
// Design
// - Variables: x[0..n-1] with domains over positive integers (1..Max)
// - Coefficients: arbitrary integers a[i] (positive, negative, or zero)
// - Total: t with domain over positive integers (1..Max)
// - Relation: sum(i) a[i]*x[i] = t
//
// Propagation (bounds consistency):
//   - Prune t to [SumMin..SumMax], where
//     SumMin = Σ (a[i]>0 ? a[i]*min(x[i]) : a[i]*max(x[i]))
//     SumMax = Σ (a[i]>0 ? a[i]*max(x[i]) : a[i]*min(x[i]))
//   - For each x[k], derive admissible interval:
//     a[k]*x[k] ∈ [t.min - OtherMax, t.max - OtherMin]
//     Convert to bounds on x[k] using sign-aware ceil/floor division and prune.
//
// Notes
//   - Mixed-sign coefficients are fully supported; negative coefficients enable
//     profit maximization, cost-benefit analysis, and offset modeling.
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
// - coeffs[i] can be positive, negative, or zero
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
	// Note: No restriction on coefficient signs; supports mixed-sign models
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

	// Compute SumMin, SumMax based on current bounds and coefficient signs
	// For positive coeffs: contribute min*c to sumMin, max*c to sumMax
	// For negative coeffs: contribute max*c to sumMin, min*c to sumMax (because c<0)
	sumMin, sumMax := 0, 0
	for i := 0; i < n; i++ {
		c := s.coeffs[i]
		if c == 0 {
			continue
		}
		minX := xdom[i].Min()
		maxX := xdom[i].Max()
		if c > 0 {
			sumMin += c * minX
			sumMax += c * maxX
		} else {
			// c < 0: minimum contribution is c*maxX, maximum is c*minX
			sumMin += c * maxX
			sumMax += c * minX
		}
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
		minX := xdom[i].Min()
		maxX := xdom[i].Max()
		if c > 0 {
			otherMinPrefix[i+1] = otherMinPrefix[i] + c*minX
			otherMaxPrefix[i+1] = otherMaxPrefix[i] + c*maxX
		} else if c < 0 {
			otherMinPrefix[i+1] = otherMinPrefix[i] + c*maxX
			otherMaxPrefix[i+1] = otherMaxPrefix[i] + c*minX
		} else {
			otherMinPrefix[i+1] = otherMinPrefix[i]
			otherMaxPrefix[i+1] = otherMaxPrefix[i]
		}
	}
	// Iterate variables and tighten bounds
	for i := 0; i < n; i++ {
		c := s.coeffs[i]
		if c == 0 {
			// Variable does not affect the sum; skip pruning x[i]
			continue
		}
		// Contribution of others (exclude contribution of x[i])
		minX := xdom[i].Min()
		maxX := xdom[i].Max()
		var myMinContrib, myMaxContrib int
		if c > 0 {
			myMinContrib = c * minX
			myMaxContrib = c * maxX
		} else {
			myMinContrib = c * maxX
			myMaxContrib = c * minX
		}
		otherMin := otherMinPrefix[n] - myMinContrib
		otherMax := otherMaxPrefix[n] - myMaxContrib

		// Admissible contribution for this var based on t range
		tMin := tdom.Min()
		tMax := tdom.Max()

		// a[i]*x[i] ∈ [tMin - otherMax, tMax - otherMin]
		contribMin := tMin - otherMax
		contribMax := tMax - otherMin

		// Bounds for x[i] depend on sign of c
		var xiMin, xiMax int
		if c > 0 {
			// c>0: use ceil/floor division
			xiMin = ceilDiv(contribMin, c)
			xiMax = floorDiv(contribMax, c)
		} else {
			// c<0: division reverses inequality
			// c*x ∈ [contribMin, contribMax] → x ∈ [contribMax/c, contribMin/c]
			// Since c<0, use specialized division
			xiMin = ceilDivNeg(contribMax, c)
			xiMax = floorDivNeg(contribMin, c)
		}

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

// ceilDivNeg returns ceil(a/b) for integers with b<0.
// Since b<0, division reverses inequality direction.
// ceil(a/b) when b<0 means the smallest integer x such that x >= a/b.
func ceilDivNeg(a, b int) int {
	if b >= 0 {
		panic("ceilDivNeg: non-negative divisor")
	}
	// Convert to positive divisor problem: a/b = a/(-|b|) = -a/|b|
	// ceil(a/b) = ceil(-a/|b|) = -floor(a/|b|)
	posB := -b
	if a >= 0 {
		// a>=0, b<0: a/b ≤ 0
		// ceil(a/b) = -(a/|b|) rounded down = -floor(a/|b|)
		return -(a / posB)
	}
	// a<0, b<0: a/b > 0
	// -a > 0, |b| > 0: (-a)/|b| > 0
	// ceil(a/b) = ceil((-a)/|b|) = ((-a) + |b| - 1) / |b|
	return ((-a) + posB - 1) / posB
}

// floorDivNeg returns floor(a/b) for integers with b<0.
func floorDivNeg(a, b int) int {
	if b >= 0 {
		panic("floorDivNeg: non-negative divisor")
	}
	posB := -b
	if a >= 0 {
		// a>=0, b<0: a/b ≤ 0
		// floor(a/b) = -(a/|b|) rounded up = -ceil(a/|b|) = -((a+|b|-1)/|b|)
		if a%posB == 0 {
			return -(a / posB)
		}
		return -(a/posB + 1)
	}
	// a<0, b<0: a/b > 0
	// floor(a/b) = floor((-a)/|b|) = (-a)/|b|
	return (-a) / posB
}
