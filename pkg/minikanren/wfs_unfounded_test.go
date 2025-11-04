package minikanren

import (
	"context"
	"testing"
)

// TestUnfoundedSet_MutualNegationUndefined constructs p :- not q, q :- not p
// and checks that not(p) and not(q) are both undefined under WFS.
func TestUnfoundedSet_MutualNegationUndefined(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{"p": 1, "q": 1}) // same stratum allowed for mutual negation test

	// We'll define p and q where p :- not q and q :- not p using proper sub-calls.
	var pEval, qEval GoalEvaluator
	pPat := NewCallPattern("p", nil)
	qPat := NewCallPattern("q", nil)

	// q :- not p
	qEval = func(ctx context.Context, answers chan<- map[int64]Term) error {
		inner := NegateEvaluator(engine, "q", pPat, pEval)
		ch, err := engine.Evaluate(ctx, NewCallPattern("q:neg", nil), inner)
		if err != nil {
			return err
		}
		for range ch {
			answers <- map[int64]Term{}
		}
		return nil
	}

	// p :- not q
	pEval = func(ctx context.Context, answers chan<- map[int64]Term) error {
		inner := NegateEvaluator(engine, "p", qPat, qEval)
		ch, err := engine.Evaluate(ctx, NewCallPattern("p:neg", nil), inner)
		if err != nil {
			return err
		}
		for range ch {
			answers <- map[int64]Term{}
		}
		return nil
	}

	// Truth of not p should be undefined
	tvP, err := engine.NegationTruth(context.Background(), "top", pPat, pEval)
	if err != nil {
		t.Fatalf("NegationTruth error: %v", err)
	}
	if tvP != TruthUndefined {
		t.Fatalf("expected not p to be Undefined, got %v", tvP)
	}

	// Truth of not q should be undefined
	tvQ, err := engine.NegationTruth(context.Background(), "top", qPat, qEval)
	if err != nil {
		t.Fatalf("NegationTruth error: %v", err)
	}
	if tvQ != TruthUndefined {
		t.Fatalf("expected not q to be Undefined, got %v", tvQ)
	}
}
