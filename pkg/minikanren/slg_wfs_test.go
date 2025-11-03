package minikanren

import (
	"context"
	"testing"
)

// TestNegation_Stratified ensures NegateEvaluator enforces stratification and
// implements negation-as-failure correctly for a simple predicate.
func TestNegation_Stratified(t *testing.T) {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{
		"p": 1, // current
		"q": 0, // inner (lower)
	})

	// Inner predicate q/1: q(a) is true, others false.
	qPattern := func(arg Term) *CallPattern {
		return NewCallPattern("q", []Term{arg})
	}
	qEval := func(wanted Term) GoalEvaluator {
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			// If wanted equals atom("a"), succeed once.
			if atom, ok := wanted.(*Atom); ok && atom.Value() == "a" {
				answers <- map[int64]Term{}
			}
			return nil
		}
	}

	// p/1(X) :- not(q(X)). We'll test for X=a (should fail) and X=b (should succeed).
	// For X=b, negation succeeds and we expect one empty binding.
	negEvalA := NegateEvaluator(engine, "p", qPattern(NewAtom("a")), qEval(NewAtom("a")))
	negEvalB := NegateEvaluator(engine, "p", qPattern(NewAtom("b")), qEval(NewAtom("b")))

	ctx := context.Background()

	// p(a) should fail: no answers produced.
	chA, err := engine.Evaluate(ctx, NewCallPattern("p", []Term{NewAtom("a")}), negEvalA)
	if err != nil {
		t.Fatalf("Evaluate p(a): %v", err)
	}
	for ans := range chA {
		_ = ans
		t.Fatalf("expected no answers for p(a), got one")
	}

	// p(b) should succeed once: one empty binding.
	chB, err := engine.Evaluate(ctx, NewCallPattern("p", []Term{NewAtom("b")}), negEvalB)
	if err != nil {
		t.Fatalf("Evaluate p(b): %v", err)
	}
	count := 0
	for range chB {
		count++
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 answer for p(b), got %d", count)
	}
}

// TestNegation_ViolatesStratification verifies we error when negating a same-or-higher
// stratum predicate.
func TestNegation_ViolatesStratification(t *testing.T) {
	engine := NewSLGEngine(nil)
	engine.SetStrata(map[string]int{
		"p": 0,
		"q": 0, // same stratum -> violation for negation
	})

	qPat := NewCallPattern("q", []Term{NewAtom("x")})
	qEval := GoalEvaluator(func(ctx context.Context, answers chan<- map[int64]Term) error { return nil })

	negEval := NegateEvaluator(engine, "p", qPat, qEval)
	pPat := NewCallPattern("p", []Term{NewAtom("x")})
	ch, _ := engine.Evaluate(context.Background(), pPat, negEval)
	// Channel should be closed with no answers due to evaluator error.
	for ans := range ch {
		_ = ans
		t.Fatalf("expected no answers for stratification violation")
	}
	// Subgoal status should be Failed
	entry := engine.subgoals.Get(pPat)
	if entry == nil || entry.Status() != StatusFailed {
		t.Fatalf("expected subgoal status Failed, got %v", func() SubgoalStatus {
			if entry == nil {
				return StatusInvalidated
			} else {
				return entry.Status()
			}
		}())
	}
}
