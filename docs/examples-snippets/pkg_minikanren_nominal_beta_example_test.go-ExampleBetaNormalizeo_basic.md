```go
func ExampleBetaNormalizeo_basic() {
	a := NewAtom("a")
	x := NewAtom("x")
	y := NewAtom("y")
	term := App(Lambda(a, Lambda(x, a)), y)

	results := Run(1, func(q *Var) Goal { return BetaNormalizeo(term, q) })
	fmt.Println(results[0])
	// Output: (tie x . y)
}

```


