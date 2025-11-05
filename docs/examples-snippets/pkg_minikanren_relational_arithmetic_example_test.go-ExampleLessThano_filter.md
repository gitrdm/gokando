```go
func ExampleLessThano_filter() {
	result := Run(10, func(q *Var) Goal {
		return Conj(
			LessThano(q, NewAtom(5)),
			Membero(q, List(NewAtom(1), NewAtom(3), NewAtom(7), NewAtom(2))),
		)
	})
	fmt.Printf("Found %d values\n", len(result))
	// Output: Found 3 values
}

```


