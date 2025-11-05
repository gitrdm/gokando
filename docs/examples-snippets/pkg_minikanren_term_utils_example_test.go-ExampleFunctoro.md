```go
func ExampleFunctoro() {
	// Create a compound term like foo(1, 2)
	term := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2)))

	result := Run(1, func(functor *Var) Goal {
		return Functoro(term, functor)
	})

	fmt.Printf("Functor: %v\n", result[0])
	// Output:
	// Functor: foo
}

```


