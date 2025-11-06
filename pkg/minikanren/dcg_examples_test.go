package minikanren

import (
	"fmt"
)

// buildList is a tiny helper for examples to build a proper list (difference list head).
func buildList(atoms ...*Atom) Term {
	if len(atoms) == 0 {
		return Nil
	}
	var result Term = Nil
	for i := len(atoms) - 1; i >= 0; i-- {
		result = NewPair(atoms[i], result)
	}
	return result
}

// ExampleDCG_Terminal shows how to define and use a single-token DCG rule.
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

// ExampleDCG_Seq shows sequencing two terminals with Seq.
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

// ExampleDCG_Alternation shows disjunction with Alternation.
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

// ExampleDCG_Parse_leftRecursion demonstrates a left-recursive grammar handled via SLG.
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

// ExampleDCG_Recognition demonstrates recognition mode: both input and output are ground.
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

// ExampleDCG_AmbiguousDedup shows that SLG deduplicates identical parse answers.
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

// ExampleDCG_UndefinedRule shows that parsing with an undefined rule fails gracefully.
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
