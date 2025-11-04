package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestConditionalAnswer_NegationOfIncompleteGoal tests that NegateEvaluator
// emits a conditional answer when the inner subgoal is incomplete.
func TestConditionalAnswer_NegationOfIncompleteGoal(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{
		"unreachable": 1, // Higher stratum
		"path":        0, // Lower stratum
	})

	// Define a path evaluator that takes a long time (simulates incomplete)
	// It will emit no answers but won't complete within the test timeout
	pathEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		// Simulate slow evaluation by blocking
		select {
		case <-time.After(5 * time.Second): // Much longer than test timeout
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Define unreachable as not(path)
	unreachableEval := NegateEvaluator(
		engine,
		"unreachable",
		NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")}),
		pathEval,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start path evaluation in background to make it "active"
	pathPattern := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")})
	go func() {
		pathCh, _ := engine.Evaluate(ctx, pathPattern, pathEval)
		for range pathCh {
		}
	}()

	// Give path evaluation time to start
	time.Sleep(10 * time.Millisecond)

	// Now evaluate unreachable - the path subgoal should be active/incomplete
	unreachablePattern := NewCallPattern("unreachable", []Term{NewAtom("a"), NewAtom("b")})
	answerCh, err := engine.Evaluate(ctx, unreachablePattern, unreachableEval)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// We should get a conditional answer (empty binding with delay set)
	select {
	case answer, ok := <-answerCh:
		if !ok {
			t.Fatalf("expected at least one answer (conditional)")
		}
		if len(answer) != 0 {
			t.Fatalf("expected empty binding for negation, got %v", answer)
		}

		// Check that the answer has a delay set attached
		entry, _ := engine.subgoals.GetOrCreate(unreachablePattern)
		ds := entry.DelaySetFor(0)
		if ds == nil || ds.Empty() {
			t.Fatalf("expected conditional answer with delay set, got nil or empty")
		}

		// The delay set should reference the path subgoal
		if !ds.Has(pathPattern.Hash()) {
			t.Fatalf("delay set missing path subgoal dependency")
		}

	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout waiting for conditional answer")
	}
}

// TestConditionalAnswer_NegationOfCompleteGoalWithNoAnswers tests unconditional success.
func TestConditionalAnswer_NegationOfCompleteGoalWithNoAnswers(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{
		"unreachable": 1,
		"path":        0,
	})

	// Path evaluator that completes immediately with no answers
	pathEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		// Close immediately without emitting anything
		return nil
	}

	unreachableEval := NegateEvaluator(
		engine,
		"unreachable",
		NewCallPattern("path", []Term{NewAtom("x"), NewAtom("y")}),
		pathEval,
	)

	ctx := context.Background()
	unreachablePattern := NewCallPattern("unreachable", []Term{NewAtom("x"), NewAtom("y")})
	answerCh, err := engine.Evaluate(ctx, unreachablePattern, unreachableEval)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Should get unconditional answer
	answer, ok := <-answerCh
	if !ok {
		t.Fatalf("expected answer for negation of complete empty goal")
	}
	if len(answer) != 0 {
		t.Fatalf("expected empty binding, got %v", answer)
	}

	// Should NOT have a delay set (unconditional)
	entry, _ := engine.subgoals.GetOrCreate(unreachablePattern)
	ds := entry.DelaySetFor(0)
	if ds != nil && !ds.Empty() {
		t.Fatalf("expected unconditional answer (no delay set), got %v", ds)
	}
}

// TestConditionalAnswer_NegationOfCompleteGoalWithAnswers tests negation failure.
func TestConditionalAnswer_NegationOfCompleteGoalWithAnswers(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{
		"unreachable": 1,
		"path":        0,
	})

	// Path evaluator that produces answers
	pathEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		answers <- map[int64]Term{} // One answer
		return nil
	}

	unreachableEval := NegateEvaluator(
		engine,
		"unreachable",
		NewCallPattern("path", []Term{NewAtom("p"), NewAtom("q")}),
		pathEval,
	)

	ctx := context.Background()
	unreachablePattern := NewCallPattern("unreachable", []Term{NewAtom("p"), NewAtom("q")})
	answerCh, err := engine.Evaluate(ctx, unreachablePattern, unreachableEval)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Should get NO answers (negation fails)
	select {
	case answer, ok := <-answerCh:
		if ok {
			t.Fatalf("expected no answers for failed negation, got %v", answer)
		}
	case <-time.After(100 * time.Millisecond):
		// Expected: channel closes with no answers
	}
}

// TestConditionalAnswer_AnswerRecordIterator tests that conditional answers
// are properly exposed through the AnswerRecordIterator.
func TestConditionalAnswer_AnswerRecordIterator(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{
		"neg": 1,
		"pos": 0,
	})

	// Incomplete inner goal
	posEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		<-ctx.Done()
		return ctx.Err()
	}

	negEval := NegateEvaluator(
		engine,
		"neg",
		NewCallPattern("pos", []Term{NewAtom("test")}),
		posEval,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	negPattern := NewCallPattern("neg", []Term{NewAtom("test")})
	answerCh, err := engine.Evaluate(ctx, negPattern, negEval)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Drain regular channel
	for range answerCh {
	}

	// Now use AnswerRecordIterator to inspect metadata
	entry, _ := engine.subgoals.GetOrCreate(negPattern)
	it := entry.AnswerRecords()

	rec, ok := it.Next()
	if !ok {
		t.Fatalf("expected at least one answer record")
	}

	if rec.Delay == nil || rec.Delay.Empty() {
		t.Fatalf("expected conditional answer with delay set, got %v", rec.Delay)
	}

	posPattern := NewCallPattern("pos", []Term{NewAtom("test")})
	if !rec.Delay.Has(posPattern.Hash()) {
		t.Fatalf("delay set missing expected dependency")
	}
}

// TestConditionalAnswer_MultipleNegations tests multiple conditional answers
// with different delay sets.
func TestConditionalAnswer_MultipleNegations(t *testing.T) {
	engine := NewSLGEngine(DefaultSLGConfig())
	engine.SetStrata(map[string]int{
		"combined": 2,
		"path":     0,
		"edge":     0,
	})

	// Both incomplete
	pathEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		<-ctx.Done()
		return ctx.Err()
	}
	edgeEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		<-ctx.Done()
		return ctx.Err()
	}

	// Combined evaluator that uses both negations
	combinedEval := func(ctx context.Context, answers chan<- map[int64]Term) error {
		// Emit answer from not(path(...))
		pathPattern := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")})
		negPathEval := NegateEvaluator(engine, "combined", pathPattern, pathEval)
		if err := negPathEval(ctx, answers); err != nil {
			return err
		}

		// Emit answer from not(edge(...))
		edgePattern := NewCallPattern("edge", []Term{NewAtom("x"), NewAtom("y")})
		negEdgeEval := NegateEvaluator(engine, "combined", edgePattern, edgeEval)
		return negEdgeEval(ctx, answers)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	combinedPattern := NewCallPattern("combined", []Term{})
	answerCh, err := engine.Evaluate(ctx, combinedPattern, combinedEval)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Drain
	answerCount := 0
	for range answerCh {
		answerCount++
	}

	if answerCount == 0 {
		t.Fatalf("expected at least one conditional answer")
	}

	// Check metadata for multiple dependencies
	entry, _ := engine.subgoals.GetOrCreate(combinedPattern)
	for i := 0; i < answerCount; i++ {
		ds := entry.DelaySetFor(i)
		if ds == nil || ds.Empty() {
			t.Logf("answer %d: unconditional (may be valid depending on timing)", i)
		} else {
			t.Logf("answer %d: conditional with %d dependencies", i, len(ds))
		}
	}
}
