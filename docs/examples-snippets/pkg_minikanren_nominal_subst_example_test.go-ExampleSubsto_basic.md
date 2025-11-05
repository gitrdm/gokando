```go
func ExampleSubsto_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := NewPair(a, NewPair(NewAtom(1), Nil)) // (a 1)

	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, b, q),
		)
	})

	// Expect (b . (1 . <nil>)) in Pair notation
	fmt.Println(results[0])
	// Output: (b . (1 . <nil>))
}

```


