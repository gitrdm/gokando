// Package minikanren provides integration between pldb relational database
// and SLG tabling for efficient recursive query evaluation.
//
// # Integration Architecture
//
// pldb queries normally return Goals that can be composed with Conj/Disj.
// SLG tabling requires GoalEvaluators that yield answer bindings via channels.
// This file bridges the two by providing:
//
//   - TabledQuery: Wraps Database.Query for use with SLG tabling
//   - RecursiveRule: Helper for defining recursive rules with pldb base cases
//   - QueryEvaluator: Converts pldb queries to GoalEvaluator format
//
// # Usage Pattern
//
//	// Define base facts
//	edge := DbRel("edge", 2, 0, 1)
//	db := NewDatabase()
//	db = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
//
//	// Define recursive rule with tabling
//	path := func(x, y Term) Goal {
//	    return TabledQuery(db, edge, x, y, "path", func() Goal {
//	        z := Fresh("z")
//	        return Conj(
//	            TabledQuery(db, edge, x, z, "path"),
//	            TabledQuery(db, edge, z, y, "path"),
//	        )
//	    })
//	}
//
// This enables terminating recursive queries over pldb relations using SLG's
// fixpoint computation.
package minikanren

import (
	"context"
	"fmt"
)

// QueryEvaluator converts a pldb query Goal into a GoalEvaluator for SLG tabling.
// It evaluates the query goal and extracts bindings for the specified variables,
// yielding them as answer substitutions via the channel.
//
// Parameters:
//   - query: The pldb query goal (from Database.Query)
//   - varIDs: Variable IDs to extract from each answer
//
// Returns a GoalEvaluator that can be passed to SLGEngine.Evaluate.
func QueryEvaluator(query Goal, varIDs ...int64) GoalEvaluator {
	return func(ctx context.Context, answers chan<- map[int64]Term) error {
		// Create a fresh constraint store for evaluation
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Execute the query goal
		stream := query(ctx, store)
		if stream == nil {
			return nil
		}

		// Extract answers from the stream using Take
		const batchSize = 100
		for {
			stores, hasMore := stream.Take(batchSize)

			for _, nextStore := range stores {
				// Extract bindings for requested variables
				answer := make(map[int64]Term, len(varIDs))
				for _, varID := range varIDs {
					binding := nextStore.GetBinding(varID)
					if binding != nil {
						answer[varID] = binding
					}
				}

				// Yield answer
				// For ground queries (no variables), yield empty answer to indicate success
				select {
				case answers <- answer:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			if !hasMore {
				return nil // no more answers
			}

			// Check for cancellation between batches
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}
	}
}

// TabledQuery wraps a pldb query with SLG tabling for recursive evaluation.
// This is the primary integration point between pldb and the SLG engine.
//
// IMPORTANT LIMITATION: TabledQuery is designed for top-level recursive predicates,
// not for use within Conj/Disj with shared variables. The SLG tabling system works
// with isolated answer substitutions, which don't compose well with miniKanren's
// threaded ConstraintStore model when joining.
//
// Use TabledQuery for:
//   - Top-level queries that need caching
//   - Recursive predicates (transitive closure, reachability, etc.)
//   - Independent queries without variable sharing
//
// Use regular Database.Query() for:
//   - Joins with Conj where variables are shared between subgoals
//   - Complex queries mixing pldb with other goals
//   - Non-recursive queries
//
// The function:
//  1. Constructs a CallPattern from the predicate ID and query arguments
//  2. Creates a GoalEvaluator from the pldb query
//  3. Evaluates via the global SLG engine
//  4. Returns a Goal that unifies results with the original pattern
//
// Parameters:
//   - db: The pldb database to query
//   - rel: The relation to query
//   - predicateID: Unique identifier for tabling (e.g., "edge", "path")
//   - args: Query pattern (may contain variables)
//
// Example (correct usage):
//
//	// Top-level tabled query
//	goal := TabledQuery(db, edge, "path", x, y)
//
// Example (incorrect - use regular Query instead):
//
//	// DON'T do this - shared variable p won't unify correctly:
//	Conj(
//	    TabledQuery(db, parent, "parent", gp, p),
//	    TabledQuery(db, parent, "parent", p, gc),  // p won't be bound from first goal
//	)
//
//	// DO this instead:
//	Conj(
//	    db.Query(parent, gp, p),
//	    db.Query(parent, p, gc),
//	)
func TabledQuery(db *Database, rel *Relation, predicateID string, args ...Term) Goal {
	if db == nil || rel == nil {
		return Failure
	}

	if len(args) != rel.Arity() {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Collect variable IDs from the argument pattern
		varIDs := make([]int64, 0, len(args))
		for _, arg := range args {
			if v, ok := arg.(*Var); ok {
				varIDs = append(varIDs, v.id)
			}
		}

		// Build the pldb query
		query := db.Query(rel, args...)

		// Wrap as a GoalEvaluator
		evaluator := QueryEvaluator(query, varIDs...)

		// Build call pattern for SLG tabling
		pattern := NewCallPattern(predicateID, args)

		// Evaluate via SLG engine
		engine := GlobalEngine()
		resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
		if err != nil {
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Create stream from tabled results
		return streamFromAnswers(ctx, store, resultChan, args)
	}
}

// streamFromAnswers converts a channel of SLG answer substitutions into a miniKanren Stream.
// It unifies each answer with the original query pattern variables using Eq goals.
func streamFromAnswers(ctx context.Context, store ConstraintStore, answers <-chan map[int64]Term, pattern []Term) *Stream {
	stream := NewStream()

	go func() {
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case answer, ok := <-answers:
				if !ok {
					return // channel closed, done
				}

				// Build a conjunction of Eq goals to unify variables with their answers
				goals := make([]Goal, 0, len(answer))
				for _, term := range pattern {
					if v, ok := term.(*Var); ok {
						if binding, exists := answer[v.id]; exists {
							// Create Eq goal for this variable
							goals = append(goals, Eq(v, binding))
						}
					}
				}

				// Execute the conjunction on the original store
				var unified *Stream
				if len(goals) > 0 {
					unified = Conj(goals...)(ctx, store)
				} else {
					// No variables - ground query success, just pass through the store
					unified = Success(ctx, store)
				}

				if unified != nil {
					// Take stores from unified stream and put them in our stream
					stores, _ := unified.Take(1)
					for _, s := range stores {
						stream.Put(s)
					}
				}
			}
		}
	}()

	return stream
}

// RecursiveRule defines a recursive pldb query rule with tabling support.
// This helper simplifies common patterns like transitive closure.
//
// The rule combines:
//   - Base case: Direct facts from the database
//   - Recursive case: User-defined recursive logic
//
// Parameters:
//   - db: The pldb database
//   - baseRel: The base relation (e.g., "edge")
//   - predicateID: Unique ID for the recursive predicate (e.g., "path")
//   - args: Query arguments (variables or ground terms)
//   - recursiveGoal: Function that builds the recursive case
//
// Example:
//
//	// path(X, Y) :- edge(X, Y).
//	// path(X, Y) :- edge(X, Z), path(Z, Y).
//	x, y := Fresh("x"), Fresh("y")
//	goal := RecursiveRule(db, edge, "path", []Term{x, y}, func() Goal {
//	    z := Fresh("z")
//	    return Conj(
//	        TabledQuery(db, edge, "edge", x, z),
//	        TabledQuery(db, edge, "path", z, y),
//	    )
//	})
func RecursiveRule(db *Database, baseRel *Relation, predicateID string, args []Term, recursiveGoal func() Goal) Goal {
	if db == nil || baseRel == nil || len(args) != baseRel.Arity() {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Extract variable IDs
		varIDs := make([]int64, 0, len(args))
		for _, arg := range args {
			if v, ok := arg.(*Var); ok {
				varIDs = append(varIDs, v.id)
			}
		}

		// Build the recursive evaluator
		evaluator := func(ctx context.Context, answers chan<- map[int64]Term) error {
			// Base case: facts from database
			baseGoal := db.Query(baseRel, args...)

			// Recursive case: user-defined rule
			recGoal := recursiveGoal()

			// Combine with disjunction
			combined := Disj(baseGoal, recGoal)

			// Evaluate and extract answers using Take
			freshStore := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := combined(ctx, freshStore)
			if stream == nil {
				return nil
			}

			const batchSize = 100
			for {
				stores, hasMore := stream.Take(batchSize)

				for _, nextStore := range stores {
					// Extract bindings
					answer := make(map[int64]Term, len(varIDs))
					for _, varID := range varIDs {
						binding := nextStore.GetBinding(varID)
						if binding != nil {
							answer[varID] = binding
						}
					}

					if len(answer) > 0 {
						select {
						case answers <- answer:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}

				if !hasMore {
					return nil
				}

				// Check cancellation between batches
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
		}

		// Build call pattern
		pattern := NewCallPattern(predicateID, args)

		// Evaluate via SLG engine
		engine := GlobalEngine()
		resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
		if err != nil {
			stream := NewStream()
			stream.Close()
			return stream
		}

		return streamFromAnswers(ctx, store, resultChan, args)
	}
}

// TabledRelation provides a convenient wrapper for creating tabled predicates
// over pldb relations. It returns a constructor function that builds tabled goals.
//
// Example:
//
//	edge := DbRel("edge", 2, 0, 1)
//	db := NewDatabase()
//	db = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
//
//	// Create tabled predicate
//	pathPred := TabledRelation(db, edge, "path")
//
//	// Use it in queries
//	x, y := Fresh("x"), Fresh("y")
//	goal := pathPred(x, y)  // Automatically tabled
func TabledRelation(db *Database, rel *Relation, predicateID string) func(...Term) Goal {
	if db == nil || rel == nil {
		return func(args ...Term) Goal {
			return Failure
		}
	}

	return func(args ...Term) Goal {
		if len(args) != rel.Arity() {
			return Failure
		}
		return TabledQuery(db, rel, predicateID, args...)
	}
}

// InvalidateRelation removes all cached answers for queries involving a specific relation.
// This should be called when the relation's facts change (AddFact/RemoveFact).
//
// Note: This is a conservative invalidation strategy. The SLG engine doesn't currently
// provide fine-grained predicate-level invalidation, so we use global Clear().
// For production use, track which facts affect which cached answers.
//
// Parameters:
//   - predicateID: The predicate identifier used in TabledQuery calls
//
// Example:
//
//	db = db.AddFact(edge, NewAtom("c"), NewAtom("d"))
//	InvalidateRelation("path")  // Clear all cached answers
func InvalidateRelation(predicateID string) {
	// TODO: Implement fine-grained invalidation in SLGEngine
	// For now, clear the entire table when any relation changes
	engine := GlobalEngine()
	engine.Clear()
}

// InvalidateAll clears the entire SLG answer table.
// Use this after major database changes when fine-grained invalidation is impractical.
func InvalidateAll() {
	engine := GlobalEngine()
	engine.Clear()
}

// WithTabledDatabase returns a wrapper that automatically tables all queries
// from a database. This is useful for applications where all queries should be cached.
//
// Example:
//
//	db := NewDatabase()
//	// ... add facts ...
//	tdb := WithTabledDatabase(db, "mydb")
//
//	// All queries are automatically tabled
//	goal := tdb.Query(edge, x, y)
type TabledDatabase struct {
	db       *Database
	idPrefix string
}

// WithTabledDatabase creates a database wrapper that tables all queries.
func WithTabledDatabase(db *Database, idPrefix string) *TabledDatabase {
	return &TabledDatabase{
		db:       db,
		idPrefix: idPrefix,
	}
}

// Query wraps Database.Query with automatic tabling.
func (tdb *TabledDatabase) Query(rel *Relation, args ...Term) Goal {
	if tdb.db == nil || rel == nil {
		return Failure
	}
	predicateID := fmt.Sprintf("%s:%s", tdb.idPrefix, rel.Name())
	return TabledQuery(tdb.db, rel, predicateID, args...)
}

// AddFact delegates to the underlying database and invalidates caches.
func (tdb *TabledDatabase) AddFact(rel *Relation, terms ...Term) (*TabledDatabase, error) {
	newDB, err := tdb.db.AddFact(rel, terms...)
	if err != nil {
		return nil, err
	}

	// Invalidate cached answers for this relation
	predicateID := fmt.Sprintf("%s:%s", tdb.idPrefix, rel.Name())
	InvalidateRelation(predicateID)

	return &TabledDatabase{
		db:       newDB,
		idPrefix: tdb.idPrefix,
	}, nil
}

// RemoveFact delegates to the underlying database and invalidates caches.
func (tdb *TabledDatabase) RemoveFact(rel *Relation, terms ...Term) (*TabledDatabase, error) {
	newDB, err := tdb.db.RemoveFact(rel, terms...)
	if err != nil {
		return nil, err
	}

	predicateID := fmt.Sprintf("%s:%s", tdb.idPrefix, rel.Name())
	InvalidateRelation(predicateID)

	return &TabledDatabase{
		db:       newDB,
		idPrefix: tdb.idPrefix,
	}, nil
}

// FactCount delegates to the underlying database.
func (tdb *TabledDatabase) FactCount(rel *Relation) int {
	if tdb.db == nil {
		return 0
	}
	return tdb.db.FactCount(rel)
}

// AllFacts delegates to the underlying database.
func (tdb *TabledDatabase) AllFacts(rel *Relation) [][]Term {
	if tdb.db == nil {
		return nil
	}
	return tdb.db.AllFacts(rel)
}

// Unwrap returns the underlying Database for operations that don't need tabling.
func (tdb *TabledDatabase) Unwrap() *Database {
	return tdb.db
}
