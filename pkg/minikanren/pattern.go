// Package minikanren provides pattern matching operators for miniKanren.
//
// Pattern matching is a fundamental operation in logic programming that
// allows matching terms against multiple patterns and executing corresponding
// goals. This module provides three pattern matching primitives following
// core.logic conventions:
//
//   - Matche: Exhaustive pattern matching (tries all matching clauses)
//   - Matcha: Committed choice pattern matching (first match wins)
//   - Matchu: Unique pattern matching (requires exactly one match)
//
// These operators significantly reduce boilerplate compared to manual
// combinations of Conde, Conj, and destructuring with Car/Cdr.
package minikanren

import (
	"context"
	"fmt"
)

// PatternClause represents a single pattern matching clause.
// Each clause consists of a pattern term and a sequence of goals to execute
// if the pattern matches.
//
// The pattern is unified with the input term. If unification succeeds,
// the goals are executed in sequence (as if by Conj).
type PatternClause struct {
	// Pattern is the term to match against. Can contain fresh variables
	// that will be bound if the pattern matches.
	Pattern Term

	// Goals are executed in sequence if the pattern matches.
	// An empty goal list succeeds immediately (useful for filtering).
	Goals []Goal
}

// NewClause creates a pattern matching clause from a pattern and goals.
// This is a convenience constructor for PatternClause.
//
// Example:
//
//	clause := NewClause(Nil, Eq(result, NewAtom(0)))
func NewClause(pattern Term, goals ...Goal) PatternClause {
	return PatternClause{
		Pattern: pattern,
		Goals:   goals,
	}
}

// Matche performs exhaustive pattern matching over multiple clauses.
// It tries to match the input term against each clause's pattern, executing
// the corresponding goals for ALL matching clauses.
//
// This is similar to Conde - multiple clauses can match and produce solutions.
// Each matching clause generates a separate branch in the search tree.
//
// Semantics:
//   - For each clause, unify term with clause.Pattern
//   - If unification succeeds, execute clause.Goals
//   - Combine all successful branches with Disj (disjunction)
//
// Example:
//
//	// Classify list length
//	Matche(list,
//	    NewClause(Nil, Eq(result, NewAtom("empty"))),
//	    NewClause(NewPair(Fresh("_"), Nil), Eq(result, NewAtom("singleton"))),
//	    NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))),
//	        Eq(result, NewAtom("multiple"))),
//	)
func Matche(term Term, clauses ...PatternClause) Goal {
	if len(clauses) == 0 {
		return func(ctx context.Context, store ConstraintStore) *Stream {
			stream := NewStream()
			stream.Close()
			return stream
		}
	}

	// Build a disjunction of all matching clauses
	goals := make([]Goal, 0, len(clauses))

	for _, clause := range clauses {
		// Capture clause for closure
		c := clause

		// Create a goal that tries this pattern
		clauseGoal := func(ctx context.Context, store ConstraintStore) *Stream {
			// First unify with pattern
			unifyGoal := Eq(term, c.Pattern)
			afterUnify := unifyGoal(ctx, store)

			// Then execute clause goals if unification succeeded
			if len(c.Goals) == 0 {
				// No goals means just check the pattern
				return afterUnify
			}

			// Execute goals in sequence for each successful unification
			stream := NewStream()
			go func() {
				defer stream.Close()

				for {
					stores, hasMore := afterUnify.Take(1)
					if len(stores) == 0 {
						if !hasMore {
							break
						}
						continue
					}

					// Execute all goals in sequence (conjunction)
					conjGoal := Conj(c.Goals...)
					resultStream := conjGoal(ctx, stores[0])

					// Forward results
					for {
						results, more := resultStream.Take(10)
						for _, r := range results {
							stream.Put(r)
						}
						if !more {
							break
						}
					}
				}
			}()

			return stream
		}

		goals = append(goals, clauseGoal)
	}

	// Combine all clause goals with disjunction
	return Disj(goals...)
}

// Matcha performs committed choice pattern matching.
// It tries each clause in order and commits to the first matching pattern.
// Once a pattern matches, subsequent clauses are not tried, even if the
// committed clause fails during goal execution.
//
// This is more efficient than Matche when you know patterns are mutually
// exclusive or you want deterministic pattern selection.
//
// Semantics:
//   - Try clauses in order
//   - Commit to first matching pattern
//   - Execute that clause's goals
//   - Do NOT try subsequent clauses even if goals fail
//
// Example:
//
//	// Safe head of list (deterministic)
//	Matcha(list,
//	    NewClause(Nil, Eq(result, NewAtom("error"))),
//	    NewClause(NewPair(Fresh("h"), Fresh("_")), Eq(result, Fresh("h"))),
//	)
func Matcha(term Term, clauses ...PatternClause) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Try each clause in order
			for _, clause := range clauses {
				// Check for cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Try to unify with this pattern
				unifyGoal := Eq(term, clause.Pattern)
				afterUnify := unifyGoal(ctx, store)

				// Take one result to check if pattern matched
				stores, hasMore := afterUnify.Take(1)
				if len(stores) == 0 {
					// Pattern didn't match, try next clause
					if !hasMore {
						continue
					}
					continue
				}

				// Pattern matched! Commit to this clause
				matchedStore := stores[0]

				if len(clause.Goals) == 0 {
					// No goals, just return the matched state
					stream.Put(matchedStore)
					return
				}

				// Execute clause goals
				conjGoal := Conj(clause.Goals...)
				resultStream := conjGoal(ctx, matchedStore)

				// Forward all results from this clause
				for {
					results, more := resultStream.Take(10)
					for _, r := range results {
						stream.Put(r)
					}
					if !more {
						break
					}
				}

				// Committed - don't try other clauses
				return
			}

			// No clauses matched - fail by closing stream with no results
		}()

		return stream
	}
}

// Matchu performs unique pattern matching.
// It requires that exactly one clause matches. If zero or multiple clauses
// match, the goal fails.
//
// This is useful for enforcing pattern exclusivity and catching ambiguous
// cases during development.
//
// Semantics:
//   - Try to match each clause's pattern (without executing goals yet)
//   - If zero matches: fail
//   - If multiple matches: fail
//   - If exactly one match: execute that clause's goals
//
// Example:
//
//	// Enforce unique classification
//	Matchu(value,
//	    NewClause(NewAtom("small"), LessThan(value, 10)),
//	    NewClause(NewAtom("medium"), Conj(GreaterThanEq(value, 10), LessThan(value, 100))),
//	    NewClause(NewAtom("large"), GreaterThanEq(value, 100)),
//	)
func Matchu(term Term, clauses ...PatternClause) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// First pass: determine which patterns match
			var matchedClause *PatternClause
			var matchedStore ConstraintStore
			matchCount := 0

			for i := range clauses {
				clause := &clauses[i]

				// Check for cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Try to unify with this pattern
				unifyGoal := Eq(term, clause.Pattern)
				afterUnify := unifyGoal(ctx, store)

				// Check if pattern matches
				stores, hasMore := afterUnify.Take(1)
				if len(stores) > 0 {
					matchCount++
					if matchCount == 1 {
						matchedClause = clause
						matchedStore = stores[0]
					} else {
						// Multiple matches - fail
						return
					}
				} else if !hasMore {
					// Pattern didn't match, continue to next
					continue
				}
			}

			// Check uniqueness constraint
			if matchCount == 0 {
				// No matches - fail
				return
			}

			if matchCount > 1 {
				// Multiple matches - fail
				return
			}

			// Exactly one match - execute its goals
			if len(matchedClause.Goals) == 0 {
				stream.Put(matchedStore)
				return
			}

			conjGoal := Conj(matchedClause.Goals...)
			resultStream := conjGoal(ctx, matchedStore)

			// Forward all results
			for {
				results, more := resultStream.Take(10)
				for _, r := range results {
					stream.Put(r)
				}
				if !more {
					break
				}
			}
		}()

		return stream
	}
}

// MatcheList is a convenience wrapper for matching lists with specific patterns.
// It handles common list matching scenarios with cleaner syntax.
//
// Clauses are specified as:
//   - Empty list: NewClause(Nil, goals...)
//   - Singleton: NewClause(NewPair(element, Nil), goals...)
//   - Cons: NewClause(NewPair(head, tail), goals...)
//
// Example:
//
//	MatcheList(list,
//	    NewClause(Nil, Eq(sum, NewAtom(0))),
//	    NewClause(NewPair(x, rest), Conj(
//	        SumList(rest, restSum),
//	        Eq(sum, Add(x, restSum)),
//	    )),
//	)
func MatcheList(list Term, clauses ...PatternClause) Goal {
	// Validate that all patterns are list-like
	for i, clause := range clauses {
		if !isListPattern(clause.Pattern) {
			// Create a failing goal with error message
			return func(ctx context.Context, store ConstraintStore) *Stream {
				stream := NewStream()
				go func() {
					defer stream.Close()
					// Log error but fail gracefully
					_ = fmt.Errorf("MatcheList: clause %d pattern is not a list pattern", i)
				}()
				return stream
			}
		}
	}

	return Matche(list, clauses...)
}

// isListPattern checks if a term is a valid list pattern (Nil or Pair).
func isListPattern(t Term) bool {
	if t.Equal(Nil) {
		return true
	}
	if _, ok := t.(*Pair); ok {
		return true
	}
	// Variables are valid list patterns
	if _, ok := t.(*Var); ok {
		return true
	}
	return false
}
