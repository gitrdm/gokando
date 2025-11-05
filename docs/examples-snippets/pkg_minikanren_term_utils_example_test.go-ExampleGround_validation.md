```go
func ExampleGround_validation() {
	// Validate that all arguments are provided before processing
	process := func(arg1, arg2, result Term) Goal {
		return Conj(
			Ground(arg1),
			Ground(arg2),
			Eq(result, NewAtom("processed")),
		)
	}

	// Valid case: both arguments bound
	result1 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), NewAtom("b"), q)
	})

	// Invalid case: argument contains unbound variable
	x := Fresh("x")
	result2 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), x, q)
	})

	fmt.Printf("Both arguments ground: %d results\n", len(result1))
	fmt.Printf("Unbound argument: %d results\n", len(result2))
	// Output:
	// Both arguments ground: 1 results
	// Unbound argument: 0 results
}

```


