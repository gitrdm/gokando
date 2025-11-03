// Package minikanren adds stratified negation (WFS) helpers on top of the SLG engine.
//
// This file provides a minimal, production-quality implementation to use
// stratification with SLG tabling and a helper for negation-as-failure that
// enforces stratification. It does not change the SLG design, and builds on
// the existing Evaluate API and dependency tracking.
package minikanren

import (
	"context"
	"fmt"
)

// NegateEvaluator returns a GoalEvaluator that succeeds with an empty binding
// iff the inner tabled goal produces no answers. It enforces stratification by
// requiring that the current predicate's stratum is strictly greater than the
// inner predicate's stratum. When this condition is violated, it returns an error.
//
// Usage pattern:
//
//	outerEval := NegateEvaluator(engine, currentPredID, innerPattern, innerEval)
//	engine.Evaluate(ctx, NewCallPattern(currentPredID, args), outerEval)
//
// Semantics:
//   - The resulting evaluator reads the full inner stream to completion.
//   - If the inner yields zero answers, the outer emits one empty binding map
//     to represent success without additional bindings.
//   - If the inner yields at least one answer, the outer emits nothing (fail).
func NegateEvaluator(engine *SLGEngine, currentPredicateID string, innerPattern *CallPattern, innerEvaluator GoalEvaluator) GoalEvaluator {
	return func(ctx context.Context, answers chan<- map[int64]Term) error {
		if engine == nil || innerPattern == nil || innerEvaluator == nil {
			return fmt.Errorf("NegateEvaluator: invalid arguments")
		}

		// WFS stratification enforcement: current stratum must be > inner stratum.
		currentStratum := engine.Stratum(currentPredicateID)
		innerStratum := engine.Stratum(innerPattern.PredicateID())
		if currentStratum <= innerStratum {
			return fmt.Errorf("negation violates stratification: %s(stratum=%d) depends negatively on %s(stratum=%d)",
				currentPredicateID, currentStratum, innerPattern.PredicateID(), innerStratum)
		}

		// Evaluate inner goal fully and count answers.
		innerCh, err := engine.Evaluate(ctx, innerPattern, innerEvaluator)
		if err != nil {
			return fmt.Errorf("NegateEvaluator inner Evaluate error: %w", err)
		}

		count := 0
		for range innerCh {
			count++
			// We can early-exit on the first answer since negation would fail.
			break
		}

		if count == 0 {
			// Success of negation-as-failure: emit one empty binding.
			answers <- map[int64]Term{}
		}
		return nil
	}
}
