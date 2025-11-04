```go
func ExampleArityo() {
	// Arity of an atom is 0
	result1 := Run(1, func(arity *Var) Goal {
		return Arityo(NewAtom("hello"), arity)
	})

	// Arity of a list is its length
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	result2 := Run(1, func(arity *Var) Goal {
		return Arityo(list, arity)
	})

	fmt.Printf("Atom arity: %v\n", result1[0])
	fmt.Printf("List arity: %v\n", result2[0])
	// Output:
	// Atom arity: 0
	// List arity: 3
}

```


