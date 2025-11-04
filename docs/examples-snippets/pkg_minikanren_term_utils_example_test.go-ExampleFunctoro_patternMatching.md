```go
func ExampleFunctoro_patternMatching() {
	// Dispatch based on functor
	dispatch := func(term, result Term) Goal {
		functor := Fresh("functor")
		return Conj(
			Functoro(term, functor),
			Conde(
				Conj(Eq(functor, NewAtom("add")), Eq(result, NewAtom("arithmetic"))),
				Conj(Eq(functor, NewAtom("cons")), Eq(result, NewAtom("list-operation"))),
				Conj(Eq(functor, NewAtom("eq")), Eq(result, NewAtom("comparison"))),
			),
		)
	}

	addTerm := NewPair(NewAtom("add"), List(NewAtom(1), NewAtom(2)))
	consTerm := NewPair(NewAtom("cons"), List(NewAtom("a"), Nil))

	result1 := Run(1, func(q *Var) Goal {
		return dispatch(addTerm, q)
	})

	result2 := Run(1, func(q *Var) Goal {
		return dispatch(consTerm, q)
	})

	fmt.Printf("add → %v\n", result1[0])
	fmt.Printf("cons → %v\n", result2[0])
	// Output:
	// add → arithmetic
	// cons → list-operation
}

```


