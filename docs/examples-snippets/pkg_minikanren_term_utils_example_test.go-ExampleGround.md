```go
func ExampleGround() {
	x := Fresh("x")

	// Check if a bound variable is ground
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Eq(x, NewAtom("hello")),
			Ground(x),
			Eq(q, NewAtom("bound-is-ground")),
		)
	})

	// Check if an unbound variable is ground (should fail)
	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(Fresh("unbound")),
			Eq(q, NewAtom("should-not-appear")),
		)
	})

	fmt.Printf("Bound variable is ground: %d results\n", len(result1))
	fmt.Printf("Unbound variable is not ground: %d results\n", len(result2))
	// Output:
	// Bound variable is ground: 1 results
	// Unbound variable is not ground: 0 results
}

```


