```go
func ExampleArityo_typeChecking() {
	// Ensure a term is a binary operation (arity 2)
	validateBinary := func(term Term) Goal {
		return Arityo(term, NewAtom(2))
	}

	binaryOp := List(NewAtom("left"), NewAtom("right"))
	unaryOp := List(NewAtom("single"))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(binaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(unaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	fmt.Printf("Binary operation: %s\n", result1[0])
	fmt.Printf("Unary operation fails: %d results\n", len(result2))
	// Output:
	// Binary operation: valid-binary
	// Unary operation fails: 0 results
}

```


