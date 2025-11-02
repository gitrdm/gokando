// Package minikanren: global constraint - Regular (DFA constraint)
//
// Regular enforces that a sequence of FD variables (x1, x2, ..., xn)
// forms a word accepted by a given deterministic finite automaton (DFA).
//
// Contract (1-based, positive integers):
//   - States are numbered 1..numStates. State 0 is reserved for "no transition".
//   - Alphabet symbols are positive integers; a value v outside the transition
//     table's width is treated as having no transition from any state.
//   - delta is a transition table where delta[s][v] = t gives the next state t
//     from state s consuming symbol v. A value of 0 denotes the absence of a
//     transition.
//
// Propagation (bounds/GAC over the DFA using forward/backward filtering):
//  1. Forward pass: compute reachable states Fi after each position i using
//     current domains. Early fail if Fi becomes empty.
//  2. Backward pass: start from accepting states intersect Fi at i=n, then for
//     i=n..1, compute predecessor states Bi-1 and, simultaneously, collect the
//     set of supported symbols for xi using only transitions consistent with
//     Fi-1 and Bi.
//  3. Prune each xi to its supported symbols. If any domain empties, signal
//     inconsistency.
//
// This achieves strong pruning typical of the classic Regular constraint
// (Pesant 2004) and composes well with other constraints in the solver's
// fixed-point loop.
package minikanren

import (
	"fmt"
)

// Regular is the DFA-based global constraint over a sequence of variables.
type Regular struct {
	vars        []*FDVariable
	numStates   int
	start       int
	accept      []bool  // length = numStates+1, 1-based indexing
	delta       [][]int // 1-based: delta[s][v] -> next state in [1..numStates] or 0
	alphabetMax int     // maximum symbol index covered by delta rows
}

// NewRegular constructs a Regular constraint over vars with a DFA.
//
// Parameters:
//   - vars: non-empty slice of FD variables (non-nil), the sequence x1..xn
//   - numStates: number of DFA states (>=1), states are 1..numStates
//   - start: start state in [1..numStates]
//   - acceptStates: list of accepting states (each in [1..numStates], may repeat)
//   - delta: transition table; must have numStates rows; each row length must be
//     identical; entry 0 denotes no transition; positive entries must be
//     in [1..numStates]. Symbols are 1..alphabetMax where alphabetMax is
//     len(delta[row])-1.
func NewRegular(vars []*FDVariable, numStates, start int, acceptStates []int, delta [][]int) (*Regular, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("Regular: vars cannot be empty")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("Regular: vars[%d] is nil", i)
		}
	}
	if numStates < 1 {
		return nil, fmt.Errorf("Regular: numStates must be >= 1")
	}
	if start < 1 || start > numStates {
		return nil, fmt.Errorf("Regular: start state %d out of range [1..%d]", start, numStates)
	}
	if len(acceptStates) == 0 {
		return nil, fmt.Errorf("Regular: acceptStates cannot be empty")
	}
	if len(delta) != numStates {
		return nil, fmt.Errorf("Regular: delta must have %d rows (states), got %d", numStates, len(delta))
	}
	rowWidth := -1
	for s := 0; s < numStates; s++ {
		if len(delta[s]) == 0 {
			return nil, fmt.Errorf("Regular: delta[%d] row is empty", s+1)
		}
		if rowWidth == -1 {
			rowWidth = len(delta[s])
		} else if len(delta[s]) != rowWidth {
			return nil, fmt.Errorf("Regular: delta rows must have equal length; row %d has %d, expected %d", s+1, len(delta[s]), rowWidth)
		}
		for v := 1; v < len(delta[s]); v++ { // v=0 is unused placeholder for 1-based symbols
			ns := delta[s][v]
			if ns < 0 || ns > numStates {
				return nil, fmt.Errorf("Regular: delta[%d][%d]=%d out of range [0..%d]", s+1, v, ns, numStates)
			}
		}
	}
	// Build accepting state bitmap
	accept := make([]bool, numStates+1)
	for _, a := range acceptStates {
		if a < 1 || a > numStates {
			return nil, fmt.Errorf("Regular: accept state %d out of range [1..%d]", a, numStates)
		}
		accept[a] = true
	}

	return &Regular{
		vars:        vars,
		numStates:   numStates,
		start:       start,
		accept:      accept,
		delta:       delta,
		alphabetMax: rowWidth - 1, // account for 1-based symbol indexing
	}, nil
}

// Variables implements ModelConstraint.
func (r *Regular) Variables() []*FDVariable { return r.vars }

// Type implements ModelConstraint.
func (r *Regular) Type() string { return "Regular" }

// String implements ModelConstraint.
func (r *Regular) String() string {
	return fmt.Sprintf("Regular(len=%d, states=%d, start=%d)", len(r.vars), r.numStates, r.start)
}

// Propagate applies forward/backward DFA filtering to prune variable domains.
// Implements PropagationConstraint.
func (r *Regular) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Regular.Propagate: nil solver")
	}
	n := len(r.vars)
	if n == 0 {
		return state, nil
	}

	// Load domains and validate non-empty
	doms := make([]Domain, n)
	for i, v := range r.vars {
		d := solver.GetDomain(state, v.ID())
		if d == nil || d.Count() == 0 {
			return nil, fmt.Errorf("Regular: variable %d has empty domain", v.ID())
		}
		doms[i] = d
	}

	// Forward reachable states Fi (0..n). Fi is a bitmap of states after position i.
	F := make([][]bool, n+1)
	for i := 0; i <= n; i++ {
		F[i] = make([]bool, r.numStates+1)
	}
	F[0][r.start] = true

	for i := 1; i <= n; i++ {
		// For each reachable state at i-1 and each feasible symbol at xi, advance.
		doms[i-1].IterateValues(func(sym int) {
			if sym < 1 || sym > r.alphabetMax {
				// No transition defined; symbol cannot contribute to reachability.
				return
			}
			for s := 1; s <= r.numStates; s++ {
				if !F[i-1][s] {
					continue
				}
				ns := r.delta[s-1][sym] // internal rows are 0-based indexed; states are 1-based
				if ns != 0 {
					F[i][ns] = true
				}
			}
		})
		// Early failure: no states reachable after position i
		any := false
		for s := 1; s <= r.numStates; s++ {
			if F[i][s] {
				any = true
				break
			}
		}
		if !any {
			return nil, fmt.Errorf("Regular: no reachable states at position %d", i)
		}
	}

	// Backward acceptable states Bi (at position i) and pruning
	B := make([][]bool, n+1)
	for i := 0; i <= n; i++ {
		B[i] = make([]bool, r.numStates+1)
	}

	// Initialize at position n: accepting states that are also forward-reachable
	anyAccept := false
	for s := 1; s <= r.numStates; s++ {
		if r.accept[s] && F[n][s] {
			B[n][s] = true
			anyAccept = true
		}
	}
	if !anyAccept {
		return nil, fmt.Errorf("Regular: no accepting state reachable at end")
	}

	newState := state
	// Work backwards and collect supports for each position
	for i := n; i >= 1; i-- {
		// Track supported symbols for xi
		supported := make(map[int]bool)

		// Compute B[i-1] and supports using only transitions consistent with F and B
		for s := 1; s <= r.numStates; s++ {
			if !F[i-1][s] { // state must be forward-reachable at i-1
				continue
			}
			// For each symbol in domain of xi
			doms[i-1].IterateValues(func(sym int) {
				if sym < 1 || sym > r.alphabetMax {
					return
				}
				ns := r.delta[s-1][sym]
				if ns != 0 && B[i][ns] { // must lead to a backward-acceptable state at i
					supported[sym] = true
					B[i-1][s] = true
				}
			})
		}

		// Prune xi to supported symbols
		if len(supported) == 0 {
			return nil, fmt.Errorf("Regular: no supported symbols at position %d", i)
		}
		vals := make([]int, 0, len(supported))
		for v := range supported {
			vals = append(vals, v)
		}
		suppDom := NewBitSetDomainFromValues(doms[i-1].MaxValue(), vals)
		pruned := doms[i-1].Intersect(suppDom)
		if pruned.Count() == 0 {
			return nil, fmt.Errorf("Regular: domain of var %d emptied at position %d", r.vars[i-1].ID(), i)
		}
		if !pruned.Equal(doms[i-1]) {
			newState, _ = solver.SetDomain(newState, r.vars[i-1].ID(), pruned)
			doms[i-1] = pruned // keep local view in sync for subsequent positions
		}

		// Early failure: B[i-1] must be non-empty to proceed further
		any := false
		for s := 1; s <= r.numStates; s++ {
			if B[i-1][s] {
				any = true
				break
			}
		}
		if !any {
			return nil, fmt.Errorf("Regular: no backward states at position %d", i-1)
		}
	}

	return newState, nil
}
