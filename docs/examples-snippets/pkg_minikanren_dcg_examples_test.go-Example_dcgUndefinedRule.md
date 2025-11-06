```go
func Example_dcgUndefinedRule() {
	engine := NewSLGEngine(nil)

	// Intentionally do not DefineRule("missing", ...)
	solutions := Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("x"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "missing", input, rest),
			Eq(q, rest),
		)
	})

	fmt.Println(len(solutions))
	// Output:
	// 0
}

```


