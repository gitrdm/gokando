```go
func ExampleAlphaEqo_basic() {
	a := NewAtom("a")
	b := NewAtom("b")

	t1 := Lambda(a, a) // λa.a
	t2 := Lambda(b, b) // λb.b

	results := Run(1, func(q *Var) Goal {
		return Conj(
			AlphaEqo(t1, t2),
			Eq(q, NewAtom(true)),
		)
	})
	fmt.Println(results)
	// Output: [true]
}

```


