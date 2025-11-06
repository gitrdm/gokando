```go
func Example_dcgRecognition() {
	engine := NewSLGEngine(nil)

	// ab ::= "a" "b"
	DefineRule("ab", Seq(Terminal(NewAtom("a")), Terminal(NewAtom("b"))))

	// Ground input and ground output (empty list) means: recognize the whole string.
	ok := len(Run(1, func(q *Var) Goal {
		input := buildList(NewAtom("a"), NewAtom("b"))
		rest := NewAtom(nil) // expect fully consumed
		return Conj(
			ParseWithSLG(engine, "ab", input, rest),
			Eq(q, NewAtom("ok")),
		)
	})) == 1

	fmt.Println(ok)
	// Output:
	// true
}

```


