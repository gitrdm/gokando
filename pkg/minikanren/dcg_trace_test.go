package minikanren

import (
	"context"
	"log"
	"testing"
	"time"
)

// TestLeftRecursionWithTracing adds detailed tracing to understand the flow.
func TestLeftRecursionWithTracing(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	engine := NewSLGEngine(nil)

	// expr ::= term  (base case only for now)
	// term ::= "1"
	log.Println("[TEST] Defining base-case-only grammar")
	DefineRule("exprTrace", NonTerminal(engine, "termTrace"))
	DefineRule("termTrace", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	log.Println("[TEST] Starting Run")
	solutions := RunWithContext(ctx, 1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		log.Printf("[TEST] Created input=%v, rest=%v", input, rest)
		return Conj(
			ParseWithSLG(engine, "exprTrace", input, rest),
			Eq(q, rest),
		)
	})

	log.Printf("[TEST] Got %d solutions", len(solutions))
	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	log.Println("[TEST] Test completed successfully")
}

// TestActualLeftRecursionWithTracing tests real left recursion.
func TestActualLeftRecursionWithTracing(t *testing.T) {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	engine := NewSLGEngine(nil)

	// expr ::= expr "+" term | term
	// term ::= "1"
	log.Println("[TEST] Defining left-recursive grammar")
	DefineRule("exprLRTrace", Alternation(
		Seq(NonTerminal(engine, "exprLRTrace"), Terminal(NewAtom("+")), NonTerminal(engine, "termLRTrace")),
		NonTerminal(engine, "termLRTrace"),
	))
	DefineRule("termLRTrace", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("[TEST] Starting Run for left-recursive grammar")
	solutions := RunWithContext(ctx, 1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		log.Printf("[TEST] Created input=%v, rest=%v", input, rest)
		return Conj(
			ParseWithSLG(engine, "exprLRTrace", input, rest),
			Eq(q, rest),
		)
	})

	log.Printf("[TEST] Got %d solutions", len(solutions))

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Test timed out - left recursion not handled")
		}
	default:
		if len(solutions) < 1 {
			t.Fatalf("Expected at least 1 solution, got %d", len(solutions))
		}
		log.Println("[TEST] Test completed successfully")
	}
}
