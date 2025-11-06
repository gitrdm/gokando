package minikanren

import (
	"context"
	"testing"
	"time"
)

// makeList creates a list from atoms (helper for tests).
func makeList(atoms ...*Atom) Term {
	if len(atoms) == 0 {
		return Nil
	}
	var result Term = Nil
	for i := len(atoms) - 1; i >= 0; i-- {
		result = NewPair(atoms[i], result)
	}
	return result
}

// Helper to extract list of atoms from a term
func extractAtomList(t Term) []string {
	var result []string
	for {
		if pair, ok := t.(*Pair); ok {
			if atom, ok := pair.car.(*Atom); ok {
				result = append(result, atom.Value().(string))
			}
			t = pair.cdr
		} else {
			// End of list (should be nil atom or variable)
			break
		}
	}
	return result
}

// TestTerminal_SingleToken tests that Terminal matches a single token.
func TestTerminal_SingleToken(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("digit", Terminal(NewAtom("1")))

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "digit", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	// rest should be empty list (nil atom)
	if atom, ok := solutions[0].(*Atom); !ok || atom.Value() != nil {
		t.Errorf("Expected nil atom (empty list), got %v", solutions[0])
	}
}

// TestTerminal_NoMatch tests that Terminal fails on mismatch.
func TestTerminal_NoMatch(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("digit1", Terminal(NewAtom("1")))

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("2")) // doesn't match "1"
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "digit1", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) != 0 {
		t.Errorf("Expected 0 solutions (mismatch), got %d", len(solutions))
	}
}

// TestSeq_TwoTokens tests sequencing of two terminals.
func TestSeq_TwoTokens(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("oneTwo", Seq(Terminal(NewAtom("1")), Terminal(NewAtom("2"))))

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("1"), NewAtom("2"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "oneTwo", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	// rest should be empty list
	if atom, ok := solutions[0].(*Atom); !ok || atom.Value() != nil {
		t.Errorf("Expected empty list, got %v", solutions[0])
	}
}

// TestSeq_PartialMatch tests that Seq fails if only first token matches.
func TestSeq_PartialMatch(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("oneTwo", Seq(Terminal(NewAtom("1")), Terminal(NewAtom("2"))))

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("1"), NewAtom("3")) // second token wrong
		rest := Fresh("rest")
		return ParseWithSLG(engine, "oneTwo", input, rest)
	})

	if len(solutions) != 0 {
		t.Errorf("Expected 0 solutions (partial match), got %d", len(solutions))
	}
}

// TestAlternation_Choice tests that Alternation explores all branches.
func TestAlternation_Choice(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("digit", Alternation(
		Terminal(NewAtom("0")),
		Terminal(NewAtom("1")),
		Terminal(NewAtom("2")),
	))

	// Try each digit
	for _, digit := range []string{"0", "1", "2"} {
		solutions := Run(1, func(q *Var) Goal {
			input := makeList(NewAtom(digit))
			rest := Fresh("rest")
			return Conj(
				ParseWithSLG(engine, "digit", input, rest),
				Eq(q, rest),
			)
		})

		if len(solutions) != 1 {
			t.Errorf("Digit %s: expected 1 solution, got %d", digit, len(solutions))
		}
	}
}

// TestNonTerminal_SimpleReference tests referencing another rule.
func TestNonTerminal_SimpleReference(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("digit", Terminal(NewAtom("1")))
	DefineRule("number", NonTerminal(engine, "digit"))

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "number", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}
}

// TestLeftRecursion_BaseFirst tests left recursion with base case first.
// This is the classic clause ordering that works without SLG.
func TestLeftRecursion_BaseFirst(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= term | expr "+" term  (base case first)
	DefineRule("expr", Alternation(
		NonTerminal(engine, "term"),
		Seq(NonTerminal(engine, "expr"), Terminal(NewAtom("+")), NonTerminal(engine, "term")),
	))
	DefineRule("term", Terminal(NewAtom("1")))

	// First test: Parse just "1" (base case only)
	t.Run("parse_single_term", func(t *testing.T) {
		solutions := Run(1, func(q *Var) Goal {
			input := makeList(NewAtom("1"))
			rest := Fresh("rest")
			return Conj(
				ParseWithSLG(engine, "expr", input, rest),
				Eq(q, rest),
			)
		})

		if len(solutions) != 1 {
			t.Fatalf("Expected 1 solution for '1', got %d", len(solutions))
		}
	})

	// Second test: Parse "1+1" (requires recursion)
	t.Run("parse_addition", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		solutions := RunWithContext(ctx, 3, func(q *Var) Goal {
			input := makeList(NewAtom("1"), NewAtom("+"), NewAtom("1"))
			rest := Fresh("rest")
			return Conj(
				ParseWithSLG(engine, "expr", input, rest),
				Eq(q, rest),
			)
		})

		if len(solutions) < 1 {
			t.Fatalf("Expected at least 1 solution for '1+1', got %d", len(solutions))
		}

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				t.Fatal("Parsing '1+1' timed out")
			}
		default:
		}
	})
}

// TestLeftRecursion_RecursiveFirst tests left recursion with recursive case first.
// This is the KEY test for clause-order independence - would timeout without SLG.
//
// TODO: Complex left-recursive parsing not yet fully supported.
func TestLeftRecursion_RecursiveFirst(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= expr "+" term | term  (recursive case first - harder!)
	DefineRule("exprRecFirst", Alternation(
		Seq(NonTerminal(engine, "exprRecFirst"), Terminal(NewAtom("+")), NonTerminal(engine, "termRecFirst")),
		NonTerminal(engine, "termRecFirst"),
	))
	DefineRule("termRecFirst", Terminal(NewAtom("1")))

	// Parse "1+1" with timeout to catch infinite loops
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions := RunWithContext(ctx, 3, func(q *Var) Goal {
		input := makeList(NewAtom("1"), NewAtom("+"), NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "exprRecFirst", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) < 1 {
		t.Fatalf("Expected at least 1 solution, got %d", len(solutions))
	}

	// Should complete without timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Test timed out - left recursion not handled properly")
		}
	default:
		// Success
	}
}

// TestLeftRecursion_Mixed tests mixed clause ordering.
//
// TODO: Complex left-recursive parsing not yet fully supported.
func TestLeftRecursion_Mixed(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= expr "+" expr | expr "*" term | term
	DefineRule("exprMixed", Alternation(
		Seq(NonTerminal(engine, "exprMixed"), Terminal(NewAtom("+")), NonTerminal(engine, "exprMixed")),
		Seq(NonTerminal(engine, "exprMixed"), Terminal(NewAtom("*")), NonTerminal(engine, "termMixed")),
		NonTerminal(engine, "termMixed"),
	))
	DefineRule("termMixed", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions := RunWithContext(ctx, 3, func(q *Var) Goal {
		input := makeList(NewAtom("1"), NewAtom("*"), NewAtom("1"))
		rest := Fresh("rest")
		return ParseWithSLG(engine, "exprMixed", input, rest)
	})

	if len(solutions) < 1 {
		t.Fatalf("Expected at least 1 solution, got %d", len(solutions))
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Mixed ordering timed out")
		}
	default:
		// Success
	}
}

// TestRightRecursion tests right-recursive grammars (should work easily).
//
// TODO: Complex recursive parsing not yet fully supported.
func TestRightRecursion(t *testing.T) {
	engine := NewSLGEngine(nil)

	// list ::= item list | empty
	DefineRule("list", Alternation(
		Seq(NonTerminal(engine, "item"), NonTerminal(engine, "list")),
		Terminal(NewAtom("end")),
	))
	DefineRule("item", Terminal(NewAtom("x")))

	solutions := Run(3, func(q *Var) Goal {
		input := makeList(NewAtom("x"), NewAtom("x"), NewAtom("end"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "list", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) < 1 {
		t.Errorf("Expected at least 1 solution, got %d", len(solutions))
	}
}

// TestContextCancellation tests that parsing respects context cancellation.
func TestContextCancellation(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("infinite", Alternation(
		Seq(NonTerminal(engine, "infinite"), Terminal(NewAtom("x"))),
		Terminal(NewAtom("base")),
	))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	solutions := RunWithContext(ctx, 10, func(q *Var) Goal {
		input := makeList(NewAtom("base"))
		return ParseWithSLG(engine, "infinite", input, q)
	})

	// Should return empty or partial results, not hang
	if len(solutions) > 10 {
		t.Errorf("Expected ≤10 solutions with cancelled context, got %d", len(solutions))
	}
}

// TestVariableScoping tests that variables created inside Run closure work correctly.
func TestVariableScoping(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("token", Terminal(NewAtom("a")))

	// Correct: variables inside closure
	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("a"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "token", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) != 1 {
		t.Errorf("Variable scoping (correct): expected 1 solution, got %d", len(solutions))
	}
}

// TestEmptyInput tests parsing empty input.
func TestEmptyInput(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("empty", Terminal(NewAtom("x")))

	solutions := Run(1, func(q *Var) Goal {
		input := NewAtom(nil) // empty list
		rest := Fresh("rest")
		return ParseWithSLG(engine, "empty", input, rest)
	})

	// Should fail (can't match token from empty list)
	if len(solutions) != 0 {
		t.Errorf("Empty input: expected 0 solutions, got %d", len(solutions))
	}
}

// TestComplexGrammar tests a more realistic grammar.
func TestComplexGrammar(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Simple expression grammar
	// expr ::= term | expr "+" term
	// term ::= factor | term "*" factor
	// factor ::= "x" | "(" expr ")"

	DefineRule("cexpr", Alternation(
		NonTerminal(engine, "cterm"),
		Seq(NonTerminal(engine, "cexpr"), Terminal(NewAtom("+")), NonTerminal(engine, "cterm")),
	))

	DefineRule("cterm", Alternation(
		NonTerminal(engine, "cfactor"),
		Seq(NonTerminal(engine, "cterm"), Terminal(NewAtom("*")), NonTerminal(engine, "cfactor")),
	))

	DefineRule("cfactor", Alternation(
		Terminal(NewAtom("x")),
		Seq(Terminal(NewAtom("(")), NonTerminal(engine, "cexpr"), Terminal(NewAtom(")"))),
	))

	// Parse "x+x"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	solutions := RunWithContext(ctx, 5, func(q *Var) Goal {
		input := makeList(NewAtom("x"), NewAtom("+"), NewAtom("x"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "cexpr", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) < 1 {
		t.Fatalf("Complex grammar: expected at least 1 solution, got %d", len(solutions))
	}
}

// TestParsingModes tests using DCG in different modes (parsing vs generation).
func TestParsingModes(t *testing.T) {
	engine := NewSLGEngine(nil)
	DefineRule("ab", Seq(Terminal(NewAtom("a")), Terminal(NewAtom("b"))))

	// Mode 1: Full parsing (ground input, variable output)
	solutions1 := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("a"), NewAtom("b"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "ab", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions1) != 1 {
		t.Errorf("Parsing mode: expected 1 solution, got %d", len(solutions1))
	}

	// Mode 2: Recognition (ground input, ground output)
	solutions2 := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("a"), NewAtom("b"))
		rest := NewAtom(nil) // expect empty
		return Conj(
			ParseWithSLG(engine, "ab", input, rest),
			Eq(q, NewAtom("success")),
		)
	})

	if len(solutions2) != 1 {
		t.Errorf("Recognition mode: expected 1 solution, got %d", len(solutions2))
	}
}

// TestMultipleSolutions tests grammar with multiple parse trees.
//
// Note: SLG tabling deduplicates answers, so even though the grammar
// has two branches that parse the same input, SLG returns only one answer
// since both branches produce identical bindings. This is correct behavior
// for tabled evaluation.
func TestMultipleSolutions(t *testing.T) {
	engine := NewSLGEngine(nil)

	// Ambiguous: "a" ::= "a" | "a"
	DefineRule("ambiguous", Alternation(
		Terminal(NewAtom("a")),
		Terminal(NewAtom("a")),
	))

	solutions := Run(5, func(q *Var) Goal {
		input := makeList(NewAtom("a"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "ambiguous", input, rest),
			Eq(q, rest),
		)
	})

	// SLG deduplicates identical answers, so we get 1 solution
	if len(solutions) != 1 {
		t.Errorf("Ambiguous grammar: expected 1 solution (SLG deduplicates), got %d", len(solutions))
	}
	// Verify the solution is correct
	if len(solutions) > 0 && solutions[0].String() != "<nil>" {
		t.Errorf("Expected rest=nil, got %v", solutions[0])
	}
}

// TestUndefinedRule tests that undefined rules fail gracefully.
func TestUndefinedRule(t *testing.T) {
	engine := NewSLGEngine(nil)
	// Don't define "missing" rule

	solutions := Run(1, func(q *Var) Goal {
		input := makeList(NewAtom("x"))
		rest := Fresh("rest")
		return ParseWithSLG(engine, "missing", input, rest)
	})

	// Should fail (rule not found)
	if len(solutions) != 0 {
		t.Errorf("Undefined rule: expected 0 solutions, got %d", len(solutions))
	}
}
