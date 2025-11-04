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
// TabledQuery properly composes with Conj/Disj by:
//   - Walking variables in the incoming ConstraintStore to get current bindings
//   - Using bound values as ground terms in the tabled query
//   - Only caching based on the effective query pattern after instantiation
//   - Unifying remaining unbound variables with tabled results
//
// This enables tabled queries to work correctly in joins:
//
//	Conj(
//	    TabledQuery(db, parent, "parent", gp, p),    // p unbound, will be bound by results
//	    TabledQuery(db, parent, "parent", p, gc),    // p now bound, used as ground term
//	)
//
// The function:
//  1. Walks all argument variables to get current bindings from store
//  2. Constructs a CallPattern from the instantiated arguments
//  3. Creates a GoalEvaluator from the pldb query
//  4. Evaluates via the global SLG engine with caching
//  5. Returns a Goal that unifies results with remaining unbound variables
//
// Parameters:
//   - db: The pldb database to query
//   - rel: The relation to query
//   - predicateID: Unique identifier for tabling (e.g., "edge", "path")
//   - args: Query pattern (may contain variables or ground terms)
//
// Example:
//
//	// Tabled transitive closure
//	goal := TabledQuery(db, edge, "path", x, y)
//
//	// Tabled join (works correctly now)
//	goal := Conj(
//	    TabledQuery(db, parent, "parent", gp, p),
//	    TabledQuery(db, parent, "parent", p, gc),
//	)
func TabledQuery(db *Database, rel *Relation, predicateID string, args ...Term) Goal {
	if db == nil || rel == nil {
		return Failure
	}

	if len(args) != rel.Arity() {
		return Failure
	}

	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Walk all arguments to get their current bindings from the store
		// This is crucial for correct composition in Conj/Disj
		instantiatedArgs := make([]Term, len(args))
		unboundVars := make([]int64, 0, len(args))

		for i, arg := range args {
			walked := store.GetSubstitution().Walk(arg)
			instantiatedArgs[i] = walked

			// Track which variables are still unbound after walking
			if v, ok := walked.(*Var); ok {
				unboundVars = append(unboundVars, v.id)
			}
		}

		// Build the pldb query with instantiated arguments
		// If a variable was bound in the store, it's now a ground term
		query := db.Query(rel, instantiatedArgs...)

		// Wrap as a GoalEvaluator - only extract bindings for unbound vars
		evaluator := QueryEvaluator(query, unboundVars...)

		// Build call pattern from instantiated arguments
		// This ensures cache hits when the same ground query is repeated
		pattern := NewCallPattern(predicateID, instantiatedArgs)

		// Evaluate via SLG engine
		engine := GlobalEngine()
		resultChan, err := engine.Evaluate(ctx, pattern, evaluator)
		if err != nil {
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Create stream from tabled results
		// Only unify the variables that were unbound when we started
		return streamFromAnswers(ctx, store, resultChan, instantiatedArgs)
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

// TabledRecursivePredicate builds a true recursive, tabled predicate over a base relation.
// It returns a predicate constructor that can be called with arguments to form a Goal.
//
// Parameters:
//   - db: pldb database
//   - baseRel: base relation providing direct facts (e.g., parent for ancestor)
//   - predicateID: unique predicate name for tabling (e.g., "ancestor")
//   - recursive: function that, given a self predicate for recursive calls and
//     the current call arguments, returns the recursive case goal.
//
// The produced predicate P(...args) succeeds if either:
//   - baseRel(...args) holds (base case), or
//   - recursive(self, ...args) holds where self is the tabled predicate itself
//
// Example usage:
//
//	ancestor := TabledRecursivePredicate(db, parent, "ancestor", func(self func(...Term) Goal, args ...Term) Goal {
//	    x, y := args[0], args[1]
//	    z := Fresh("z")
//	    return Conj(
//	        db.Query(parent, x, z),
//	        self(z, y),
//	    )
//	})
//	goal := ancestor(Fresh("x"), Fresh("y"))
func TabledRecursivePredicate(db *Database, baseRel *Relation, predicateID string, recursive func(self func(...Term) Goal, args ...Term) Goal) func(...Term) Goal {
	if db == nil || baseRel == nil || recursive == nil || predicateID == "" {
		return func(...Term) Goal { return Failure }
	}

	// Build an evaluator for a specific argument vector.
	var makeEvaluator func(callArgs []Term) GoalEvaluator
	makeEvaluator = func(callArgs []Term) GoalEvaluator {
		// Extract variable IDs present in call arguments
		varIDs := make([]int64, 0, len(callArgs))
		for _, a := range callArgs {
			if v, ok := a.(*Var); ok {
				varIDs = append(varIDs, v.id)
			}
		}

		// Define self as a tabled call to the same predicate using a fresh evaluator for callArgs.
		self := func(args ...Term) Goal {
			return func(ctx context.Context, store ConstraintStore) *Stream {
				// Instantiate arguments with current bindings
				instantiated := make([]Term, len(args))
				for i, a := range args {
					instantiated[i] = store.GetSubstitution().Walk(a)
				}
				pattern := NewCallPattern(predicateID, instantiated)
				engine := GlobalEngine()
				// Use an evaluator specific to these instantiated args
				ev := makeEvaluator(instantiated)
				resultChan, err := engine.Evaluate(ctx, pattern, ev)
				if err != nil {
					s := NewStream()
					s.Close()
					return s
				}
				return streamFromAnswers(ctx, store, resultChan, instantiated)
			}
		}

		// Build the combined goal for this call: base OR recursive
		return func(ctx context.Context, answers chan<- map[int64]Term) error {
			// Evaluate base first to seed answers for potential self-loops
			emitFromGoal := func(goal Goal) error {
				freshStore := NewLocalConstraintStore(NewGlobalConstraintBus())
				stream := goal(ctx, freshStore)
				if stream == nil {
					return nil
				}
				const batchSize = 100
				for {
					stores, hasMore := stream.Take(batchSize)
					for _, nextStore := range stores {
						answer := make(map[int64]Term, len(varIDs))
						for _, varID := range varIDs {
							if binding := nextStore.GetBinding(varID); binding != nil {
								answer[varID] = binding
							}
						}
						if len(answer) > 0 || len(varIDs) == 0 {
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
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
				}
			}

			// Base case
			if err := emitFromGoal(db.Query(baseRel, callArgs...)); err != nil {
				return err
			}

			// Recursive case
			if err := emitFromGoal(recursive(self, callArgs...)); err != nil {
				return err
			}
			return nil
		}
	}

	// Return the predicate constructor
	return func(args ...Term) Goal {
		if len(args) != baseRel.Arity() {
			return Failure
		}
		return func(ctx context.Context, store ConstraintStore) *Stream {
			// Instantiate with current bindings
			instantiated := make([]Term, len(args))
			for i, a := range args {
				instantiated[i] = store.GetSubstitution().Walk(a)
			}
			pattern := NewCallPattern(predicateID, instantiated)
			engine := GlobalEngine()
			ev := makeEvaluator(instantiated)
			resultChan, err := engine.Evaluate(ctx, pattern, ev)
			if err != nil {
				s := NewStream()
				s.Close()
				return s
			}
			return streamFromAnswers(ctx, store, resultChan, instantiated)
		}
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
// The SLG engine now provides fine-grained predicate-level invalidation, removing
// only the cached answers for the specified predicateID while preserving unrelated
// tabled predicates. This is more efficient than clearing the entire table.
//
// Parameters:
//   - predicateID: The predicate identifier used in TabledQuery calls
//
// Example:
//
//	db = db.AddFact(edge, NewAtom("c"), NewAtom("d"))
//	InvalidateRelation("path")  // Clear only "path" predicate answers
func InvalidateRelation(predicateID string) {
	engine := GlobalEngine()
	engine.ClearPredicate(predicateID)
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
