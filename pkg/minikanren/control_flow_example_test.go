package minikanren

import (
	"fmt"
	"sort"
)

// ExampleIfa demonstrates if-then-else with all condition solutions.
func ExampleIfa() {
	res := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		cond := Conde(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
		thenG := Project([]Term{x}, func(v []Term) Goal { return Eq(q, v[0]) })
		elseG := Eq(q, NewAtom("none"))
		return Ifa(cond, thenG, elseG)
	})

	// Extract and sort values for deterministic output
	var values []int
	for _, r := range res {
		if atom, ok := r.(*Atom); ok {
			if val, ok := atom.Value().(int); ok {
				values = append(values, val)
			}
		}
	}
	sort.Ints(values)
	fmt.Println(values)
	// Output: [1 2]
}

// ExampleIfte demonstrates early commitment.
func ExampleIfte() {
	res := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		cond := Conde(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
		thenG := Project([]Term{x}, func(v []Term) Goal { return Eq(q, v[0]) })
		elseG := Eq(q, NewAtom("none"))
		return Ifte(cond, thenG, elseG)
	})

	// Should get exactly one solution (commits to first)
	fmt.Println(len(res))
	// Output: 1
}

// ExampleCallGoal shows meta-calling a goal stored as a term.
func ExampleCallGoal() {
	res := Run(10, func(q *Var) Goal {
		// Create a goal and store it in an atom
		goalAtom := NewAtom(Eq(q, NewAtom("ok")))
		// Invoke the goal indirectly
		return CallGoal(goalAtom)
	})
	for _, r := range res {
		fmt.Println(r)
	}
	// Output: ok
}
