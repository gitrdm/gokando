// Package minikanren adds stratified negation (WFS) helpers on top of the SLG engine.
//
// This file provides production-quality Well-Founded Semantics (WFS) implementation
// for stratified and general negation with conditional answers, delay sets, and
// completion. It builds on the existing SLG Evaluate API and dependency tracking.
//
// About the small event-based peek in negation:
//   - NegateEvaluator first uses a non-blocking fast path (draining innerCh if already
//     closed or if an answer is already buffered). This handles the common immediate
//     outcomes without waiting.
//   - To make the “inner completes immediately with no answers” case deterministic,
//     NegateEvaluator may optionally wait for a tiny, event-driven window
//     (SLGConfig.NegationPeekTimeout, default 1ms) for the inner subgoal’s first
//     state change (answer or completion). This is an event wait, not a sleep.
//   - Correctness does not depend on this window. If it elapses with no event, we
//     emit a conditional answer that will be simplified/retracted by the engine’s
//     reverse-dependency mechanism as soon as the child’s outcome is known. The
//     peek only influences the initial “shape” (conditional vs. unconditional), not
//     the final semantics.
//   - You can set NegationPeekTimeout to 0 for purely non-blocking behavior, or raise
//     it slightly (e.g., a few milliseconds) on heavily loaded systems if you prefer
//     more unconditional answers when inner goals finish immediately.
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

		// Evaluate the inner subgoal and consume its answers.
		// This handles both new and existing subgoals correctly via the engine's
		// standard producer/consumer mechanism.
		innerCh, err := engine.Evaluate(ctx, innerPattern, innerEvaluator)
		if err != nil {
			return fmt.Errorf("negation inner evaluation failed: %w", err)
		}

		// Get the inner entry
		innerEntry, _ := engine.subgoals.GetOrCreate(innerPattern)

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

		// Try to detect immediate completion or new answers via event channel.
		// Optionally wait for a short, configurable grace period to catch
		// immediate completion/answers without racing on scheduling.
		if engine.config.NegationPeekTimeout > 0 {
			waitCh := innerEntry.Event()
			// Try an immediate, zero-allocation check first; only allocate a timer if needed
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
				// Still active: fall through to conditional
			default:
				timer := time.NewTimer(engine.config.NegationPeekTimeout)
				defer timer.Stop()
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
					// Still active: fall through to conditional
				case <-timer.C:
					// No immediate event; treat as still active
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			// Non-blocking peek only
			select {
			case <-innerEntry.Event():
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
			default:
				// No event; proceed to emit conditional
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
