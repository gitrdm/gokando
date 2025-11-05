package minikanren

import (
	"context"
	"fmt"
)

// ExampleFresho_basic demonstrates a simple freshness constraint where the
// nominal name does not occur free inside the term due to binding by Tie.
func ExampleFresho_basic() {
	a := NewAtom("a")

	// term: (tie a . a) — 'a' is bound, not free
	term := Tie(a, a)

	solutions := Run(1, func(q *Var) Goal {
		return Conj(
			Fresho(a, term),
			Eq(q, NewAtom("ok")),
		)
	})

	fmt.Println(solutions)
	// Output: [ok]
}

// ExampleFresho_violation shows that if the name occurs free, the goal fails.
func ExampleFresho_violation() {
	a := NewAtom("a")

	// term: (a . ()) — 'a' appears free in the list
	list := NewPair(a, Nil)

	ctx := context.Background()
	goal := Fresho(a, list)
	stream := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	results, _ := stream.Take(1)
	fmt.Println(len(results))
	// Output: 0
}

// ExampleAlphaEqo_basic shows alpha-equivalence for lambda-like Tie terms.
func ExampleAlphaEqo_basic() {
	a := NewAtom("a")
	b := NewAtom("b")

	t1 := Lambda(a, a) // λa.a
	t2 := Lambda(b, b) // λb.b

	results := Run(1, func(q *Var) Goal {
		return Conj(
			AlphaEqo(t1, t2),
			Eq(q, NewAtom(true)),
		)
	})
	fmt.Println(results)
	// Output: [true]
}

// ExampleAlphaEqo_nested distinguishes non-equivalent structures.
func ExampleAlphaEqo_nested() {
	a := NewAtom("a")
	b := NewAtom("b")

	// λa.λb.a  vs  λa.λb.b  (not alpha-equivalent)
	t1 := Lambda(a, Lambda(b, a))
	t2 := Lambda(a, Lambda(b, b))

	ctx := context.Background()
	goal := AlphaEqo(t1, t2)
	stream := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rs, _ := stream.Take(1)
	fmt.Println(len(rs))
	// Output: 0
}
