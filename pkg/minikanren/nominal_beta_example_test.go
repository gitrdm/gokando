package minikanren

import (
	"fmt"
)

// ExampleBetaReduceo_basic shows a single beta-reduction step.
func ExampleBetaReduceo_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := App(Lambda(a, a), b)

	results := Run(1, func(q *Var) Goal { return BetaReduceo(term, q) })
	fmt.Println(results[0])
	// Output: b
}

// ExampleBetaNormalizeo_basic shows normalization to normal form.
func ExampleBetaNormalizeo_basic() {
	a := NewAtom("a")
	x := NewAtom("x")
	y := NewAtom("y")
	term := App(Lambda(a, Lambda(x, a)), y)

	results := Run(1, func(q *Var) Goal { return BetaNormalizeo(term, q) })
	fmt.Println(results[0])
	// Output: (tie x . y)
}

// ExampleFreeNameso_basic lists free nominal names in a term.
func ExampleFreeNameso_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := Lambda(a, App(a, b))

	results := Run(1, func(q *Var) Goal { return FreeNameso(term, q) })
	fmt.Println(results[0])
	// Output: (b . <nil>)
}
