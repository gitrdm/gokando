package minikanren

import (
	"fmt"
)

// ExampleNewRegular demonstrates the `Regular` global constraint by
// constructing a small DFA and constraining a short sequence of variables to
// be accepted by that automaton.
//
// The DFA recognizes exactly the strings whose last symbol is `1` over the
// alphabet {1,2}. We create three decision variables x1,x2,x3 each ranging
// over {1,2}, post `Regular([x1,x2,x3], dfa)`, and run propagation. The DFA
// semantics causes the last variable x3 to be pruned to the singleton {1},
// illustrating how `Regular` enforces sequence-level constraints via
// automata-based propagation.
func ExampleNewRegular() {
	// Build DFA: states 1=start, 2=last=1, 3=last=2; accept={2}
	numStates, start, accept, delta := buildEndsWith1DFA()

	model := NewModel()
	x1 := model.NewVariableWithName(NewBitSetDomain(2), "x1")
	x2 := model.NewVariableWithName(NewBitSetDomain(2), "x2")
	x3 := model.NewVariableWithName(NewBitSetDomain(2), "x3")

	c, _ := NewRegular([]*FDVariable{x1, x2, x3}, numStates, start, accept, delta)
	model.AddConstraint(c)
	solver := NewSolver(model)

	st, _ := solver.propagate(nil)
	fmt.Println("x1:", solver.GetDomain(st, x1.ID()))
	fmt.Println("x2:", solver.GetDomain(st, x2.ID()))
	fmt.Println("x3:", solver.GetDomain(st, x3.ID()))
	// Output:
	// x1: {1..2}
	// x2: {1..2}
	// x3: {1}
}
