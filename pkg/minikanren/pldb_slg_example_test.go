package minikanren_test

import (
	"context"
	"fmt"
	"sort"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleTabledQuery demonstrates basic tabled queries over pldb relations.
// Tabling caches query results for reuse, improving performance for repeated queries.
func ExampleTabledQuery() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	x := Fresh("x")
	y := Fresh("y")

	// Tabled query caches results
	goal := TabledQuery(db, edge, "edge", x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 2 edges
}

// ExampleQueryEvaluator shows how to convert pldb queries to SLG GoalEvaluators.
// This is useful for custom tabling scenarios or integration with SLG engine directly.
func ExampleQueryEvaluator() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))

	child := Fresh("child")
	query := db.Query(parent, NewAtom("alice"), child)

	// Convert to GoalEvaluator
	evaluator := QueryEvaluator(query, child.ID())

	ctx := context.Background()
	answers := make(chan map[int64]Term, 10)

	go func() {
		defer close(answers)
		_ = evaluator(ctx, answers)
	}()

	count := 0
	for range answers {
		count++
	}

	fmt.Printf("Alice has %d children\n", count)

	// Output:
	// Alice has 2 children
}

// ExampleTabledRelation demonstrates the convenient wrapper for tabled predicates.
// This creates a reusable predicate function that automatically applies tabling.
func ExampleTabledRelation() {
	// Clear cache for clean test
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

	// Create tabled predicate constructor
	edgePred := TabledRelation(db, edge, "edge_example")

	x := Fresh("x")
	y := Fresh("y")

	// Use it like a normal predicate
	goal := edgePred(x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 3 edges
}

// ExampleWithTabledDatabase shows automatic tabling for all database queries.
// This wrapper ensures all queries are cached without explicit TabledQuery calls.
func ExampleWithTabledDatabase() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	// Wrap database for automatic tabling
	tdb := WithTabledDatabase(db, "mydb")

	x := Fresh("x")
	y := Fresh("y")

	// Regular Query call, but automatically tabled
	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges\n", len(results))

	// Output:
	// Found 1 edges
}

// ExampleWithTabledDatabase_mutation shows cache invalidation on database changes.
// When facts are added or removed, the cache is automatically cleared.
func ExampleWithTabledDatabase_mutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutdb")

	// Add more facts - cache invalidates automatically
	tdb, _ = tdb.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb, _ = tdb.AddFact(edge, NewAtom("c"), NewAtom("d"))

	x := Fresh("x")
	y := Fresh("y")

	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges after additions\n", len(results))

	// Output:
	// Found 3 edges after additions
}

// ExampleTabledQuery_join demonstrates joining tabled relations.
// TabledQuery now correctly handles shared variables in Conj by walking
// the incoming ConstraintStore to instantiate bound variables.
func ExampleTabledQuery_join() {
	InvalidateAll()

	parent, _ := DbRel("parent", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("charlie"), NewAtom("diana"))

	// TabledQuery now works correctly in joins with shared variables
	gp := Fresh("gp")
	gc := Fresh("gc")
	p := Fresh("p")

	goal := Conj(
		TabledQuery(db, parent, "parent_join_ex", gp, p),
		TabledQuery(db, parent, "parent_join_ex", p, gc),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d grandparent relationships\n", len(results))

	// Output:
	// Found 2 grandparent relationships
}

// ExampleInvalidateAll demonstrates clearing the entire tabling cache.
// Useful after major database changes or for benchmarking.
func ExampleInvalidateAll() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache
	goal := TabledQuery(db, edge, "edge_inv", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Clear all cached answers
	InvalidateAll()

	engine := GlobalEngine()
	stats := engine.Stats()

	fmt.Printf("Cached subgoals after invalidation: %d\n", stats.CachedSubgoals)

	// Output:
	// Cached subgoals after invalidation: 0
}

// ExampleTabledQuery_multipleVariables shows querying with multiple variable bindings.
// The SLG engine caches all variable bindings from each answer.
func ExampleTabledQuery_multipleVariables() {
	person, _ := DbRel("person", 3, 0, 1, 2) // name, age, city
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30), NewAtom("nyc"))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25), NewAtom("sf"))
	db, _ = db.AddFact(person, NewAtom("charlie"), NewAtom(35), NewAtom("nyc"))

	name := Fresh("name")
	age := Fresh("age")
	city := Fresh("city")

	// Query all fields
	goal := TabledQuery(db, person, "person", name, age, city)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect names for consistent output
	names := make([]string, 0, len(results))
	for _, s := range results {
		if n := s.GetBinding(name.ID()); n != nil {
			if atom, ok := n.(*Atom); ok {
				names = append(names, atom.Value().(string))
			}
		}
	}
	sort.Strings(names)

	fmt.Printf("Found people: %v\n", names)

	// Output:
	// Found people: [alice bob charlie]
}

// ExampleTabledQuery_groundQuery shows tabled queries with ground terms.
// Even fully ground queries benefit from caching when repeated.
func ExampleTabledQuery_groundQuery() {
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	// Fully ground query - checks existence
	goal := TabledQuery(db, edge, "edge_ground_ex", NewAtom("a"), NewAtom("b"))

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		fmt.Println("Edge a->b exists")
	} else {
		fmt.Println("Edge a->b does not exist")
	}

	// Output:
	// Edge a->b exists
}

// ExampleInvalidateRelation demonstrates fine-grained cache invalidation.
// InvalidateRelation now clears only the specified predicate, leaving other
// cached predicates intact. This is more efficient than clearing the entire cache.
func ExampleInvalidateRelation() {
	// Start with a clean cache to make the example deterministic
	InvalidateAll()

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	x := Fresh("x")
	y := Fresh("y")

	// Populate cache with edge_rel predicate
	goal := TabledQuery(db, edge, "edge_rel", x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	stream.Take(10)

	// Invalidate only the edge_rel predicate
	InvalidateRelation("edge_rel")

	engine := GlobalEngine()
	stats := engine.Stats()

	// With fine-grained invalidation, only edge_rel is cleared
	fmt.Printf("Cache cleared: %v\n", stats.CachedSubgoals == 0)

	// Output:
	// Cache cleared: true
}
