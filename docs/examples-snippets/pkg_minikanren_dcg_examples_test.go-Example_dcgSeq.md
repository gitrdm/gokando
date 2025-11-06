```go
func Example_dcgSeq() {
	engine := NewSLGEngine(nil)
	DefineRule("oneTwo", Seq(Terminal(NewAtom("1")), Terminal(NewAtom("2"))))

	solutions := Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("1"), NewAtom("2"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "oneTwo", input, rest),
			Eq(q, rest),
		)
	})

	fmt.Println(len(solutions) == 1)
	// Output:
	// true
}

```


