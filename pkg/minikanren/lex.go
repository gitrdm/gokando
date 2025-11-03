// Package minikanren adds a lexicographic ordering global constraint.
//
// This file implements LexLess and LexLessEq over two equal-length vectors
// of FD variables. These constraints are commonly used for symmetry breaking
// and sequencing models.
//
// Contract:
//   - X = [x1..xn], Y = [y1..yn], n >= 1
//   - Domains are positive integers as usual (1..MaxValue)
//   - LexLess(X, Y)  enforces (x1, x2, ..., xn) <  (y1, y2, ..., yn)
//   - LexLessEq(X, Y) enforces (x1, x2, ..., xn) <= (y1, y2, ..., yn)
//
// Propagation (bounds-consistent, O(n)):
//   - Maintain whether the prefix can still be equal: eqPrefix = true initially.
//   - For i = 1..n while eqPrefix holds:
//   - Prune xi > max(yi): xi ∈ (-∞ .. maxYi]
//   - Prune yi < min(xi): yi ∈ [minXi .. +∞)
//   - If max(xi) < min(yi), the constraint is already satisfied at i and
//     later positions are unconstrained by Lex; we may stop.
//   - Update eqPrefix := eqPrefix && (xi and yi have a non-empty intersection)
//   - For strict LexLess, detect the all-equal tuple case early:
//   - If for all i, dom(xi) and dom(yi) are singletons with the same value,
//     the constraint is inconsistent.
//
// This filtering is sound and inexpensive. Stronger propagation can be achieved
// using reified decompositions, but this implementation integrates cleanly with
// the solver's fixed-point propagation loop and avoids adding internal goals.
package minikanren

import "fmt"

// lexKind distinguishes strict vs non-strict variants.
type lexKind int

const (
	lexLT lexKind = iota // strict <
	lexLE                // non-strict ≤
)

// Lexicographic orders two equal-length vectors of variables.
type Lexicographic struct {
	xs   []*FDVariable
	ys   []*FDVariable
	kind lexKind
}

// NewLexLess creates a strict lexicographic ordering constraint X < Y.
func NewLexLess(xs, ys []*FDVariable) (PropagationConstraint, error) {
	return newLex(xs, ys, lexLT)
}

// NewLexLessEq creates a non-strict lexicographic ordering constraint X ≤ Y.
func NewLexLessEq(xs, ys []*FDVariable) (PropagationConstraint, error) {
	return newLex(xs, ys, lexLE)
}

func newLex(xs, ys []*FDVariable, k lexKind) (PropagationConstraint, error) {
	if len(xs) == 0 || len(ys) == 0 {
		return nil, fmt.Errorf("Lex: vectors must be non-empty")
	}
	if len(xs) != len(ys) {
		return nil, fmt.Errorf("Lex: length mismatch xs=%d ys=%d", len(xs), len(ys))
	}
	for i := range xs {
		if xs[i] == nil || ys[i] == nil {
			return nil, fmt.Errorf("Lex: nil variable at index %d", i)
		}
	}
	vx := make([]*FDVariable, len(xs))
	vy := make([]*FDVariable, len(ys))
	copy(vx, xs)
	copy(vy, ys)
	return &Lexicographic{xs: vx, ys: vy, kind: k}, nil
}

// Variables returns all variables in X followed by Y.
func (l *Lexicographic) Variables() []*FDVariable {
	return append(append([]*FDVariable{}, l.xs...), l.ys...)
}

// Type names the constraint.
func (l *Lexicographic) Type() string {
	if l.kind == lexLT {
		return "LexLess"
	}
	return "LexLessEq"
}

// String returns a readable description.
func (l *Lexicographic) String() string {
	k := "<"
	if l.kind == lexLE {
		k = "≤"
	}
	return fmt.Sprintf("Lex(%s, n=%d)", k, len(l.xs))
}

// Propagate enforces bounds-consistent pruning for lexicographic ordering.
func (l *Lexicographic) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Lex.Propagate: nil solver")
	}
	n := len(l.xs)
	if n == 0 {
		return state, nil
	}

	// Read domains once.
	dx := make([]Domain, n)
	dy := make([]Domain, n)
	for i := 0; i < n; i++ {
		dx[i] = solver.GetDomain(state, l.xs[i].ID())
		dy[i] = solver.GetDomain(state, l.ys[i].ID())
		if dx[i] == nil || dy[i] == nil {
			return nil, fmt.Errorf("Lex: nil domain at index %d", i)
		}
		if dx[i].Count() == 0 || dy[i].Count() == 0 {
			return nil, fmt.Errorf("Lex: empty domain at index %d", i)
		}
	}

	// Strict check: all-equal tuple is forbidden for lexLT.
	if l.kind == lexLT {
		allEqualSingleton := true
		for i := 0; i < n; i++ {
			if !(dx[i].IsSingleton() && dy[i].IsSingleton() && dx[i].SingletonValue() == dy[i].SingletonValue()) {
				allEqualSingleton = false
				break
			}
		}
		if allEqualSingleton {
			return nil, fmt.Errorf("LexLess: all positions equal; strict ordering impossible")
		}
	}

	newState := state
	eqPrefix := true
	for i := 0; i < n && eqPrefix; i++ {
		xi, yi := dx[i], dy[i]
		minX, maxX := xi.Min(), xi.Max()
		minY, maxY := yi.Min(), yi.Max()

		// If intervals are disjoint with minX >= minY and maxX >= maxY and there is no xi<yi possible,
		// detect obvious infeasibility early.
		if minX > maxY { // xi cannot be <= yi for any values
			return nil, fmt.Errorf("Lex: xi.min(%d) > yi.max(%d) at index %d", minX, maxY, i)
		}

		// Bounds pruning under equal-prefix hypothesis.
		// Prune xi values strictly greater than maxY.
		if maxX > maxY {
			nd := xi.RemoveAbove(maxY)
			if nd.Count() == 0 {
				return nil, fmt.Errorf("Lex: pruning x[%d] above %d empties domain", i, maxY)
			}
			if !nd.Equal(xi) {
				var changed bool
				newState, changed = solver.SetDomain(newState, l.xs[i].ID(), nd)
				if changed {
					dx[i] = nd
				}
			}
		}
		// Prune yi values strictly less than minX.
		if minY < minX {
			nd := yi.RemoveBelow(minX)
			if nd.Count() == 0 {
				return nil, fmt.Errorf("Lex: pruning y[%d] below %d empties domain", i, minX)
			}
			if !nd.Equal(yi) {
				var changed bool
				newState, changed = solver.SetDomain(newState, l.ys[i].ID(), nd)
				if changed {
					dy[i] = nd
				}
			}
		}

		// If at this position all remaining values satisfy xi < yi by bounds,
		// the lex constraint is already satisfied regardless of later positions.
		if dx[i].Max() < dy[i].Min() {
			break
		}

		// Update eqPrefix: check if equality could still hold here.
		// We approximate by testing if there exists any common value.
		// Fast path using bounds and membership checks around the overlap range.
		if dx[i].Max() < dy[i].Min() || dy[i].Max() < dx[i].Min() {
			eqPrefix = false
			continue
		}
		// Try to find at least one candidate value in [max(minX,minY) .. min(maxX,maxY)]
		lo := minX
		if minY > lo {
			lo = minY
		}
		hi := maxX
		if maxY < hi {
			hi = maxY
		}
		foundEq := false
		for v := lo; v <= hi; v++ {
			if dx[i].Has(v) && dy[i].Has(v) {
				foundEq = true
				break
			}
		}
		eqPrefix = foundEq
	}

	return newState, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
