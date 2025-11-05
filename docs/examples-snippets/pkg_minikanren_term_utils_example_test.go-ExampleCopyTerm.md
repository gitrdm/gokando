```go
func ExampleCopyTerm() {
	x := Fresh("x")
	original := List(x, NewAtom("middle"), x)

	// Copy the term - x will be replaced with a fresh variable
	_ = Run(1, func(copy *Var) Goal {
		return Conj(
			CopyTerm(original, copy),
			Eq(x, NewAtom("original-binding")), // Bind original x
		)
	})

	// The copy has fresh variables, not bound to "original-binding"
	fmt.Printf("Copy preserves structure with fresh variables\n")
	// Output:
	// Copy preserves structure with fresh variables
}

```


