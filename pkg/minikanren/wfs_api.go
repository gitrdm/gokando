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

// Internal sentinel used to identify NegationTruth probes so the engine can
// avoid recording permanent negative edges and bypass stratification checks.
const negTruthProbeID = "__negation_truth_probe__"

// Context key to mark evaluations initiated by NegationTruth. When present,
// negation evaluators bypass stratification enforcement and avoid recording
// permanent negative edges in the dependency graph.
type negTruthProbeCtxKey struct{}

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

	// Mark this evaluation as a truth probe to avoid side effects like
	// stratification enforcement and permanent negative-edge recording.
	ctx = context.WithValue(ctx, negTruthProbeCtxKey{}, true)

	// Evaluate the inner subgoal directly and infer truth without emitting answers.
	innerCh, innerEntry, preSeq, startedCh, err := e.evaluateWithHandshake(ctx, innerPattern, innerEvaluator)
	if err != nil {
		return TruthUndefined, err
	}

	// Fast path: immediate observation
	select {
	case _, ok := <-innerCh:
		if ok {
			// Inner produced an answer: if it's conditional or involved in a negative-edge cycle, treat as Undefined
			entry, _ := e.subgoals.GetOrCreate(innerPattern)
			if ds := entry.DelaySetFor(0); ds != nil && !ds.Empty() {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			e.computeUndefinedSCCs()
			if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			go func() {
				for range innerCh {
				}
			}()
			return TruthFalse, nil
		}
		// Channel closed with no answers: check for unfounded-set undefined
		e.computeUndefinedSCCs()
		if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
			go func() {
				for range innerCh {
				}
			}()
			return TruthUndefined, nil
		}
		// No timer-based peeks; rely on event sequencing + synchronous SCC compute
		go func() {
			for range innerCh {
			}
		}()
		return TruthTrue, nil
	default:
	}

	// State-based logic
	st := innerEntry.Status()
	cnt := innerEntry.Answers().Count()
	if st == StatusComplete || st == StatusFailed {
		if cnt > 0 {
			// If first answer is conditional or involved in negative-edge cycles, truth is Undefined
			if ds := innerEntry.DelaySetFor(0); ds != nil && !ds.Empty() {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			e.computeUndefinedSCCs()
			if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			go func() {
				for range innerCh {
				}
			}()
			return TruthFalse, nil
		}
		e.computeUndefinedSCCs()
		if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
			go func() {
				for range innerCh {
				}
			}()
			return TruthUndefined, nil
		}
		// No timer-based peeks; rely on event sequencing + synchronous SCC compute
		go func() {
			for range innerCh {
			}
		}()
		return TruthTrue, nil
	}
	if cnt > 0 {
		if ds := innerEntry.DelaySetFor(0); ds != nil && !ds.Empty() {
			go func() {
				for range innerCh {
				}
			}()
			return TruthUndefined, nil
		}
		e.computeUndefinedSCCs()
		if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
			go func() {
				for range innerCh {
				}
			}()
			return TruthUndefined, nil
		}
		go func() {
			for range innerCh {
			}
		}()
		return TruthFalse, nil
	}

	// Wait deterministically for either a change or producer start; prefer real changes over start signal.
	waitCh := innerEntry.WaitChangeSince(preSeq)
	// Prefer actual change if both are ready
	select {
	case <-waitCh:
		st2 := innerEntry.Status()
		cnt2 := innerEntry.Answers().Count()
		if cnt2 > 0 {
			if ds := innerEntry.DelaySetFor(0); ds != nil && !ds.Empty() {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			e.computeUndefinedSCCs()
			if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			go func() {
				for range innerCh {
				}
			}()
			return TruthFalse, nil
		}
		if st2 == StatusComplete || st2 == StatusFailed {
			e.computeUndefinedSCCs()
			if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
				go func() {
					for range innerCh {
					}
				}()
				return TruthUndefined, nil
			}
			// No timer-based peeks; rely on event sequencing + synchronous SCC compute
			go func() {
				for range innerCh {
				}
			}()
			return TruthTrue, nil
		}
		// Still active: undefined
		return TruthUndefined, nil
	default:
		// If no immediate change, wait for either change or producer start
		select {
		case <-waitCh:
			st2 := innerEntry.Status()
			cnt2 := innerEntry.Answers().Count()
			if cnt2 > 0 {
				if ds := innerEntry.DelaySetFor(0); ds != nil && !ds.Empty() {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				e.computeUndefinedSCCs()
				if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				go func() {
					for range innerCh {
					}
				}()
				return TruthFalse, nil
			}
			if st2 == StatusComplete || st2 == StatusFailed {
				e.computeUndefinedSCCs()
				if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				go func() {
					for range innerCh {
					}
				}()
				return TruthTrue, nil
			}
			return TruthUndefined, nil
		case <-startedCh:
			// Producer started without immediate change: re-check state to catch immediate completion
			st2 := innerEntry.Status()
			cnt2 := innerEntry.Answers().Count()
			if cnt2 > 0 {
				go func() {
					for range innerCh {
					}
				}()
				return TruthFalse, nil
			}
			if st2 == StatusComplete || st2 == StatusFailed {
				e.computeUndefinedSCCs()
				if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				go func() {
					for range innerCh {
					}
				}()
				return TruthTrue, nil
			}
			// Still active: wait for the next change deterministically or context cancellation
			select {
			case <-waitCh:
			case <-ctx.Done():
				return TruthUndefined, ctx.Err()
			}
			st3 := innerEntry.Status()
			cnt3 := innerEntry.Answers().Count()
			if cnt3 > 0 {
				if ds := innerEntry.DelaySetFor(0); ds != nil && !ds.Empty() {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				e.computeUndefinedSCCs()
				if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				go func() {
					for range innerCh {
					}
				}()
				return TruthFalse, nil
			}
			if st3 == StatusComplete || st3 == StatusFailed {
				e.computeUndefinedSCCs()
				if e.isInNegativeSCC(innerPattern.Hash()) || e.hasNegativeIncoming(innerPattern.Hash()) || e.hasNegEdgeReachableFrom(innerPattern.Hash()) {
					go func() {
						for range innerCh {
						}
					}()
					return TruthUndefined, nil
				}
				go func() {
					for range innerCh {
					}
				}()
				return TruthTrue, nil
			}
			return TruthUndefined, nil
		case <-ctx.Done():
			return TruthUndefined, ctx.Err()
		}
	}
}
