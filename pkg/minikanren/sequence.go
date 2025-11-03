// Package minikanren provides the Sequence global constraint.
//
// Sequence(vars, S, k, minCount, maxCount) enforces that in every sliding
// window of length k over vars, the number of variables taking a value in S
// is between minCount and maxCount (inclusive).
//
// Implementation uses composition over existing primitives:
//   - For each Xi, create a boolean bi reifying Xi ∈ S via InSetReified
//   - For each window i..i+k-1, post BoolSum(b[i..i+k-1], totalWin)
//     with totalWin domain set to [minCount+1 .. maxCount+1]
//
// This achieves safe bounds-consistent propagation. Stronger filters (e.g.,
// sequential counters) can be layered later without API changes.
package minikanren

import "fmt"

type Sequence struct {
	vars     []*FDVariable
	set      []int
	k        int
	minCount int
	maxCount int
	b        []*FDVariable
	reifs    []PropagationConstraint
	windows  []PropagationConstraint
}

// NewSequence constructs the Sequence constraint.
func NewSequence(model *Model, vars []*FDVariable, setValues []int, windowLen, minCount, maxCount int) (*Sequence, error) {
	if model == nil {
		return nil, fmt.Errorf("NewSequence: model cannot be nil")
	}
	n := len(vars)
	if n == 0 {
		return nil, fmt.Errorf("NewSequence: vars cannot be empty")
	}
	if windowLen <= 0 || windowLen > n {
		return nil, fmt.Errorf("NewSequence: windowLen must be in [1..%d]", n)
	}
	if minCount < 0 || maxCount < 0 || minCount > maxCount || maxCount > windowLen {
		return nil, fmt.Errorf("NewSequence: require 0≤minCount≤maxCount≤windowLen; got min=%d max=%d k=%d", minCount, maxCount, windowLen)
	}
	if len(setValues) == 0 {
		return nil, fmt.Errorf("NewSequence: setValues cannot be empty")
	}

	// Deduplicate set values and validate positivity
	seen := map[int]struct{}{}
	S := make([]int, 0, len(setValues))
	for _, v := range setValues {
		if v < 1 {
			return nil, fmt.Errorf("NewSequence: set values must be positive, got %d", v)
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			S = append(S, v)
		}
	}

	boolDom := NewBitSetDomain(2)
	b := make([]*FDVariable, n)
	reifs := make([]PropagationConstraint, n)
	for i, x := range vars {
		if x == nil {
			return nil, fmt.Errorf("NewSequence: vars[%d] is nil", i)
		}
		bi := model.NewVariable(boolDom)
		b[i] = bi
		r, err := NewInSetReified(x, S, bi)
		if err != nil {
			return nil, fmt.Errorf("NewSequence: InSetReified[%d]: %w", i, err)
		}
		reifs[i] = r
		model.AddConstraint(r)
	}

	// Per-window BoolSum with total domain [min+1 .. max+1]
	totalDom := NewBitSetDomain(windowLen + 1).RemoveBelow(minCount + 1).RemoveAbove(maxCount + 1)
	windows := make([]PropagationConstraint, 0, n-windowLen+1)
	for i := 0; i+windowLen <= n; i++ {
		total := model.NewVariable(totalDom)
		sum, err := NewBoolSum(b[i:i+windowLen], total)
		if err != nil {
			return nil, fmt.Errorf("NewSequence: BoolSum window %d: %w", i, err)
		}
		windows = append(windows, sum)
		model.AddConstraint(sum)
	}

	vv := make([]*FDVariable, n)
	copy(vv, vars)
	c := &Sequence{vars: vv, set: S, k: windowLen, minCount: minCount, maxCount: maxCount, b: b, reifs: reifs, windows: windows}
	model.AddConstraint(c)
	return c, nil
}

func (s *Sequence) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(s.vars))
	out = append(out, s.vars...)
	return out
}
func (s *Sequence) Type() string { return "Sequence" }
func (s *Sequence) String() string {
	return fmt.Sprintf("Sequence(n=%d,k=%d,[%d..%d])", len(s.vars), s.k, s.minCount, s.maxCount)
}
func (s *Sequence) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}
