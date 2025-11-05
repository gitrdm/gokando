```go
func ExampleBetaReduceo_basic() {
	a := NewAtom("a")
	b := NewAtom("b")
	term := App(Lambda(a, a), b)

	results := Run(1, func(q *Var) Goal { return BetaReduceo(term, q) })
	fmt.Println(results[0])
	// Output: b
}

```


