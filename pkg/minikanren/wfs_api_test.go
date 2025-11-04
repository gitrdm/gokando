package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestNegationTruth_Basic verifies True/False outcomes for complete inner goals.
func TestNegationTruth_Basic(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{"unreachable": 1, "path": 0})

	// Inner complete with no answers => True
	pathNone := func(ctx context.Context, answers chan<- map[int64]Term) error { return nil }
	tv, err := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("x"), NewAtom("y")}), pathNone)
	if err != nil {
		t.Fatalf("NegationTruth error: %v", err)
	}
	if tv != TruthTrue {
		t.Fatalf("expected TruthTrue, got %v", tv)
	}

	// Inner produces an answer => False
	pathSome := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{}
		return nil
	}
	tv2, err := engine.NegationTruth(context.Background(), "unreachable", NewCallPattern("path", []Term{NewAtom("p"), NewAtom("q")}), pathSome)
	if err != nil {
		t.Fatalf("NegationTruth error: %v", err)
	}
	if tv2 != TruthFalse {
		t.Fatalf("expected TruthFalse, got %v", tv2)
	}
}

// TestNegationTruth_Undefined verifies Undefined when inner is incomplete (conditional).
func TestNegationTruth_Undefined(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{"unreachable": 1, "path": 0})

	// Slow evaluator to keep inner active briefly
	pathSlow := func(ctx context.Context, answers chan<- map[int64]Term) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	tv, err := engine.NegationTruth(ctx, "unreachable", NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")}), pathSlow)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
	// Either we returned Undefined due to conditional, or context deadline interrupted before decision.
	if tv != TruthUndefined {
		t.Fatalf("expected TruthUndefined, got %v", tv)
	}
}
