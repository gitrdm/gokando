// Package minikanren provides the Diffn (2D non-overlap) global constraint.
//
// Differ from NoOverlap (1D disjunctive), Diffn enforces that axis-aligned
// rectangles do not overlap in the plane. For each rectangle i we have two
// finite-domain start variables X[i], Y[i] and fixed positive sizes W[i], H[i].
// Rectangles are closed-open on both axes: [X[i], X[i]+W[i)) × [Y[i], Y[i]+H[i)).
//
// Implementation strategy (production, composition-based):
//   - For each pair (i, j), post a disjunction that at least one of these holds:
//     1) X[i] + W[i] ≤ X[j]
//     2) X[j] + W[j] ≤ X[i]
//     3) Y[i] + H[i] ≤ Y[j]
//     4) Y[j] + H[j] ≤ Y[i]
//   - We construct each inequality using Arithmetic (offset helper) and
//     Inequality, then reify the inequality into a boolean with the generic
//     reifier. A BoolSum over the four booleans is constrained to have
//     domain [5..8] (since booleans are encoded {1=false,2=true}, a sum ≥5
//     guarantees at least one true among four).
//
// This decomposition favors correctness and integration with existing, well-
// tested primitives. It achieves safe bounds-consistent pruning and is commonly
// used as a baseline Diffn encoding. Stronger filtering (e.g., energy-based,
// edge-finding) can be layered later without changing the API.
package minikanren

import "fmt"

// Diffn composes reified pairwise non-overlap disjunctions for rectangles.
type Diffn struct {
	x, y  []*FDVariable
	w, h  []int
	reifs [][]*ReifiedConstraint // per-pair, four reified inequalities
}

// NewDiffn posts a 2D non-overlap constraint over rectangles defined by
// positions (x[i], y[i]) and fixed sizes (w[i], h[i]). All sizes must be ≥1.
func NewDiffn(model *Model, x, y []*FDVariable, w, h []int) (*Diffn, error) {
	if model == nil {
		return nil, fmt.Errorf("NewDiffn: model cannot be nil")
	}
	n := len(x)
	if n == 0 || len(y) != n || len(w) != n || len(h) != n {
		return nil, fmt.Errorf("NewDiffn: x, y, w, h must have same non-zero length")
	}
	for i := 0; i < n; i++ {
		if x[i] == nil || y[i] == nil {
			return nil, fmt.Errorf("NewDiffn: x[%d] or y[%d] is nil", i, i)
		}
		if w[i] <= 0 || h[i] <= 0 {
			return nil, fmt.Errorf("NewDiffn: sizes must be positive at %d (w=%d,h=%d)", i, w[i], h[i])
		}
	}

	boolDom := NewBitSetDomain(2) // {1,2}
	// For m=4 booleans, BoolSum total encodes count+1 in [1..5].
	// Require ≥1 true ⇒ total ∈ [2..5].
	atLeastOneTrueDom := NewBitSetDomainFromValues(5, []int{2, 3, 4, 5})

	// Helper to make offset var: zi = xi + offset
	makeOffset := func(base *FDVariable, offset int) (*FDVariable, *Arithmetic, error) {
		// z domain upper bound: base.Max()+offset, but we conservatively
		// use base domain's MaxValue and let propagation refine. We need a
		// valid domain size; over-approximate with same Max + offset.
		max := base.Domain().MaxValue()
		if max+offset < 1 {
			max = 1
		} else {
			max = max + offset
		}
		z := model.NewVariable(NewBitSetDomain(max))
		arith, err := NewArithmetic(base, z, offset)
		if err != nil {
			return nil, nil, err
		}
		model.AddConstraint(arith)
		return z, arith, nil
	}

	reifs := make([][]*ReifiedConstraint, n)
	for i := 0; i < n; i++ {
		reifs[i] = make([]*ReifiedConstraint, 0)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// Build the 4 disjuncts as reified inequalities
			var bools []*FDVariable
			var reifList []*ReifiedConstraint
			// 1) X[i] + W[i] ≤ X[j]
			xiPlus, _, err := makeOffset(x[i], w[i])
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: offset for x[%d]: %w", i, err)
			}
			ineq1, err := NewInequality(xiPlus, x[j], LessEqual)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: ineq1: %w", err)
			}
			b1 := model.NewVariable(boolDom)
			r1, err := NewReifiedConstraint(ineq1, b1)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: reify ineq1: %w", err)
			}
			model.AddConstraint(r1)
			bools = append(bools, b1)
			reifList = append(reifList, r1)

			// 2) X[j] + W[j] ≤ X[i]
			xjPlus, _, err := makeOffset(x[j], w[j])
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: offset for x[%d]: %w", j, err)
			}
			ineq2, err := NewInequality(xjPlus, x[i], LessEqual)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: ineq2: %w", err)
			}
			b2 := model.NewVariable(boolDom)
			r2, err := NewReifiedConstraint(ineq2, b2)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: reify ineq2: %w", err)
			}
			model.AddConstraint(r2)
			bools = append(bools, b2)
			reifList = append(reifList, r2)

			// 3) Y[i] + H[i] ≤ Y[j]
			yiPlus, _, err := makeOffset(y[i], h[i])
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: offset for y[%d]: %w", i, err)
			}
			ineq3, err := NewInequality(yiPlus, y[j], LessEqual)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: ineq3: %w", err)
			}
			b3 := model.NewVariable(boolDom)
			r3, err := NewReifiedConstraint(ineq3, b3)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: reify ineq3: %w", err)
			}
			model.AddConstraint(r3)
			bools = append(bools, b3)
			reifList = append(reifList, r3)

			// 4) Y[j] + H[j] ≤ Y[i]
			yjPlus, _, err := makeOffset(y[j], h[j])
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: offset for y[%d]: %w", j, err)
			}
			ineq4, err := NewInequality(yjPlus, y[i], LessEqual)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: ineq4: %w", err)
			}
			b4 := model.NewVariable(boolDom)
			r4, err := NewReifiedConstraint(ineq4, b4)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: reify ineq4: %w", err)
			}
			model.AddConstraint(r4)
			bools = append(bools, b4)
			reifList = append(reifList, r4)

			// At least one true among the four disjuncts
			total := model.NewVariable(atLeastOneTrueDom)
			sum, err := NewBoolSum(bools, total)
			if err != nil {
				return nil, fmt.Errorf("NewDiffn: BoolSum per pair (%d,%d): %w", i, j, err)
			}
			model.AddConstraint(sum)

			reifs[i] = append(reifs[i], r1, r2, r3, r4)
		}
	}

	// Keep a defensive copy
	xx := make([]*FDVariable, n)
	copy(xx, x)
	yy := make([]*FDVariable, n)
	copy(yy, y)
	ww := make([]int, n)
	copy(ww, w)
	hh := make([]int, n)
	copy(hh, h)

	d := &Diffn{x: xx, y: yy, w: ww, h: hh, reifs: reifs}
	model.AddConstraint(d)
	return d, nil
}

func (d *Diffn) Variables() []*FDVariable {
	out := make([]*FDVariable, 0, len(d.x)+len(d.y))
	out = append(out, d.x...)
	out = append(out, d.y...)
	return out
}

func (d *Diffn) Type() string   { return "Diffn" }
func (d *Diffn) String() string { return fmt.Sprintf("Diffn(n=%d)", len(d.x)) }

// Propagate is a no-op: pruning is performed by the internal reified inequalities
// and their BoolSum disjunctions.
func (d *Diffn) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	return state, nil
}
