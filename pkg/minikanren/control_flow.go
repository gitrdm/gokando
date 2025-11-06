// Package minikanren provides advanced control flow operators that extend
// the core conjunction (Conj) and disjunction (Disj/Conde) primitives.
//
// # Control Flow Operators
//
// This package implements four fundamental control flow operators inspired by
// Prolog and advanced logic programming systems:
//
//   - Ifa: If-then-else with backtracking through all condition solutions
//   - Ifte: If-then-else with commitment to first condition solution
//   - SoftCut: Prolog-style soft cut (*->) for conditional commitment
//   - CallGoal: Meta-call for indirect goal invocation
//
// # Design Philosophy
//
// These operators are implemented using the existing Goal/Stream and
// ConstraintStore interfaces with no special runtime support. They respect
// context cancellation and integrate seamlessly with the SLG tabling system.
//
// # Variable Scoping
//
// CRITICAL: All variables used in control flow goals must be created inside
// the Run closure to ensure proper projection and substitution:
//
//	// CORRECT - variables inside closure
//	Run(5, func(q *Var) Goal {
//	    x := Fresh("x")
//	    return Ifa(Eq(x, NewAtom(1)), Eq(q, x), Eq(q, NewAtom("none")))
//	})
//
//	// WRONG - variables outside closure (will return unbound)
//	x := Fresh("x")
//	Run(5, func(q *Var) Goal {
//	    return Ifa(Eq(x, NewAtom(1)), Eq(q, x), Eq(q, NewAtom("none")))
//	})
//
// # Search Behavior
//
// The operators differ in how they handle multiple solutions from the condition:
//
//   - Ifa: Evaluates thenGoal for EACH solution of condition; if condition fails, evaluates elseGoal
//   - Ifte: Commits to FIRST solution of condition and evaluates thenGoal; if condition fails, evaluates elseGoal
//   - SoftCut: Synonym for Ifte with Prolog-compatible semantics
//
// # Integration with SLG Tabling
//
// These operators are compatible with SLG/WFS tabling. They do not execute
// goals during pattern construction, avoiding circular dependencies. All goal
// evaluation happens lazily during stream consumption.
package minikanren

import (
	"context"
)

// Ifa implements if-then-else with backtracking through all condition solutions.
//
// Semantics: Ifa(C, T, E) = (C, T) ∨ (¬C, E)
//
// If the condition has one or more solutions, thenGoal is evaluated for each
// solution and all results are collected (via disjunction). If the condition
// has no solutions, elseGoal is evaluated once with the original store.
//
// # Examples
//
//	// Conditional with multiple solutions - both paths explored
//	Ifa(
//	    Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),  // x = 1 or x = 2
//	    Eq(q, x),                                      // then: q = x (two solutions)
//	    Eq(q, NewAtom("none"))                         // else: not reached
//	)
//	// Results: q = 1, q = 2
//
//	// Condition failure - else branch taken
//	Ifa(
//	    Eq(NewAtom(1), NewAtom(2)),  // fails
//	    Eq(q, NewAtom("success")),   // not evaluated
//	    Eq(q, NewAtom("failure"))    // evaluated once
//	)
//	// Results: q = "failure"
//
// # Performance
//
// Ifa explores the full search space of the condition, which can be expensive
// for conditions with many solutions. Use Ifte for early commitment when only
// the first solution is needed.
//
// # Integration Notes
//
// Works with SLG tabling. Variables in condition, thenGoal, and elseGoal must
// be properly scoped within the Run closure for correct substitution.
func Ifa(condition, thenGoal, elseGoal Goal) Goal {
	if condition == nil || thenGoal == nil || elseGoal == nil {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Evaluate condition to determine which branch to take
		condStream := condition(ctx, store)
		if condStream == nil {
			// Condition construction failed - take else branch
			return elseGoal(ctx, store)
		}

		// Check if condition produces any solutions
		firstBatch, hasMore := condStream.Take(1)
		if len(firstBatch) == 0 {
			// Condition failed - evaluate else branch with original store
			return elseGoal(ctx, store)
		}

		// Condition succeeded at least once - evaluate thenGoal for ALL solutions
		// This implements the full backtracking semantics of (C, T)
		out := NewStream()
		go func() {
			defer out.Close()

			// Process first solution
			for _, s := range firstBatch {
				select {
				case <-ctx.Done():
					return
				default:
				}
				thenStream := thenGoal(ctx, s)
				if thenStream != nil {
					for {
						solutions, more := thenStream.Take(10)
						for _, result := range solutions {
							select {
							case <-ctx.Done():
								return
							default:
								out.Put(result)
							}
						}
						if !more {
							break
						}
					}
				}
			}

			// Process remaining condition solutions
			if hasMore {
				for {
					batch, more := condStream.Take(10)
					for _, s := range batch {
						select {
						case <-ctx.Done():
							return
						default:
						}
						thenStream := thenGoal(ctx, s)
						if thenStream != nil {
							for {
								solutions, m := thenStream.Take(10)
								for _, result := range solutions {
									select {
									case <-ctx.Done():
										return
									default:
										out.Put(result)
									}
								}
								if !m {
									break
								}
							}
						}
					}
					if !more {
						break
					}
				}
			}
		}()

		return out
	}
}

// Ifte implements if-then-else with commitment to the first condition solution.
//
// Semantics: Ifte(C, T, E) commits to the first solution of C and evaluates T;
// if C has no solutions, evaluates E.
//
// This is the "committed choice" variant where once the condition succeeds,
// we commit to that solution and ignore any other solutions the condition
// might produce. This is useful for deterministic control flow and optimization.
//
// # Examples
//
//	// Commits to first solution of condition
//	Ifte(
//	    Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),  // x = 1 or x = 2
//	    Eq(q, x),                                      // then: q = x (first solution only)
//	    Eq(q, NewAtom("none"))                         // else: not reached
//	)
//	// Results: q = 1 (commits to first, ignores x = 2)
//
//	// Condition failure - else branch taken
//	Ifte(
//	    Eq(NewAtom(1), NewAtom(2)),  // fails
//	    Eq(q, NewAtom("success")),   // not evaluated
//	    Eq(q, NewAtom("failure"))    // evaluated once
//	)
//	// Results: q = "failure"
//
// # Prolog Comparison
//
// This implements Prolog's (C -> T ; E) semantics, also known as soft cut.
// It's "soft" because it only cuts within the condition, not affecting outer
// choice points.
//
// # Performance
//
// Ifte is more efficient than Ifa when you only need the first solution,
// as it avoids exploring the full search space of the condition.
//
// # Integration Notes
//
// Works with SLG tabling. The commitment happens at the stream level, so
// tabled predicates in the condition still cache all their answers (the
// commitment only affects which answers we consume).
func Ifte(condition, thenGoal, elseGoal Goal) Goal {
	if condition == nil || thenGoal == nil || elseGoal == nil {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Evaluate condition to determine which branch to take
		condStream := condition(ctx, store)
		if condStream == nil {
			// Condition construction failed - take else branch
			return elseGoal(ctx, store)
		}

		// Take only the first solution (committed choice)
		firstSolution, _ := condStream.Take(1)
		if len(firstSolution) == 0 {
			// Condition failed - evaluate else branch with original store
			return elseGoal(ctx, store)
		}

		// Condition succeeded - commit to first solution and evaluate thenGoal
		// We ignore any additional solutions from the condition stream
		return thenGoal(ctx, firstSolution[0])
	}
}

// SoftCut implements Prolog's soft cut operator (*->).
//
// This is a synonym for Ifte provided for Prolog compatibility. The name
// "soft cut" reflects that it cuts within the condition (commits to first
// solution) but doesn't affect outer choice points.
//
// Semantics: SoftCut(C, T, E) ≡ Ifte(C, T, E) ≡ (C *-> T ; E)
//
// # Examples
//
//	// Prolog-style conditional with commitment
//	SoftCut(
//	    member(X, List),      // condition
//	    Eq(q, X),             // then (commits to first member)
//	    Eq(q, NewAtom("empty"))  // else
//	)
//
// # Performance
//
// Identical to Ifte - commits to first condition solution.
//
// # Integration Notes
//
// This is purely a naming convenience for developers familiar with Prolog.
// Use Ifte for a more descriptive name in Go code.
func SoftCut(condition, thenGoal, elseGoal Goal) Goal {
	return Ifte(condition, thenGoal, elseGoal)
}

// CallGoal implements meta-call for indirect goal invocation.
//
// This allows goals to be stored in terms and invoked dynamically, enabling
// higher-order logic programming patterns like meta-interpreters, dynamic
// rule construction, and goal factories.
//
// The goalTerm must be an Atom wrapping a Goal function. When evaluated,
// CallGoal extracts the goal and invokes it with the current context and store.
//
// # Examples
//
//	// Store a goal in a term and call it
//	Run(5, func(q *Var) Goal {
//	    g := NewAtom(Eq(q, NewAtom("hello")))
//	    return CallGoal(g)
//	})
//	// Results: q = "hello"
//
//	// Dynamic goal selection
//	Run(5, func(q *Var) Goal {
//	    x := Fresh("x")
//	    chooseGoal := func(choice int) Goal {
//	        if choice == 1 {
//	            return Eq(q, NewAtom("first"))
//	        }
//	        return Eq(q, NewAtom("second"))
//	    }
//	    return Conj(
//	        Eq(x, NewAtom(1)),
//	        CallGoal(NewAtom(chooseGoal(1)))
//	    )
//	})
//
// # Type Safety
//
// CallGoal performs runtime type checking. If goalTerm is not an Atom or
// doesn't contain a Goal function, it returns a failure stream. This is
// necessary because Go's type system can't enforce the constraint statically.
//
// # Performance
//
// The overhead of CallGoal is minimal - just a type assertion and function
// call. The goal itself is evaluated normally once extracted.
//
// # Integration Notes
//
// Works with SLG tabling. If the called goal is tabled, it will use the
// tabling infrastructure normally. CallGoal itself doesn't interfere with
// tabling semantics.
//
// # Meta-Programming Patterns
//
// CallGoal enables several advanced patterns:
//
//   - Meta-interpreters: Implement custom evaluation strategies
//   - Goal factories: Generate goals based on runtime data
//   - Dynamic rules: Construct and invoke rules at runtime
//   - Higher-order predicates: Pass goals as arguments to other goals
func CallGoal(goalTerm Term) Goal {
	if goalTerm == nil {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Walk the term to handle variable bindings
		walked := store.GetSubstitution().Walk(goalTerm)
		if walked == nil {
			// Create and immediately close an empty stream
			s := NewStream()
			s.Close()
			return s
		}

		// Extract goal from atom
		atom, ok := walked.(*Atom)
		if !ok {
			// Not an atom - fail
			s := NewStream()
			s.Close()
			return s
		}

		// Type assert to Goal
		goal, ok := atom.value.(Goal)
		if !ok {
			// Atom doesn't contain a Goal - fail
			s := NewStream()
			s.Close()
			return s
		}

		// Invoke the goal
		return goal(ctx, store)
	}
}
