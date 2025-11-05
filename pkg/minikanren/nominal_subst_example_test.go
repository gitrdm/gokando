package minikanren

import (
	"fmt"
)

// ExampleSubsto_basic demonstrates straightforward substitution inside structures.
func ExampleSubsto_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := NewPair(a, NewPair(NewAtom(1), Nil)) // (a 1)

	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, b, q),
		)
	})

	// Expect (b . (1 . <nil>)) in Pair notation
	fmt.Println(results[0])
	// Output: (b . (1 . <nil>))
}

// ExampleSubsto_avoidCapture shows alpha-renaming of binders to avoid capture.
func ExampleSubsto_avoidCapture() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := Lambda(b, a) // λb.a

	results := Run(1, func(q *Var) Goal {
		return Substo(term, a, b, q)
	})

	// Result is λb'. b where b' is fresh (not b)
	tie := results[0].(*TieTerm)
	// Print whether binder==b and whether body==b for a stable assertion
	fmt.Printf("binderIsB:%v bodyIsB:%v\n", tie.name.Equal(b), tie.body.Equal(b))
	// Output: binderIsB:false bodyIsB:true
}
