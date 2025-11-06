```go
func Example_dcgAlternation() {
	engine := NewSLGEngine(nil)
	DefineRule("digit", Alternation(
		Terminal(NewAtom("0")),
		Terminal(NewAtom("1")),
	))

	ok0 := len(Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("0"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "digit", input, rest),
			Eq(q, rest),
		)
	})) == 1

	ok1 := len(Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "digit", input, rest),
			Eq(q, rest),
		)
	})) == 1

	fmt.Println(ok0 && ok1)
	// Output:
	// true
}

```


