// Package minikanren provides hybrid integration helpers for combining pldb
// relational queries with finite-domain constraint solving.
//
// This file implements convenience functions that reduce boilerplate when
// working with both pldb databases and FD constraints. The helpers maintain
// the compositional design while making common patterns more ergonomic.
//
// Design Philosophy:
//   - Explicit over implicit: Users control when FD filtering happens
//   - Compositional: Helpers wrap existing primitives without magic
//   - Thread-safe: All operations safe for concurrent use
//   - Zero overhead: No performance penalty vs manual implementation
//
// The key insight is that pldb queries and FD constraints are separate
// concerns that can be composed at the Goal level. These helpers encapsulate
// proven patterns from the test suite into reusable library functions.
package minikanren

import (
	"context"
)

// FDFilteredQuery creates a Goal that queries a database and automatically
// filters results based on an FD variable's domain. This is the recommended
// way to integrate pldb with FD constraints.
//
// The function builds a query over the relation, then filters those results to
// include only values where filterVar's binding is present in fdVar's domain.
// This implements the "FD domains filter database queries" pattern.
//
// Parameters:
//   - db: The database to query
//   - rel: The relation to query
//   - fdVar: The FD variable whose domain constrains the results
//   - filterVar: The relational variable to filter (must appear in queryTerms)
//   - queryTerms: The complete list of terms for the query (must include filterVar)
//
// The filterVar is typically a Fresh variable that will be bound by the query.
// The fdVar is an FD variable from a Model whose domain has been constrained.
// The queryTerms specify the complete query pattern - filterVar must appear
// somewhere in this list.
//
// Thread Safety: Safe for concurrent use. Each invocation creates independent
// stream processing that doesn't share mutable state.
//
// Example:
//
//	// Database of employees with ages
//	employee, _ := DbRel("employee", 2, 0)
//	db, _ := db.AddFact(employee, NewAtom("alice"), NewAtom(28))
//	db, _ = db.AddFact(employee, NewAtom("bob"), NewAtom(42))
//
//	// FD model constrains age to [25, 35]
//	model := NewModel()
//	ageVar := model.NewVariable(NewBitSetDomainFromValues(50, []int{25,26,27,28,29,30,31,32,33,34,35}))
//
//	// Query with automatic filtering
//	name := Fresh("name")
//	age := Fresh("age")
//	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)  // query: (employee name age)
//
//	// Execute - only employees aged 25-35 returned
//	store := NewUnifiedStore()
//	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
//	adapter := NewUnifiedStoreAdapter(store)
//	results, _ := goal(context.Background(), adapter).Take(10)
//	// results contains only alice (age 28), not bob (age 42)
func FDFilteredQuery(
	db *Database,
	rel *Relation,
	fdVar *FDVariable,
	filterVar *Var,
	queryTerms ...Term,
) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Build the complete query with all terms
		baseQuery := db.Query(rel, queryTerms...)

		// Execute base query
		dbStream := baseQuery(ctx, store)

		// Create filtered output stream
		filteredStream := NewStream()

		go func() {
			defer filteredStream.Close()

			for {
				// Process one result at a time to avoid buffering
				results, hasMore := dbStream.Take(1)
				if len(results) == 0 {
					if !hasMore {
						break
					}
					continue
				}

				result := results[0]

				// Get the binding for the filter variable
				binding := result.GetBinding(filterVar.ID())
				if binding == nil {
					// Variable not bound - shouldn't happen with proper query
					// but pass through to maintain compositionality
					filteredStream.Put(result)
					continue
				}

				// Check if result is from a hybrid store with FD domains
				if adapter, ok := result.(*UnifiedStoreAdapter); ok {
					domain := adapter.GetDomain(fdVar.ID())
					if domain == nil {
						// No FD domain set - pass through
						filteredStream.Put(result)
						continue
					}

					// Extract integer value from binding
					if atom, ok := binding.(*Atom); ok {
						if val, ok := atom.value.(int); ok {
							// Check FD domain membership
							if domain.Has(val) {
								filteredStream.Put(result)
							}
							// If value not in domain, filter out (don't put)
						} else {
							// Binding is not an integer - pass through
							// (FD constraints only apply to integers)
							filteredStream.Put(result)
						}
					} else {
						// Binding is not an atom - pass through
						filteredStream.Put(result)
					}
				} else {
					// Not a hybrid store - pass through without filtering
					filteredStream.Put(result)
				}
			}
		}()

		return filteredStream
	}
}

// MapQueryResult extracts a binding from a query result and maps it to an
// FD variable in the UnifiedStore. This encapsulates the manual mapping
// pattern used when propagating database facts to FD domains.
//
// This is a convenience function for the common operation:
//
//	binding := result.GetBinding(relVar.ID())
//	store, err = store.AddBinding(int64(fdVar.ID()), binding)
//
// Parameters:
//   - result: The query result containing bindings
//   - relVar: The relational variable to extract from result
//   - fdVar: The FD variable to bind in the store
//   - store: The current UnifiedStore
//
// Returns the updated store with the new binding, or the original store
// if relVar has no binding in result.
//
// Thread Safety: Safe for concurrent use. UnifiedStore operations are
// immutable and return new store instances.
//
// Example:
//
//	// Query for age
//	goal := db.Query(employee, NewAtom("alice"), age)
//	results, _ := goal(ctx, adapter).Take(1)
//
//	// Map result to FD variable
//	store, err := MapQueryResult(results[0], age, ageVar, store)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Now ageVar has the binding from the query
func MapQueryResult(
	result ConstraintStore,
	relVar *Var,
	fdVar *FDVariable,
	store *UnifiedStore,
) (*UnifiedStore, error) {
	if result == nil || relVar == nil || fdVar == nil || store == nil {
		return store, nil
	}

	binding := result.GetBinding(relVar.ID())
	if binding == nil {
		// No binding for this variable - return unchanged
		return store, nil
	}

	// Add binding to the FD variable in the unified store
	return store.AddBinding(int64(fdVar.ID()), binding)
}

// HybridConj creates a Goal that combines multiple FD-filtered queries with
// conjunction. This is useful when multiple database facts need to be checked
// against different FD constraints.
//
// Each query is executed and filtered independently, then results are combined
// via conjunction (all queries must succeed for the overall goal to succeed).
//
// Parameters:
//   - goals: Variable number of Goals to execute in conjunction
//
// Returns a Goal that succeeds only if all input goals succeed, combining
// their bindings in the result.
//
// Thread Safety: Safe for concurrent use.
//
// Example:
//
//	// Two queries with different FD constraints
//	goal1 := FDFilteredQuery(db, employee, ageVar, age, name)
//	goal2 := FDFilteredQuery(db, salary, salaryVar, sal, name)
//
//	// Both must succeed
//	combined := HybridConj(goal1, goal2)
//	results, _ := combined(ctx, adapter).Take(10)
func HybridConj(goals ...Goal) Goal {
	return Conj(goals...)
}

// HybridDisj creates a Goal that combines multiple FD-filtered queries with
// disjunction. This is useful when any of several database facts can satisfy
// the constraint.
//
// Each query is executed and filtered independently, then results are combined
// via disjunction (any query succeeding makes the overall goal succeed).
//
// Parameters:
//   - goals: Variable number of Goals to execute in disjunction
//
// Returns a Goal that succeeds if any input goal succeeds.
//
// Thread Safety: Safe for concurrent use.
//
// Example:
//
//	// Either constraint can be satisfied
//	youngGoal := FDFilteredQuery(db, employee, youngAgeVar, age, name)
//	seniorGoal := FDFilteredQuery(db, employee, seniorAgeVar, age, name)
//
//	// Accept either
//	eitherGoal := HybridDisj(youngGoal, seniorGoal)
//	results, _ := eitherGoal(ctx, adapter).Take(10)
func HybridDisj(goals ...Goal) Goal {
	return Disj(goals...)
}
