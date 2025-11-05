```go
func ExampleFresho_basic() {
	a := NewAtom("a")

	// term: (tie a . a) — 'a' is bound, not free
	term := Tie(a, a)

	solutions := Run(1, func(q *Var) Goal {
		return Conj(
			Fresho(a, term),
			Eq(q, NewAtom("ok")),
		)
	})

	fmt.Println(solutions)
	// Output: [ok]
}

```


