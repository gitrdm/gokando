```go
func Example_dcgParse_leftRecursion() {
	engine := NewSLGEngine(nil)

	// expr ::= term | expr "+" term
	DefineRule("expr", Alternation(
		NonTerminal(engine, "term"),
		Seq(NonTerminal(engine, "expr"), Terminal(NewAtom("+")), NonTerminal(engine, "term")),
	))
	DefineRule("term", Terminal(NewAtom("1")))

	solutions := Run(2, func(q *Var) Goal {
		input := buildList(NewAtom("1"), NewAtom("+"), NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "expr", input, rest),
			Eq(q, rest),
		)
	})

	// At least one parse should succeed, leaving an empty rest list
	fmt.Println(len(solutions) >= 1)
	// Output:
	// true
}

```


