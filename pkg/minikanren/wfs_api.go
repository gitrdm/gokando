package minikanren

// Public-facing helpers for Well-Founded Semantics (WFS).
//
// These APIs provide a minimal, composable surface for end users to observe
// the truth status induced by negation without depending on internal engine
// details. They build on the existing SLG engine and NegateEvaluator.

import (
	"context"
	"fmt"
)

// TruthValue represents the three-valued logic outcomes under WFS.
// For negation-as-failure over a subgoal G, the truth of not(G) is:
//   - True:     G completes with no answers
//   - False:    G produces at least one answer
//   - Undefined: G is incomplete (conditional)
type TruthValue int

const (
	TruthUndefined TruthValue = iota // computation is incomplete or conditionally delayed
	TruthFalse                       // negation fails (inner has answers)
	TruthTrue                        // negation succeeds unconditionally (inner complete, no answers)
)

func (t TruthValue) String() string {
	switch t {
	case TruthTrue:
		return "true"
	case TruthFalse:
		return "false"
	case TruthUndefined:
		return "undefined"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// NegationTruth evaluates not(innerPattern) using the provided inner evaluator
// and reports the WFS truth value. It does not enumerate all answers; it only
// determines whether the negation holds (true), fails (false), or is currently
// undefined (conditional due to active dependencies).
//
// Contract:
// - Returns (TruthTrue, nil) if an unconditional empty binding is produced.
// - Returns (TruthFalse, nil) if no binding is produced because the inner has answers.
// - Returns (TruthUndefined, nil) if a conditional binding is produced (delayed).
// - Returns (TruthUndefined, ctx.Err()) if the context is canceled before a decision.
func (e *SLGEngine) NegationTruth(ctx context.Context, currentPredicateID string, innerPattern *CallPattern, innerEvaluator GoalEvaluator) (TruthValue, error) {
	if e == nil || innerPattern == nil || innerEvaluator == nil {
		return TruthUndefined, fmt.Errorf("NegationTruth: invalid arguments")
	}

	negEval := NegateEvaluator(e, currentPredicateID, innerPattern, innerEvaluator)

	// Build a unique negation call pattern keyed by the inner call pattern to avoid
	// cache collisions across different inner subgoals.
	// Use the inner pattern hash to create a unique negation call key.
	negPat := NewCallPattern(fmt.Sprintf("not#%d", innerPattern.Hash()), nil)

	// Evaluate the negation goal and observe the first (and only) binding if any.
	ch, err := e.Evaluate(ctx, negPat, negEval)
	if err != nil {
		return TruthUndefined, err
	}

	select {
	case ans, ok := <-ch:
		if !ok {
			// No answer emitted => negation failed (inner has answers)
			return TruthFalse, nil
		}
		// Answer emitted: check if conditional (has delay set) or unconditional
		entry, _ := e.subgoals.GetOrCreate(negPat)
		// First answer index is 0
		if ds := entry.DelaySetFor(0); ds != nil && !ds.Empty() {
			_ = ans // binding is always empty for negation; ignore content
			return TruthUndefined, nil
		}
		return TruthTrue, nil
	case <-ctx.Done():
		return TruthUndefined, ctx.Err()
	}
}
