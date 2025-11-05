```go
func ExampleGround_list() {
	// Fully ground list
	groundList := List(NewAtom(1), NewAtom(2), NewAtom(3))

	// Partially ground list
	x := Fresh("x")
	partialList := List(NewAtom(1), x, NewAtom(3))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(groundList),
			Eq(q, NewAtom("fully-ground")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(partialList),
			Eq(q, NewAtom("partially-ground")),
		)
	})

	fmt.Printf("Fully ground list: %s\n", result1[0])
	fmt.Printf("Partially ground list fails: %d results\n", len(result2))
	// Output:
	// Fully ground list: fully-ground
	// Partially ground list fails: 0 results
}

```


