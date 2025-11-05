package minikanren_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Example_hlapi_noOverlap demonstrates how to use the HLAPI helper
// to post a NoOverlap constraint and inspect the resulting solutions.
//
// In this small model we:
//   - create two integer start variables `s1` and `s2` with domain [1..3];
//   - post `NoOverlap` with fixed durations [2,2], which requires that the
//     intervals [s1, s1+2) and [s2, s2+2) do not overlap;
//   - enumerate all solutions and print how many distinct start pairs satisfy
//     the constraint.
//
// Intuitively only the assignments (s1=1,s2=3) and (s1=3,s2=1) are valid
// starts for two tasks of duration 2 placed in the domain [1..4), so the
// example prints "2".
func Example_hlapi_noOverlap() {
	m := NewModel()
	s := m.IntVarsWithNames([]string{"s1", "s2"}, 1, 3)
	_ = m.NoOverlap(s, []int{2, 2})

	// Enumerate solutions; only (1,3) and (3,1) are valid starts
	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 2
}

// Example_hlapi_cumulative mirrors the NewCumulative example via HLAPI.
func Example_hlapi_cumulative() {
	m := NewModel()
	A := m.IntVar(2, 2, "A") // fixed start=2
	B := m.IntVar(1, 4, "B") // start in [1..4]
	_ = m.Cumulative([]*FDVariable{A, B}, []int{2, 2}, []int{2, 1}, 2)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 1) // trigger propagation

	fmt.Println("A:", s.GetDomain(nil, A.ID()))
	fmt.Println("B:", s.GetDomain(nil, B.ID()))
	// Output:
	// A: {2}
	// B: {4}
}

// Example_hlapi_gcc mirrors the NewGlobalCardinality example via HLAPI.
func Example_hlapi_gcc() {
	m := NewModel()
	a := m.IntVar(1, 1, "a") // fixed to 1
	b := m.IntVar(1, 2, "b")
	c := m.IntVar(1, 2, "c")

	min := make([]int, 3)
	max := make([]int, 3)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 3
	_ = m.GlobalCardinality([]*FDVariable{a, b, c}, min, max)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	fmt.Println("a:", s.GetDomain(nil, a.ID()))
	fmt.Println("b:", s.GetDomain(nil, b.ID()))
	fmt.Println("c:", s.GetDomain(nil, c.ID()))
	// Output:
	// a: {1}
	// b: {2}
	// c: {2}
}

// Example_hlapi_lexLessEq mirrors the NewLexLessEq example via HLAPI.
func Example_hlapi_lexLessEq() {
	m := NewModel()
	// Use compact range helpers instead of explicit value sets
	x1 := m.IntVar(2, 4, "x1")
	x2 := m.IntVar(1, 3, "x2")
	y1 := m.IntVar(3, 5, "y1")
	y2 := m.IntVar(2, 4, "y2")
	_ = m.LexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})

	// Minimal propagation: background context and solution cap 0
	s := NewSolver(m)
	_, _ = s.Solve(context.Background(), 0)

	fmt.Printf("y1: %s\n", s.GetDomain(nil, y1.ID()))
	// Output:
	// y1: {3..5}
}

// Example_hlapi_regular mirrors the NewRegular example via HLAPI.
func Example_hlapi_regular() {
	// Build DFA: accepts sequences ending with symbol 1 over alphabet {1,2}
	numStates, start, accept, delta := endsWith1DFA()

	m := NewModel()
	// x1 := m.NewVariableWithName(NewBitSetDomain(2), "x1")
	x1 := m.IntVar(1, 2, "x1")
	// x2 := m.NewVariableWithName(NewBitSetDomain(2), "x2")
	x2 := m.IntVar(1, 2, "x2")
	// x3 := m.NewVariableWithName(NewBitSetDomain(2), "x3")
	x3 := m.IntVar(1, 2, "x3")
	_ = m.Regular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)
	fmt.Println("x1:", s.GetDomain(nil, x1.ID()))
	fmt.Println("x2:", s.GetDomain(nil, x2.ID()))
	fmt.Println("x3:", s.GetDomain(nil, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}

// Example_hlapi_table mirrors the NewTable example via HLAPI.
func Example_hlapi_table() {
	m := NewModel()
	// x := m.NewVariableWithName(NewBitSetDomain(5), "x")
	x := m.IntVar(1, 5, "x")
	// y ∈ {1,2} upfront so we can avoid internal propagation calls
	// y := m.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "y")
	y := m.IntVarValues([]int{1, 2}, "y")

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	_ = m.Table([]*FDVariable{x, y}, rows)

	s := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = s.Solve(ctx, 0)

	xd := s.GetDomain(nil, x.ID())
	yd := s.GetDomain(nil, y.ID())

	fmt.Printf("x: %v\n", xd)
	fmt.Printf("y: %v\n", yd)
	// Output:
	// x: {1,3}
	// y: {1..2}
}

// endsWith1DFA returns a simple DFA over symbols {1,2} that accepts
// exactly those strings whose last symbol is 1.
func endsWith1DFA() (numStates, start int, accept []int, delta [][]int) {
	// States: 1=start, 2=last=1, 3=last=2; accept={2}
	numStates, start = 3, 1
	accept = []int{2}
	// delta rows sized to alphabetMax+1=3 (index 0 unused)
	delta = [][]int{
		// s=1
		{0, 2, 3},
		// s=2
		{0, 2, 3},
		// s=3
		{0, 2, 3},
	}
	return
}
