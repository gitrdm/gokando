// Package minikanren: global constraint - Table (extensional constraint)
//
// Table enforces that the n-tuple of FD variables (vars[0],...,vars[n-1])
// must be exactly equal to one of the rows in a fixed table of allowed tuples.
//
// Propagation (generalized arc consistency over the fixed table in one pass):
//  1. Discard any table row that is incompatible with current domains.
//  2. For each variable i, collect the set of values that appear at column i in
//     at least one remaining compatible row (a support).
//  3. Prune each variable's domain to the supported set.
//
// Notes
//   - Tuples must be positive integers to respect Domain invariants (1-based).
//   - Rows may contain repeated values; rows may be duplicated; both are handled.
//   - Propagation is monotonic; if pruning happens, the solver will call this
//     constraint again during the fixed-point loop for further pruning.
//   - If no compatible rows remain, the constraint signals inconsistency.
package minikanren

import (
	"fmt"
)

// Table is an extensional constraint over a fixed list of allowed tuples.
type Table struct {
	vars []*FDVariable
	rows [][]int // len of each row equals len(vars)
}

// NewTable constructs a new Table constraint given variables and allowed rows.
//
// Contract:
// - len(vars) > 0, all vars non-nil
// - len(rows) > 0, each row has exactly len(vars) entries
// - All row values are >= 1
func NewTable(vars []*FDVariable, rows [][]int) (*Table, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("Table: vars cannot be empty")
	}
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("Table: vars[%d] is nil", i)
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("Table: rows cannot be empty")
	}
	arity := len(vars)
	// deep copy for immutability
	copied := make([][]int, len(rows))
	for r, row := range rows {
		if len(row) != arity {
			return nil, fmt.Errorf("Table: row %d has arity %d, expected %d", r, len(row), arity)
		}
		copied[r] = make([]int, arity)
		copy(copied[r], row)
		// validate positive values
		for c, val := range copied[r] {
			if val < 1 {
				return nil, fmt.Errorf("Table: row %d col %d has non-positive value %d", r, c, val)
			}
		}
	}
	return &Table{vars: vars, rows: copied}, nil
}

// Variables returns the involved variables. Implements ModelConstraint.
func (t *Table) Variables() []*FDVariable { return t.vars }

// Type returns the constraint identifier. Implements ModelConstraint.
func (t *Table) Type() string { return "Table" }

// String returns a human-readable description. Implements ModelConstraint.
func (t *Table) String() string {
	return fmt.Sprintf("Table(arity=%d, rows=%d)", len(t.vars), len(t.rows))
}

// Propagate enforces generalized arc consistency against the extensional table.
// Implements PropagationConstraint.
func (t *Table) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Table.Propagate: nil solver")
	}

	n := len(t.vars)
	doms := make([]Domain, n)
	for i, v := range t.vars {
		d := solver.GetDomain(state, v.ID())
		if d == nil || d.Count() == 0 {
			return nil, fmt.Errorf("Table: variable %d has empty domain", v.ID())
		}
		doms[i] = d
	}

	// Determine compatible rows under current domains and collect supports.
	supported := make([][]int, n) // per variable values with a support
	compatibleRows := 0
	for _, row := range t.rows {
		// Check row compatibility: every column value must be in its domain
		ok := true
		for i := 0; i < n; i++ {
			if !doms[i].Has(row[i]) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		compatibleRows++
		// Record supports
		for i := 0; i < n; i++ {
			supported[i] = append(supported[i], row[i])
		}
	}

	if compatibleRows == 0 {
		return nil, fmt.Errorf("Table: no compatible rows remain (inconsistency)")
	}

	cur := state
	// Intersect each domain with its supported values
	for i, v := range t.vars {
		// If no specific supports were gathered for i (shouldn't happen when compatibleRows>0),
		// then there is a logic error; guard defensively.
		if len(supported[i]) == 0 {
			return nil, fmt.Errorf("Table: internal error - variable %d has no supports despite compatible rows", v.ID())
		}
		suppDom := NewBitSetDomainFromValues(doms[i].MaxValue(), supported[i])
		newDom := doms[i].Intersect(suppDom)
		if newDom.Count() == 0 {
			return nil, fmt.Errorf("Table: domain of var %d emptied by table filtering", v.ID())
		}
		if !newDom.Equal(doms[i]) {
			cur, _ = solver.SetDomain(cur, v.ID(), newDom)
		}
	}

	return cur, nil
}
