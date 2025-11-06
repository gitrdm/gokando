```go
func Example_dcgAmbiguousDedup() {
	engine := NewSLGEngine(nil)

	// ambiguous ::= "a" | "a" (two identical branches)
	DefineRule("ambiguous", Alternation(
		Terminal(NewAtom("a")),
		Terminal(NewAtom("a")),
	))

	solutions := Run(5, func(q *Var) Goal {
		input := buildList(NewAtom("a"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "ambiguous", input, rest),
			Eq(q, rest),
		)
	})

	// Even though there are two derivations, identical bindings are deduped to one answer.
	fmt.Println(len(solutions))
	// Output:
	// 1
}

```


