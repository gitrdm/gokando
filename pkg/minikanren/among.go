// Package minikanren adds an Among global constraint.
//
// Among(vars, S, K) counts how many variables in vars take a value from the set S
// and constrains that count to equal K (with the solver's positive-domain encoding).
//
// Contract and encoding:
//   - vars: non-empty slice of FD variables with positive integer domains (1..MaxValue)
//   - S: finite set of allowed values (subset of 1..MaxValue); represented internally as a BitSetDomain
//   - K: FD variable encoding the count using the solver's convention from Count:
//     K ∈ [1 .. n+1] encodes actual count = K-1, where n = len(vars)
//
// Propagation (bounds-consistent, O(n·d)):
//   - Classify each variable Xi relative to S:
//   - mandatory: domain(Xi) ⊆ S  (Xi must count toward K)
//   - possible:  domain(Xi) ∩ S ≠ ∅ (Xi could count toward K)
//   - disjoint:  domain(Xi) ∩ S = ∅ (Xi cannot count toward K)
//   - Let m = |mandatory| and p = |possible|.
//     This implies the count must satisfy m ≤ count ≤ p.
//     Using the K-encoding, we prune K to [m+1 .. p+1].
//   - Tight bounds enable useful domain pruning on Xi:
//   - If m == maxCount (i.e., K.max-1), then all other variables that could be in S must be forced OUT of S: prune Xi := Xi \ S.
//   - If p == minCount (i.e., K.min-1), then all variables that could be in S must be forced INTO S: prune Xi := Xi ∩ S.
//
// This filtering is sound and efficient; it mirrors classical Among propagation used in CP.
// Stronger propagation (e.g., generalized arc consistency using flows) is possible but beyond scope;
// this implementation integrates cleanly with the solver's fixed-point loop and avoids technical debt.
package minikanren

import "fmt"

// Among is a global constraint that counts how many variables take values from S.
type Among struct {
	vars []*FDVariable
	set  Domain      // bitset mask for S over [1..maxV]
	k    *FDVariable // encoded count: value = count+1
}

// NewAmong creates an Among(vars, S, K) constraint.
//
// Parameters:
//   - vars: non-empty list of variables
//   - values: the explicit set S of allowed values (each in [1..maxValue]); duplicates are ignored
//   - k: the encoded count variable with domain in [1..len(vars)+1]
func NewAmong(vars []*FDVariable, values []int, k *FDVariable) (PropagationConstraint, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("Among: vars must be non-empty")
	}
	if k == nil {
		return nil, fmt.Errorf("Among: K must not be nil")
	}
	// Determine a safe maxValue for the set mask: cover all vars and values
	maxV := 0
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("Among: nil variable at index %d", i)
		}
		if v.Domain() == nil || v.Domain().MaxValue() <= 0 {
			return nil, fmt.Errorf("Among: variable %d has invalid domain", i)
		}
		if mv := v.Domain().MaxValue(); mv > maxV {
			maxV = mv
		}
	}
	for _, val := range values {
		if val <= 0 {
			return nil, fmt.Errorf("Among: value %d not in positive range", val)
		}
		if val > maxV {
			maxV = val
		}
	}
	if maxV <= 0 {
		return nil, fmt.Errorf("Among: could not determine positive maxValue")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("Among: values set must be non-empty")
	}

	// Build set domain mask
	setMask := NewBitSetDomainFromValues(maxV, values)
	if setMask.Count() == 0 {
		return nil, fmt.Errorf("Among: values set is empty after filtering")
	}

	// Defensive check for K domain
	if k.Domain() == nil || k.Domain().MaxValue() <= 0 {
		return nil, fmt.Errorf("Among: K has invalid domain")
	}

	vv := make([]*FDVariable, len(vars))
	copy(vv, vars)
	return &Among{vars: vv, set: setMask, k: k}, nil
}

// Variables returns all variables involved (vars plus K).
func (a *Among) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(a.vars)+1)
	out = append(out, a.vars...)
	out = append(out, a.k)
	return out
}

// Type names the constraint.
func (a *Among) Type() string { return "Among" }

// String returns a human-readable description.
func (a *Among) String() string {
	return fmt.Sprintf("Among(|vars|=%d, |S|=%d)", len(a.vars), a.set.Count())
}

// Propagate enforces bounds-consistent pruning for Among.
func (a *Among) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Among.Propagate: nil solver")
	}
	n := len(a.vars)
	if n == 0 {
		return state, nil
	}

	// Read domains
	dx := make([]Domain, n)
	for i := 0; i < n; i++ {
		dx[i] = solver.GetDomain(state, a.vars[i].ID())
		if dx[i] == nil {
			return nil, fmt.Errorf("Among: nil domain at index %d", i)
		}
		if dx[i].Count() == 0 {
			return nil, fmt.Errorf("Among: empty domain at index %d", i)
		}
	}
	dk := solver.GetDomain(state, a.k.ID())
	if dk == nil || dk.Count() == 0 {
		return nil, fmt.Errorf("Among: invalid or empty K domain")
	}

	// Compute mandatory and possible counts
	mandatory := 0
	possible := 0
	subset := make([]bool, n)
	mayIn := make([]bool, n)

	for i := 0; i < n; i++ {
		// Determine mayIn and subset via membership checks to avoid Intersect size assumptions
		count := 0
		inCount := 0
		dx[i].IterateValues(func(v int) {
			count++
			if a.set.Has(v) {
				inCount++
			}
		})
		if inCount > 0 {
			mayIn[i] = true
			possible++
		}
		if inCount == count && count > 0 { // domain(Xi) ⊆ S
			subset[i] = true
			mandatory++
		}
	}

	// K encodes count+1
	kMinAllowed := mandatory
	kMaxAllowed := possible
	if kMinAllowed > kMaxAllowed {
		return nil, fmt.Errorf("Among: infeasible bounds m=%d > p=%d", kMinAllowed, kMaxAllowed)
	}

	// Prune K to [m+1 .. p+1]
	kLo := kMinAllowed + 1
	kHi := kMaxAllowed + 1
	newDk := dk
	if dk.Min() < kLo {
		newDk = newDk.RemoveBelow(kLo)
	}
	if newDk.Count() == 0 {
		return nil, fmt.Errorf("Among: pruning K below %d empties domain", kLo)
	}
	if newDk.Max() > kHi {
		newDk = newDk.RemoveAbove(kHi)
	}
	if newDk.Count() == 0 {
		return nil, fmt.Errorf("Among: pruning K above %d empties domain", kHi)
	}

	newState := state
	if !newDk.Equal(dk) {
		var changed bool
		newState, changed = solver.SetDomain(newState, a.k.ID(), newDk)
		if changed {
			dk = newDk
		}
	}

	// After K pruning, recompute effective min/max counts
	currMin := dk.Min() - 1
	currMax := dk.Max() - 1

	// Temporary debug logging removed after stabilization

	// If m == currMax, all optional-but-mayIn must be OUT of S
	if mandatory == currMax {
		for i := 0; i < n; i++ {
			if mayIn[i] && !subset[i] {
				// prune S from Xi by removing each value in S explicitly
				nd := dx[i]
				// Iterate values of S and remove when present in Xi
				a.set.IterateValues(func(v int) {
					if nd.Has(v) {
						nd = nd.Remove(v)
					}
				})
				if nd.Count() == 0 {
					return nil, fmt.Errorf("Among: pruning Xi\\S at %d empties domain", i)
				}
				if !nd.Equal(dx[i]) {
					var changed bool
					newState, changed = solver.SetDomain(newState, a.vars[i].ID(), nd)
					if changed {
						dx[i] = nd
					}
				}
			}
		}
	} else if possible == currMin { // else-if: avoid conflicting simultaneous actions
		// If p == currMin, all mayIn must be forced INTO S
		for i := 0; i < n; i++ {
			if mayIn[i] && !subset[i] {
				// Xi := Xi ∩ S
				nd := dx[i].Intersect(a.set)
				if nd.Count() == 0 {
					return nil, fmt.Errorf("Among: forcing Xi∩S at %d empties domain", i)
				}
				if !nd.Equal(dx[i]) {
					var changed bool
					newState, changed = solver.SetDomain(newState, a.vars[i].ID(), nd)
					if changed {
						dx[i] = nd
					}
				}
			}
		}
	}

	return newState, nil
}
