```go
func Example_dcgTerminal() {
	engine := NewSLGEngine(nil)
	DefineRule("digit1", Terminal(NewAtom("1")))

	solutions := Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "digit1", input, rest),
			Eq(q, rest),
		)
	})

	// Prints whether we matched and consumed the only token (leaving empty list)
	fmt.Println(len(solutions) == 1)
	// Output:
	// true
}

```


