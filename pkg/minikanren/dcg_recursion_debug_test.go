package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestSimpleLeftRecursion is a minimal test for left recursion.
func TestSimpleLeftRecursion(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= term
	// term ::= "1"
	DefineRule("exprSimple", NonTerminal(engine, "termSimple"))
	DefineRule("termSimple", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	t.Log("Starting simple non-recursive test...")
	solutions := RunWithContext(ctx, 1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "exprSimple", input, rest),
			Eq(q, rest),
		)
	})

	t.Logf("Got %d solutions", len(solutions))
	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Test timed out")
		}
	default:
		t.Log("Test completed successfully")
	}
}

// TestActualLeftRecursion tests actual left recursion.
func TestActualLeftRecursion(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= expr "+" term | term
	// term ::= "1"
	DefineRule("exprLR", Alternation(
		Seq(NonTerminal(engine, "exprLR"), Terminal(NewAtom("+")), NonTerminal(engine, "termLR")),
		NonTerminal(engine, "termLR"),
	))
	DefineRule("termLR", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	t.Log("Starting left-recursive test...")
	solutions := RunWithContext(ctx, 1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "exprLR", input, rest),
			Eq(q, rest),
		)
	})

	t.Logf("Got %d solutions", len(solutions))
	if len(solutions) < 1 {
		t.Fatalf("Expected at least 1 solution, got %d", len(solutions))
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Test timed out - left recursion not handled")
		}
	default:
		t.Log("Test completed successfully")
	}
}
