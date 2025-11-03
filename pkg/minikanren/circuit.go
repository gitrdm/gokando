// Package minikanren: global constraints - Circuit (single Hamiltonian cycle)
//
// Circuit models a permutation of successors that forms a single cycle
// visiting all nodes exactly once (a Hamiltonian circuit). It is a classic
// global constraint used in routing and sequencing problems.
//
// Interface and semantics
// - Inputs: succ[1..n], where succ[i] is the successor node index of node i
// - Domains: succ[i] ⊆ {1..n} for all i
// - startIndex: distinguished start node in [1..n]
//
// Enforced relations
//  1. Exactly-one successor per node i (already implicit in a single-valued succ[i],
//     but we encode with reified booleans for strong propagation):
//     For each i, exactly one j has (succ[i] == j)
//  2. Exactly-one predecessor per node j:
//     For each j, exactly one i has (succ[i] == j)
//  3. No self-loops: succ[i] ≠ i
//  4. Subtour elimination via order variables u[1..n]:
//     - u[start] = 1, and for all k ≠ start: u[k] ∈ [2..n]
//     - For every arc (i -> j) with j ≠ start, if succ[i] == j then u[j] = u[i] + 1
//     (reified arithmetic). We deliberately do NOT enforce order on arcs leading
//     back to the start to avoid a wrap-around equality that would overconstrain
//     the Hamiltonian cycle.
//
// Construction strategy
// - Create boolean matrix b[i][j] reifying (succ[i] == j)
// - Post row and column BoolSum constraints enforcing exactly one true in each
// - Force b[i][i] = false to forbid self-loops
// - Create order variables u with domains {1} for start and {2..n} for others
// - For each (i, j ≠ start), post Reified(Arithmetic(u[i] + 1 = u[j]), b[i][j])
//
// Notes
//   - This approach uses O(n^2) auxiliary booleans and reified constraints,
//     which is standard and provides robust propagation without bespoke graph
//     algorithms. It integrates cleanly with the solver's immutable state.
package minikanren

import "fmt"

// Circuit is a composite global constraint that owns auxiliary variables and
// reified constraints to enforce a single Hamiltonian circuit over successors.
//
// The Propagate method itself does no work; all pruning is done by the posted
// sub-constraints. This mirrors the Count and ElementValues pattern.
type Circuit struct {
	succ       []*FDVariable
	startIndex int
	bools      [][]*FDVariable           // b[i][j] ∈ {1=false,2=true}
	rowSums    []PropagationConstraint   // exactly-one per row
	colSums    []PropagationConstraint   // exactly-one per column
	eqReifs    [][]PropagationConstraint // b[i][j] ↔ succ[i]==j
	orderVars  []*FDVariable             // u[1..n]
	orderReifs []PropagationConstraint   // Reified(Arithmetic(u[i]+1=u[j]), b[i][j]) for j!=start
}

// NewCircuit constructs a Circuit global constraint and posts all auxiliary
// variables and constraints into the model.
//
// Contract:
// - model != nil
// - len(succ) = n >= 2
// - startIndex in [1..n]
func NewCircuit(model *Model, succ []*FDVariable, startIndex int) (*Circuit, error) {
	if model == nil {
		return nil, fmt.Errorf("NewCircuit: model cannot be nil")
	}
	if len(succ) < 2 {
		return nil, fmt.Errorf("NewCircuit: need at least 2 successor variables")
	}
	for i, v := range succ {
		if v == nil {
			return nil, fmt.Errorf("NewCircuit: succ[%d] is nil", i)
		}
	}
	if startIndex < 1 || startIndex > len(succ) {
		return nil, fmt.Errorf("NewCircuit: startIndex %d out of range [1..%d]", startIndex, len(succ))
	}

	n := len(succ)

	// 1) Create boolean matrix b[i][j] reifying succ[i] == j
	bools := make([][]*FDVariable, n)
	eqReifs := make([][]PropagationConstraint, n)
	for i := 0; i < n; i++ {
		bools[i] = make([]*FDVariable, n)
		eqReifs[i] = make([]PropagationConstraint, n)
		for j := 1; j <= n; j++ {
			// Self-loop boolean forced to false {1}
			var b *FDVariable
			if j == i+1 {
				b = model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), fmt.Sprintf("b_%d_%d", i+1, j))
			} else {
				b = model.NewVariableWithName(NewBitSetDomain(2), fmt.Sprintf("b_%d_%d", i+1, j))
			}
			bools[i][j-1] = b

			// Reify equality: b[i][j] ↔ (succ[i] == j)
			reif, err := NewValueEqualsReified(succ[i], j, b)
			if err != nil {
				return nil, fmt.Errorf("NewCircuit: reify succ[%d]==%d failed: %w", i+1, j, err)
			}
			eqReifs[i][j-1] = reif
			model.AddConstraint(reif)
		}
	}

	// 2) Exactly-one per row and per column via BoolSum with total fixed to {2}
	rowSums := make([]PropagationConstraint, n)
	for i := 0; i < n; i++ {
		// total represents count+1, so {2} enforces exactly-one true
		total := model.NewVariableWithName(NewBitSetDomainFromValues(n+1, []int{2}), fmt.Sprintf("row_%d_total", i+1))
		rowSum, err := NewBoolSum(bools[i], total)
		if err != nil {
			return nil, fmt.Errorf("NewCircuit: row BoolSum[%d] failed: %w", i+1, err)
		}
		rowSums[i] = rowSum
		model.AddConstraint(rowSum)
	}

	colSums := make([]PropagationConstraint, n)
	for j := 0; j < n; j++ {
		col := make([]*FDVariable, n)
		for i := 0; i < n; i++ {
			col[i] = bools[i][j]
		}
		total := model.NewVariableWithName(NewBitSetDomainFromValues(n+1, []int{2}), fmt.Sprintf("col_%d_total", j+1))
		colSum, err := NewBoolSum(col, total)
		if err != nil {
			return nil, fmt.Errorf("NewCircuit: column BoolSum[%d] failed: %w", j+1, err)
		}
		colSums[j] = colSum
		model.AddConstraint(colSum)
	}

	// 3) Order variables u with u[start]=1, others in [2..n]
	orderVars := make([]*FDVariable, n)
	for k := 1; k <= n; k++ {
		var u *FDVariable
		if k == startIndex {
			u = model.NewVariableWithName(NewBitSetDomainFromValues(n, []int{1}), fmt.Sprintf("u_%d", k))
		} else {
			vals := make([]int, 0, n-1)
			for v := 2; v <= n; v++ {
				vals = append(vals, v)
			}
			u = model.NewVariableWithName(NewBitSetDomainFromValues(n, vals), fmt.Sprintf("u_%d", k))
		}
		orderVars[k-1] = u
	}

	// 4) Reified ordering: if b[i][j] true and j != start, enforce u[j] = u[i] + 1
	orderReifs := make([]PropagationConstraint, 0, n*n)
	for i := 0; i < n; i++ {
		for j := 1; j <= n; j++ {
			if j == startIndex {
				continue // skip arcs into start for ordering to avoid wrap-around
			}
			arith, err := NewArithmetic(orderVars[i], orderVars[j-1], 1)
			if err != nil {
				return nil, fmt.Errorf("NewCircuit: arithmetic u[%d]+1=u[%d] failed: %w", i+1, j, err)
			}
			reif, err := NewReifiedConstraint(arith, bools[i][j-1])
			if err != nil {
				return nil, fmt.Errorf("NewCircuit: reified arithmetic for (%d->%d) failed: %w", i+1, j, err)
			}
			orderReifs = append(orderReifs, reif)
			model.AddConstraint(reif)
		}
	}

	// Defensive copy of succ
	succCopy := make([]*FDVariable, n)
	copy(succCopy, succ)

	return &Circuit{
		succ:       succCopy,
		startIndex: startIndex,
		bools:      bools,
		rowSums:    rowSums,
		colSums:    colSums,
		eqReifs:    eqReifs,
		orderVars:  orderVars,
		orderReifs: orderReifs,
	}, nil
}

// Variables returns the primary decision variables for this global constraint.
// Implements ModelConstraint.
func (c *Circuit) Variables() []*FDVariable { return c.succ }

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (c *Circuit) Type() string { return "Circuit" }

// String returns a human-readable description.
// Implements ModelConstraint.
func (c *Circuit) String() string {
	return fmt.Sprintf("Circuit(n=%d, start=%d)", len(c.succ), c.startIndex)
}

// Propagate is a no-op: all pruning is handled by posted sub-constraints.
// Implements PropagationConstraint.
func (c *Circuit) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}
