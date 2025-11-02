// Package minikanren provides the Stretch global constraint.
//
// Stretch(vars, values, minLen, maxLen) constrains run lengths of values along
// a sequence of FD variables. For each value v in values, every maximal run of
// consecutive occurrences of v must have a length between minLen[v] and
// maxLen[v] (inclusive). Values not listed in 'values' are unconstrained by
// default (equivalent to minLen=1, maxLen=len(vars)).
//
// Implementation strategy: DFA via Regular
//   - Build a deterministic finite automaton whose states encode
//     "currently in a run of value v of length c" for c ∈ [1..maxLen[v]].
//   - Transitions:
//   - From start: on symbol v → state (v,1)
//   - From (v,c) on symbol v:
//     if c < maxLen[v], go to (v,c+1); else no transition (forbid > max)
//   - From (v,c) on symbol w ≠ v:
//     allowed iff c ≥ minLen[v], then go to (w,1); else no transition
//   - Accepting states are exactly those (v,c) with c ≥ minLen[v], ensuring that
//     the final run also satisfies its minimum length.
//
// This reduction achieves strong propagation using the existing Regular
// constraint (forward/backward DFA filtering), composes cleanly with other
// constraints, and avoids technical debt.
package minikanren

import "fmt"

// Stretch is a thin wrapper around the constructed Regular constraint to expose
// the high-level intent and variables involved.
type Stretch struct {
	vars       []*FDVariable
	values     []int // values explicitly parameterized
	minByValue map[int]int
	maxByValue map[int]int
	dfa        *Regular // underlying DFA constraint
}

// NewStretch constructs Stretch(vars, values, minLen, maxLen).
//
// Parameters:
//   - model: hosting model (non-nil)
//   - vars: non-empty sequence of variables (positive domains)
//   - values: distinct positive values to constrain explicitly
//   - minLen, maxLen: same length as values; for each i, enforce
//     minLen[i] ≤ run length of values[i] ≤ maxLen[i]
//     Values not listed inherit defaults: minLen=1, maxLen=len(vars).
func NewStretch(model *Model, vars []*FDVariable, values []int, minLen []int, maxLen []int) (*Stretch, error) {
	if model == nil {
		return nil, fmt.Errorf("NewStretch: model cannot be nil")
	}
	n := len(vars)
	if n == 0 {
		return nil, fmt.Errorf("NewStretch: vars cannot be empty")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("NewStretch: vars[%d] is nil", i)
		}
		if v.Domain() == nil || v.Domain().MaxValue() <= 0 {
			return nil, fmt.Errorf("NewStretch: vars[%d] has invalid domain", i)
		}
	}
	if len(values) != len(minLen) || len(values) != len(maxLen) {
		return nil, fmt.Errorf("NewStretch: values, minLen, maxLen must have equal length")
	}
	// Deduplicate values and validate lengths
	seen := map[int]int{} // value -> index in compact arrays
	compactVals := make([]int, 0, len(values))
	compactMin := make([]int, 0, len(values))
	compactMax := make([]int, 0, len(values))
	for i, v := range values {
		if v <= 0 {
			return nil, fmt.Errorf("NewStretch: values[%d]=%d must be positive", i, v)
		}
		mn := minLen[i]
		mx := maxLen[i]
		if mn < 1 || mx < 1 || mn > mx {
			return nil, fmt.Errorf("NewStretch: invalid lengths for value %d: min=%d max=%d", v, mn, mx)
		}
		if mn > n || mx > n {
			return nil, fmt.Errorf("NewStretch: run lengths for value %d exceed sequence length n=%d: min=%d max=%d", v, n, mn, mx)
		}
		if _, ok := seen[v]; !ok {
			seen[v] = len(compactVals)
			compactVals = append(compactVals, v)
			compactMin = append(compactMin, mn)
			compactMax = append(compactMax, mx)
		} else {
			// If duplicates appear, enforce they are consistent
			idx := seen[v]
			if compactMin[idx] != mn || compactMax[idx] != mx {
				return nil, fmt.Errorf("NewStretch: duplicate constraints for value %d with conflicting lengths", v)
			}
		}
	}

	// Determine symbol universe to cover: union(domains) ∪ values
	widthMax := 0
	present := map[int]bool{}
	for _, x := range vars {
		d := x.Domain()
		if mv := d.MaxValue(); mv > widthMax {
			widthMax = mv
		}
		d.IterateValues(func(v int) { present[v] = true })
	}
	for _, v := range compactVals {
		if v > widthMax {
			widthMax = v
		}
		present[v] = true
	}
	if widthMax == 0 {
		return nil, fmt.Errorf("NewStretch: could not determine alphabet width")
	}

	// Build per-value min/max maps with defaults
	minBy := make(map[int]int, len(present))
	maxBy := make(map[int]int, len(present))
	for v := range present {
		minBy[v] = 1
		maxBy[v] = n
	}
	for i, v := range compactVals {
		minBy[v] = compactMin[i]
		maxBy[v] = compactMax[i]
	}

	// Number DFA states: 1 start state + sum_v maxBy[v] for v in present
	startID := 1
	nextID := startID + 1
	idx := make(map[[2]int]int) // (v,c) -> state id
	orderedVals := make([]int, 0, len(present))
	for v := range present {
		orderedVals = append(orderedVals, v)
	}
	// We don't require a stable order as long as idx mapping is consistent.
	// Assign state ids for each run counter c in [1..maxBy[v]]
	for _, v := range orderedVals {
		for c := 1; c <= maxBy[v]; c++ {
			idx[[2]int{v, c}] = nextID
			nextID++
		}
	}
	numStates := nextID - 1

	// Build transition table (1-based states; columns 0..widthMax with 0 unused)
	delta := make([][]int, numStates)
	for s := 0; s < numStates; s++ {
		row := make([]int, widthMax+1)
		// row[0] remains 0 as placeholder
		delta[s] = row
	}

	// Start transitions: on any present symbol v, go to (v,1)
	for sym := 1; sym <= widthMax; sym++ {
		if present[sym] {
			delta[startID-1][sym] = idx[[2]int{sym, 1}]
		}
	}

	// Transitions from (v,c)
	for key, stateID := range idx {
		v := key[0]
		c := key[1]
		mn := minBy[v]
		mx := maxBy[v]
		row := delta[stateID-1]
		for sym := 1; sym <= widthMax; sym++ {
			if !present[sym] {
				continue // no transition; symbol cannot occur
			}
			if sym == v {
				if c < mx {
					row[sym] = idx[[2]int{v, c + 1}]
				} else {
					// c == mx ⇒ cannot extend run of v further
					row[sym] = 0
				}
			} else {
				// Switching value allowed only if current run meets the minimum
				if c >= mn {
					row[sym] = idx[[2]int{sym, 1}]
				} else {
					row[sym] = 0
				}
			}
		}
	}

	// Accepting states: all (v,c) with c ≥ minBy[v]
	accept := make([]int, 0, len(idx))
	for key, stateID := range idx {
		v := key[0]
		c := key[1]
		if c >= minBy[v] {
			accept = append(accept, stateID)
		}
	}

	// Construct Regular and register constraints
	// Sequence variables are the given vars in order
	reg, err := NewRegular(vars, numStates, startID, accept, delta)
	if err != nil {
		return nil, fmt.Errorf("NewStretch: Regular construction failed: %w", err)
	}

	// Keep an owned copy of vars slice for introspection
	vv := make([]*FDVariable, n)
	copy(vv, vars)
	cc := &Stretch{
		vars:       vv,
		values:     append([]int(nil), compactVals...),
		minByValue: minBy,
		maxByValue: maxBy,
		dfa:        reg,
	}

	// Add sub-constraint and wrapper to the model
	model.AddConstraint(reg)
	model.AddConstraint(cc)
	return cc, nil
}

// Variables returns the sequence variables.
func (s *Stretch) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(s.vars))
	out = append(out, s.vars...)
	return out
}

// Type returns the constraint type name.
func (s *Stretch) Type() string { return "Stretch" }

// String returns a human-readable description.
func (s *Stretch) String() string {
	return fmt.Sprintf("Stretch(n=%d, |vals|=%d)", len(s.vars), len(s.values))
}

// Propagate is a no-op for the wrapper; pruning is performed by the Regular DFA.
func (s *Stretch) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}
