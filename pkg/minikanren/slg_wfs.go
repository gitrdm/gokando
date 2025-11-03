// Package minikanren adds stratified negation (WFS) helpers on top of the SLG engine.
//
// This file provides production-quality Well-Founded Semantics (WFS) implementation
// for stratified and general negation with conditional answers, delay sets, and
// completion. It builds on the existing SLG Evaluate API and dependency tracking.
//
// Synchronization approach (no sleeps/timers):
//   - Non-blocking fast path: we first drain innerCh if it's already closed or has a
//     buffered answer to catch immediate outcomes with zero wait.
//   - Race-free subscription: we use a versioned event sequence (EventSeq/WaitChangeSince)
//     to avoid missing just-fired events.
//   - Engine handshake: we obtain a pre-start sequence and a Started() signal from the
//     engine to deterministically handle the "inner completes immediately with no answers"
//     case without any timers. We also prioritize real change events over the started signal.
//   - Reverse-dependency propagation ensures conditional answers are simplified or
//     retracted as soon as child outcomes are known.
package minikanren

import (
	"context"
	"fmt"
	"time"
)

// NegateEvaluator returns a GoalEvaluator that succeeds with an empty binding
// iff the inner tabled goal produces no answers. It enforces stratification by
// requiring that the current predicate's stratum is strictly greater than the
// inner predicate's stratum. When this condition is violated, it returns an error.
//
// WFS Semantics:
//   - If the inner subgoal is complete and has no answers: emit unconditional success (empty binding).
//   - If the inner subgoal is complete and has answers: fail (emit nothing).
//   - If the inner subgoal is incomplete (still evaluating): emit a conditional answer
//     delayed on the completion of the inner subgoal.
//
// Usage pattern:
//
//	outerEval := NegateEvaluator(engine, currentPredID, innerPattern, innerEval)
//	engine.Evaluate(ctx, NewCallPattern(currentPredID, args), outerEval)
//
// Note: For conditional answers to work correctly, this evaluator must be called
// within an SLG evaluation context where the parent subgoal entry is accessible.
// The engine automatically provides this context via produceAndConsume.
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

		// Evaluate the inner subgoal with handshake information to enable
		// deterministic initial shape without timers.
		innerCh, innerEntry, preSeq, startedCh, err := engine.evaluateWithHandshake(ctx, innerPattern, innerEvaluator)
		if err != nil {
			return fmt.Errorf("negation inner evaluation failed: %w", err)
		}

		// Fast path: non-blocking attempt to observe immediate inner outcome.
		select {
		case _, ok := <-innerCh:
			if ok {
				// Inner produced at least one answer: negation fails
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Channel closed with no answers: unconditional success
			answers <- map[int64]Term{}
			go func() {
				for range innerCh {
				}
			}()
			return nil
		default:
			// No immediate result; proceed to state-based logic
		}

		// The key question: Is the inner goal already complete, or is it still evaluating?
		//
		// If complete: we can safely consume from channel (won't block indefinitely)
		// If active: consuming might block, so we should emit conditional answer
		//
		// Strategy: Check initial status, then consume accordingly.

		initialStatus := innerEntry.Status()
		initialCount := innerEntry.Answers().Count()

		if initialStatus == StatusComplete || initialStatus == StatusFailed {
			// Inner is already complete - safe to drain channel
			if initialCount > 0 {
				// Has answers: negation fails
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// No answers: unconditional success
			answers <- map[int64]Term{}
			go func() {
				for range innerCh {
				}
			}()
			return nil
		}

		// Inner is still active. Check if it has produced answers yet.
		if initialCount > 0 {
			// Already has answers: negation fails
			go func() {
				for range innerCh {
				}
			}()
			return nil
		}

		// Race-free, zero-wait event check using pre-start sequence captured
		// before the producer was started (for new subgoals). This ensures we
		// observe immediate completion or first answer deterministically.
		waitCh := innerEntry.WaitChangeSince(preSeq)
		// Handshake: prefer immediate change if available. First do a non-blocking
		// check on waitCh to give priority to actual changes over the started signal.
		select {
		case <-waitCh:
			// Some change occurred; re-evaluate status and count
			st := innerEntry.Status()
			cnt := innerEntry.Answers().Count()
			if cnt > 0 {
				// Now has answers: negation fails
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			if st == StatusComplete || st == StatusFailed {
				// Completed with no answers: unconditional success
				answers <- map[int64]Term{}
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Still active: fall through to emit conditional
		default:
			// If no immediate change, wait for either a change or producer start
			select {
			case <-waitCh:
				st := innerEntry.Status()
				cnt := innerEntry.Answers().Count()
				if cnt > 0 {
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				if st == StatusComplete || st == StatusFailed {
					answers <- map[int64]Term{}
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				// Still active: fall through to emit conditional
			case <-startedCh:
				// Producer is running but no immediate change; optionally wait a tiny
				// event-driven window to catch an immediate completion/answer.
				if engine.config != nil && engine.config.NegationPeekTimeout > 0 {
					timer := time.NewTimer(engine.config.NegationPeekTimeout)
					select {
					case <-waitCh:
						st := innerEntry.Status()
						cnt := innerEntry.Answers().Count()
						if cnt > 0 {
							go func() {
								for range innerCh {
								}
							}()
							if !timer.Stop() {
								<-timer.C
							}
							return nil
						}
						if st == StatusComplete || st == StatusFailed {
							answers <- map[int64]Term{}
							go func() {
								for range innerCh {
								}
							}()
							if !timer.Stop() {
								<-timer.C
							}
							return nil
						}
						// Still active; fall through to emit conditional
					case <-timer.C:
						// timeout: proceed with conditional
					case <-ctx.Done():
						if !timer.Stop() {
							<-timer.C
						}
						return ctx.Err()
					}
				}
				// Emit conditional
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Inner is active with no answers yet: emit conditional answer
		// Queue delay set and register reverse dependency so engine can
		// simplify or retract this answer later based on child outcome.
		if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
			if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
				ds := NewDelaySet()
				ds.Add(innerPattern.Hash())
				parentEntry.QueueDelaySetForNextAnswer(ds)
				engine.addReverseDependency(innerPattern.Hash(), parentEntry)
			}
		}
		answers <- map[int64]Term{}

		// Monitor channel in background to drain it
		go func() {
			for range innerCh {
			}
		}()

		return nil
	}
}

// slgProducerEntryKey is used to pass the current SubgoalEntry in producer context
// so evaluators can attach metadata (delay sets) to answers they produce.
type slgProducerEntryKey struct{}
