```go
func ExampleCopyTerm_metaProgramming() {
	// Define a template with variables
	x := Fresh("x")
	y := Fresh("y")
	template := NewPair(NewAtom("add"), List(x, y))

	// Create multiple instances of the template
	result := Run(2, func(q *Var) Goal {
		instance := Fresh("instance")
		return Conj(
			CopyTerm(template, instance),
			// Each instance can be instantiated differently
			Membero(q, List(
				NewPair(NewAtom("instance"), instance),
			)),
		)
	})

	fmt.Printf("Template instances: %d\n", len(result))
	// Output:
	// Template instances: 1
}

```


