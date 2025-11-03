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

		// Enforce stratification: disallow negation to a same-or-higher stratum unless this is a truth probe.
		if currentPredicateID != negTruthProbeID && ctx.Value(negTruthProbeCtxKey{}) == nil && engine != nil && engine.config != nil && engine.config.EnforceStratification {
			curr := engine.Stratum(currentPredicateID)
			innerStr := engine.Stratum(innerPattern.PredicateID())
			if curr <= innerStr {
				// Violation: cause the subgoal to fail
				return fmt.Errorf("stratification violation: %s negates %s", currentPredicateID, innerPattern.PredicateID())
			}
		}

		// Evaluate the inner subgoal with handshake information to enable
		// deterministic initial shape without timers.
		innerCh, innerEntry, preSeq, startedCh, err := engine.evaluateWithHandshake(ctx, innerPattern, innerEvaluator)
		if err != nil {
			return fmt.Errorf("negation inner evaluation failed: %w", err)
		}

		// Record the negative dependency edge early for unfounded set analysis,
		// except for internal NegationTruth probes where we avoid graph pollution.
		if currentPredicateID != negTruthProbeID && ctx.Value(negTruthProbeCtxKey{}) == nil {
			if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
				if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
					engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerPattern.Hash())
				}
			}
		}

		// Fast path: non-blocking attempt to observe immediate inner outcome.
		select {
		case _, ok := <-innerCh:
			if ok {
				// Inner produced at least one answer: negation fails
				wfsTracef("NegEval(%s <- %s): fast-path inner produced answer", currentPredicateID, innerPattern.PredicateID())
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Channel closed with no answers: may be Undefined if inner is in a negative-edge SCC
			wfsTracef("NegEval(%s <- %s): fast-path inner closed no answers", currentPredicateID, innerPattern.PredicateID())
			if engine.isInNegativeSCC(innerPattern.Hash()) {
				// Emit conditional (undefined) and register reverse dependency
				if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
					if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
						ds := NewDelaySet()
						ds.Add(innerPattern.Hash())
						parentEntry.QueueDelaySetForNextAnswer(ds)
						wfsTracef("NegEval(%s <- %s): queued delay set (fast-path negative SCC)", currentPredicateID, innerPattern.PredicateID())
						engine.addReverseDependency(innerPattern.Hash(), parentEntry)
						engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerPattern.Hash())
					}
				}
				answers <- map[int64]Term{}
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Otherwise: unconditional success
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
		wfsTracef("NegEval(%s <- %s): initial status=%v count=%d", currentPredicateID, innerPattern.PredicateID(), initialStatus, initialCount)

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
			// No answers: may be Undefined if inner is in a negative-edge SCC
			if engine.isInNegativeSCC(innerPattern.Hash()) {
				if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
					if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
						ds := NewDelaySet()
						ds.Add(innerPattern.Hash())
						parentEntry.QueueDelaySetForNextAnswer(ds)
						wfsTracef("NegEval(%s <- %s): queued delay set (initial complete negative SCC)", currentPredicateID, innerPattern.PredicateID())
						engine.addReverseDependency(innerPattern.Hash(), parentEntry)
						engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
					}
				}
				answers <- map[int64]Term{}
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Otherwise: unconditional success
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
			wfsTracef("NegEval(%s <- %s): change event st=%v cnt=%d", currentPredicateID, innerPattern.PredicateID(), st, cnt)
			if cnt > 0 {
				// Now has answers: negation fails
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			if st == StatusComplete || st == StatusFailed {
				// Completed with no answers: may be Undefined if inner is in a negative-edge SCC
				if engine.isInNegativeSCC(innerPattern.Hash()) {
					if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
						if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
							ds := NewDelaySet()
							ds.Add(innerPattern.Hash())
							parentEntry.QueueDelaySetForNextAnswer(ds)
							wfsTracef("NegEval(%s <- %s): queued delay set (change event negative SCC)", currentPredicateID, innerPattern.PredicateID())
							engine.addReverseDependency(innerPattern.Hash(), parentEntry)
							engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
						}
					}
					answers <- map[int64]Term{}
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				// Otherwise: unconditional success
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
					if engine.isInNegativeSCC(innerPattern.Hash()) {
						if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
							if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
								ds := NewDelaySet()
								ds.Add(innerPattern.Hash())
								parentEntry.QueueDelaySetForNextAnswer(ds)
								wfsTracef("NegEval(%s <- %s): queued delay set (started immediate complete negative SCC)", currentPredicateID, innerPattern.PredicateID())
								engine.addReverseDependency(innerPattern.Hash(), parentEntry)
								engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
							}
						}
						answers <- map[int64]Term{}
						go func() {
							for range innerCh {
							}
						}()
						return nil
					}
					answers <- map[int64]Term{}
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				// Still active: fall through to emit conditional
			case <-startedCh:
				// Producer started; re-check state to catch immediate completion or first answer
				wfsTracef("NegEval(%s <- %s): started signal", currentPredicateID, innerPattern.PredicateID())
				st2 := innerEntry.Status()
				cnt2 := innerEntry.Answers().Count()
				if cnt2 > 0 {
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				if st2 == StatusComplete || st2 == StatusFailed {
					if engine.isInNegativeSCC(innerPattern.Hash()) {
						if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
							if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
								ds := NewDelaySet()
								ds.Add(innerPattern.Hash())
								parentEntry.QueueDelaySetForNextAnswer(ds)
								wfsTracef("NegEval(%s <- %s): queued delay set (started then change negative SCC)", currentPredicateID, innerPattern.PredicateID())
								engine.addReverseDependency(innerPattern.Hash(), parentEntry)
								engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
							}
						}
						answers <- map[int64]Term{}
						go func() {
							for range innerCh {
							}
						}()
						return nil
					}
					answers <- map[int64]Term{}
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				// Still active: prefer an immediate change if available, else emit conditional
				select {
				case <-waitCh:
					st3 := innerEntry.Status()
					cnt3 := innerEntry.Answers().Count()
					if cnt3 > 0 {
						go func() {
							for range innerCh {
							}
						}()
						return nil
					}
					if st3 == StatusComplete || st3 == StatusFailed {
						if engine.isInNegativeSCC(innerPattern.Hash()) {
							if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
								if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
									ds := NewDelaySet()
									ds.Add(innerPattern.Hash())
									parentEntry.QueueDelaySetForNextAnswer(ds)
									engine.addReverseDependency(innerPattern.Hash(), parentEntry)
									engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
								}
							}
							answers <- map[int64]Term{}
							go func() {
								for range innerCh {
								}
							}()
							return nil
						}
						answers <- map[int64]Term{}
						go func() {
							for range innerCh {
							}
						}()
						return nil
					}
					// Still active: fall through to emit conditional
				default:
					// no immediate change; emit conditional below
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Inner is active with no answers yet: emit conditional answer
		// Queue delay set and register reverse dependency so engine can
		// simplify or retract this answer later based on child outcome.
		// Before emitting conditional, perform final non-blocking checks to catch
		// immediate completion/answer that may have occurred just after Started.
		// First, try a non-blocking read from innerCh to see if it's already
		// closed (no answers) or has produced at least one answer.
		select {
		case _, ok := <-innerCh:
			if ok {
				// Inner produced at least one answer: negation fails
				wfsTracef("NegEval(%s <- %s): final nb read saw answer -> fail", currentPredicateID, innerPattern.PredicateID())
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			// Channel closed with no answers: unconditional success
			wfsTracef("NegEval(%s <- %s): final nb read saw closed -> unconditional", currentPredicateID, innerPattern.PredicateID())
			answers <- map[int64]Term{}
			go func() {
				for range innerCh {
				}
			}()
			return nil
		default:
		}
		// Next, check if a change event has just fired.
		select {
		case <-waitCh:
			st := innerEntry.Status()
			cnt := innerEntry.Answers().Count()
			wfsTracef("NegEval(%s <- %s): final nb change st=%v cnt=%d", currentPredicateID, innerPattern.PredicateID(), st, cnt)
			if cnt > 0 {
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
			if st == StatusComplete || st == StatusFailed {
				if engine.isInNegativeSCC(innerPattern.Hash()) {
					if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
						if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
							ds := NewDelaySet()
							ds.Add(innerPattern.Hash())
							parentEntry.QueueDelaySetForNextAnswer(ds)
							engine.addReverseDependency(innerPattern.Hash(), parentEntry)
							engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
						}
					}
					answers <- map[int64]Term{}
					go func() {
						for range innerCh {
						}
					}()
					return nil
				}
				answers <- map[int64]Term{}
				go func() {
					for range innerCh {
					}
				}()
				return nil
			}
		default:
			// No change detected; proceed to emit conditional below
		}

		if parentRaw := ctx.Value(slgProducerEntryKey{}); parentRaw != nil {
			if parentEntry, ok := parentRaw.(*SubgoalEntry); ok {
				ds := NewDelaySet()
				ds.Add(innerPattern.Hash())
				parentEntry.QueueDelaySetForNextAnswer(ds)
				wfsTracef("NegEval(%s <- %s): queued delay set (final conditional)", currentPredicateID, innerPattern.PredicateID())
				engine.addReverseDependency(innerPattern.Hash(), parentEntry)
				// Record negative dependency (parent depends negatively on inner)
				engine.addNegativeEdge(parentEntry.Pattern().Hash(), innerEntry.Pattern().Hash())
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
