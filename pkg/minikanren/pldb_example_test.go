package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleDbRel demonstrates creating a relation with indexed columns.
// Relations define the structure of facts in pldb, similar to table schemas
// in relational databases.
func ExampleDbRel() {
	// Create a binary relation for parent-child relationships
	// Index both columns for fast lookups
	parent, err := DbRel("parent", 2, 0, 1)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Relation: %s (arity=%d)\n", parent.Name(), parent.Arity())
	fmt.Printf("Column 0 indexed: %v\n", parent.IsIndexed(0))
	fmt.Printf("Column 1 indexed: %v\n", parent.IsIndexed(1))

	// Output:
	// Relation: parent (arity=2)
	// Column 0 indexed: true
	// Column 1 indexed: true
}

// ExampleDatabase_AddFact demonstrates adding facts to a relation.
// Facts are ground terms (no variables) that represent concrete data.
// The database is immutable - each operation returns a new database instance.
func ExampleDatabase_AddFact() {
	parent, _ := DbRel("parent", 2, 0, 1)

	// Start with an empty database
	db := NewDatabase()

	// Add facts using copy-on-write semantics
	db1, _ := db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db2, _ := db1.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
	db3, _ := db2.AddFact(parent, NewAtom("alice"), NewAtom("diana"))

	// Each version maintains its own state
	fmt.Printf("Original: %d facts\n", db.FactCount(parent))
	fmt.Printf("After 1:  %d facts\n", db1.FactCount(parent))
	fmt.Printf("After 2:  %d facts\n", db2.FactCount(parent))
	fmt.Printf("After 3:  %d facts\n", db3.FactCount(parent))

	// Output:
	// Original: 0 facts
	// After 1:  1 facts
	// After 2:  2 facts
	// After 3:  3 facts
}

// ExampleDatabase_Query_simple demonstrates basic pattern matching queries.
// Queries unify patterns with facts, where Fresh variables act as wildcards.
func ExampleDatabase_Query_simple() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"alice", "charlie"},
		[]interface{}{"bob", "diana"},
	)

	// Query: Who are alice's children?
	child := Fresh("child")
	goal := db.Query(parent, NewAtom("alice"), child)

	// Execute the query
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order
	fmt.Printf("Alice has %d children\n", len(results))

	// Output:
	// Alice has 2 children
}

// ExampleDatabase_Query_join demonstrates using conjunction to join relations.
// This is analogous to SQL joins, enabling complex relational queries.
func ExampleDatabase_Query_join() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	// Query: Find grandparent-grandchild pairs
	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	gp := Fresh("grandparent")
	gc := Fresh("grandchild")
	p := Fresh("parent")

	goal := Conj(
		db.Query(parent, gp, p),
		db.Query(parent, p, gc),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d grandparent relationships\n", len(results))

	// Output:
	// Found 2 grandparent relationships
}

// ExampleDatabase_Query_repeated demonstrates repeated variable constraints.
// When the same variable appears multiple times, it must unify to the same value.
// This is useful for finding self-referential relationships.
func ExampleDatabase_Query_repeated() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "c"}, // self-loop
		[]interface{}{"d", "d"}, // self-loop
	)

	// Query: Find all self-loops
	x := Fresh("x")
	goal := db.Query(edge, x, x) // same variable in both positions

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d self-loops\n", len(results))

	// Output:
	// Found 2 self-loops
}

// ExampleDatabase_Query_datalog demonstrates a more complex datalog-style query.
// This shows how pldb can express recursive logic programs similar to Datalog.
func ExampleDatabase_Query_datalog() {
	edge, _ := DbRel("edge", 2, 0, 1)
	// Build a graph: a -> b -> c
	//                ^-------|
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "a"},
	)

	// Query: Find all nodes reachable from 'a' in exactly 2 hops
	// path2(X, Z) :- edge(X, Y), edge(Y, Z)
	start := NewAtom("a")
	middle := Fresh("middle")
	dest := Fresh("destination")

	goal := Conj(
		db.Query(edge, start, middle),
		db.Query(edge, middle, dest),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Nodes reachable from 'a' in 2 hops:\n")
	for _, r := range results {
		val := r.GetBinding(dest.ID())
		if atom, ok := val.(*Atom); ok {
			fmt.Printf("  %v\n", atom.Value())
		}
	}

	// Output:
	// Nodes reachable from 'a' in 2 hops:
	//   c
}

// ExampleDatabase_RemoveFact demonstrates fact removal with tombstone semantics.
// Removal creates a new database version without physically deleting data,
// enabling efficient copy-on-write and version management.
func ExampleDatabase_RemoveFact() {
	person, _ := DbRel("person", 1, 0)

	// Create database with some people using low-level API versus the HLAPI for demonstration
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"))
	db, _ = db.AddFact(person, NewAtom("bob"))
	db, _ = db.AddFact(person, NewAtom("charlie"))

	fmt.Printf("Before removal: %d people\n", db.FactCount(person))

	// Remove bob
	db2, _ := db.RemoveFact(person, NewAtom("bob"))

	fmt.Printf("After removal:  %d people\n", db2.FactCount(person))

	// Original database unchanged
	fmt.Printf("Original still: %d people\n", db.FactCount(person))

	// Facts can be re-added
	db3, _ := db2.AddFact(person, NewAtom("bob"))
	fmt.Printf("After re-add:   %d people\n", db3.FactCount(person))

	// Output:
	// Before removal: 3 people
	// After removal:  2 people
	// Original still: 3 people
	// After re-add:   3 people
}

// ExampleDatabase_Query_disjunction demonstrates using disjunction for OR queries.
// This finds results that match any of several patterns.
func ExampleDatabase_Query_disjunction() {
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	// Query: Find children of alice OR bob
	child := Fresh("child")
	goal := Disj(
		db.Query(parent, NewAtom("alice"), child),
		db.Query(parent, NewAtom("bob"), child),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Results may come in any order due to parallel evaluation
	fmt.Printf("Found %d children\n", len(results))

	// Output:
	// Found 2 children
}
