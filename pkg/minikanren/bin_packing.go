// Package minikanren provides a composition-based BinPacking global constraint.
//
// BinPacking assigns each item i to one of m bins with capacity cap[k], and
// enforces that, for every bin k, the total size of items assigned to k does
// not exceed cap[k]. Items are represented by FD variables x[i] with domains
// in {1..m} (bin indices). Sizes are positive integers.
//
// Implementation uses reified assignment booleans and a weighted sum:
//   - For each bin k, create booleans b[i,k] ↔ (x[i] == k)
//   - For each bin k, compute: load_k = Σ size[i] * (b[i,k] - 1)
//     Note: booleans are {1=false, 2=true}. (b-1) turns them into {0,1}
//   - We implement Σ size[i]*b[i,k] as a LinearSum to a total variable sum_k,
//     then tie sum_k and the encoded load LkPlus1 via Arithmetic:
//     sum_k = LkPlus1 + (base_k - 1), where base_k = Σ size[i]
//     and domain(LkPlus1) ⊆ [1..cap[k]+1]. This guarantees load ≤ cap[k].
//
// This construction achieves safe bounds-consistent propagation using existing
// primitives. Stronger propagation (e.g., subset sum reasoning) can be layered
// later without changing the API.
package minikanren

import "fmt"

type BinPacking struct {
	items      []*FDVariable
	sizes      []int
	capacities []int
	m          int
	// per-bin artifacts (for introspection)
	binBools [][]*FDVariable // b[i][k]
	binSums  []*FDVariable   // sum_k variables (Σ size[i]*b[i,k])
	binLoads []*FDVariable   // LkPlus1 variables encoding load+1 ≤ cap+1
	reifs    [][]PropagationConstraint
	sums     []PropagationConstraint
	ties     []PropagationConstraint // Arithmetic links load to sum
}

// NewBinPacking constructs the capacity constraints for m bins.
//
// Parameters:
//   - model: hosting model
//   - items: variables with domains ⊆ {1..m}
//   - sizes: positive integers (len = len(items))
//   - capacities: positive integers (len = m)
func NewBinPacking(model *Model, items []*FDVariable, sizes []int, capacities []int) (*BinPacking, error) {
	if model == nil {
		return nil, fmt.Errorf("NewBinPacking: model cannot be nil")
	}
	n := len(items)
	if n == 0 {
		return nil, fmt.Errorf("NewBinPacking: items cannot be empty")
	}
	if len(sizes) != n {
		return nil, fmt.Errorf("NewBinPacking: len(sizes) != len(items)")
	}
	m := len(capacities)
	if m == 0 {
		return nil, fmt.Errorf("NewBinPacking: capacities cannot be empty")
	}
	for i, it := range items {
		if it == nil {
			return nil, fmt.Errorf("NewBinPacking: items[%d] is nil", i)
		}
	}
	for i, s := range sizes {
		if s <= 0 {
			return nil, fmt.Errorf("NewBinPacking: sizes[%d] must be positive, got %d", i, s)
		}
	}
	for k, c := range capacities {
		if c < 0 {
			return nil, fmt.Errorf("NewBinPacking: capacities[%d] must be ≥0, got %d", k, c)
		}
	}

	// Allocate structures
	boolDom := NewBitSetDomain(2)
	binBools := make([][]*FDVariable, m)
	reifs := make([][]PropagationConstraint, m)
	binSums := make([]*FDVariable, m)
	binLoads := make([]*FDVariable, m)
	sums := make([]PropagationConstraint, m)
	ties := make([]PropagationConstraint, m)

	// Precompute base = sum(sizes) and maxCap
	base := 0
	maxCap := 0
	for _, s := range sizes {
		base += s
	}
	for _, c := range capacities {
		if c > maxCap {
			maxCap = c
		}
	}

	// Prepare coefficients shared across bins
	coeffs := make([]int, n)
	copy(coeffs, sizes)

	for k := 1; k <= m; k++ {
		// Build booleans and reify x[i]==k
		rowB := make([]*FDVariable, n)
		rowR := make([]PropagationConstraint, n)
		for i, x := range items {
			b := model.NewVariable(boolDom)
			rowB[i] = b
			r, err := NewValueEqualsReified(x, k, b)
			if err != nil {
				return nil, fmt.Errorf("NewBinPacking: reify item %d==%d: %w", i, k, err)
			}
			rowR[i] = r
			model.AddConstraint(r)
		}
		binBools[k-1] = rowB
		reifs[k-1] = rowR

		// sum_k = Σ size[i]*b[i,k]
		// Domain upper bound needs to cover base + cap[k]. Using base+maxCap safely bounds all bins.
		sumK := model.NewVariable(NewBitSetDomain(base + maxCap))
		ls, err := NewLinearSum(rowB, coeffs, sumK)
		if err != nil {
			return nil, fmt.Errorf("NewBinPacking: LinearSum bin %d: %w", k, err)
		}
		binSums[k-1] = sumK
		sums[k-1] = ls
		model.AddConstraint(ls)

		// LkPlus1 ∈ [1..cap[k]+1], and sum_k = LkPlus1 + (base-1)
		LkPlus1 := model.NewVariable(NewBitSetDomain(capacities[k-1] + 1))
		binLoads[k-1] = LkPlus1
		tie, err := NewArithmetic(LkPlus1, sumK, base-1)
		if err != nil {
			return nil, fmt.Errorf("NewBinPacking: Arithmetic tie bin %d: %w", k, err)
		}
		ties[k-1] = tie
		model.AddConstraint(tie)
	}

	ii := make([]*FDVariable, n)
	copy(ii, items)
	ss := make([]int, n)
	copy(ss, sizes)
	cc := make([]int, m)
	copy(cc, capacities)
	bp := &BinPacking{items: ii, sizes: ss, capacities: cc, m: m, binBools: binBools, binSums: binSums, binLoads: binLoads, reifs: reifs, sums: sums, ties: ties}
	model.AddConstraint(bp)
	return bp, nil
}

func (bp *BinPacking) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(bp.items))
	out = append(out, bp.items...)
	return out
}
func (bp *BinPacking) Type() string { return "BinPacking" }
func (bp *BinPacking) String() string {
	return fmt.Sprintf("BinPacking(n=%d,m=%d)", len(bp.items), bp.m)
}
func (bp *BinPacking) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}
