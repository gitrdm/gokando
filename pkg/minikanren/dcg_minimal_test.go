package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestMinimalLeftRecursion - absolute minimal test
func TestMinimalLeftRecursion(t *testing.T) {
	engine := NewSLGEngine(nil)

	// expr ::= expr "+" term | term  (classic left recursion)
	// term ::= "1"
	DefineRule("exprMin", Alternation(
		Seq(NonTerminal(engine, "exprMin"), Terminal(NewAtom("+")), NonTerminal(engine, "termMin")),
		NonTerminal(engine, "termMin"),
	))
	DefineRule("termMin", Terminal(NewAtom("1")))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Just parse "1" - should match via base case (term branch)
	solutions := RunWithContext(ctx, 1, func(q *Var) Goal {
		input := makeList(NewAtom("1"))
		rest := Fresh("rest")
		return Conj(
			ParseWithSLG(engine, "exprMin", input, rest),
			Eq(q, rest),
		)
	})

	if len(solutions) == 0 {
		t.Fatal("Expected at least 1 solution (from base case), got 0")
	}
}
