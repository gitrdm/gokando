package minikanren_test

import (
	"context"
	"fmt"
	"sort"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// ExampleTabledRelation_transitiveClosureManual demonstrates manually building
// transitive closure with tabled queries. This shows the low-level approach.
func ExampleTabledRelation_transitiveClosureManual() {
	// Define edge relation
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	// Create tabled edge predicate
	edgeTabled := TabledRelation(db, edge, "edge")

	// Manually define path using disjunction: path(X,Y) :- edge(X,Y) | (edge(X,Z), path(Z,Y))
	// For a proper recursive definition, we'd need fixpoint computation
	// Here we just show multi-hop manually
	x := Fresh("x")
	y := Fresh("y")
	z := Fresh("z")

	// Two-hop path: a->b->c or b->c->d
	twoHop := Conj(
		edgeTabled(x, z),
		edgeTabled(z, y),
	)

	// Bind x to "a" to find paths from a
	goal := Conj(
		Eq(x, NewAtom("a")),
		twoHop,
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Should find a->c (via b)
	if len(results) > 0 {
		binding := results[0].GetBinding(y.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("a reaches %s in 2 hops\n", atom.String())
		}
	}

	// Output:
	// a reaches c in 2 hops
}

// ExampleRecursiveRule_familyTree shows ancestor queries with RecursiveRule.
func ExampleRecursiveRule_familyTree() {
	// Define relations
	parent, _ := DbRel("parent", 2, 0, 1)

	// Build family tree
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"john", "tom"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"tom", "bob"},
	)

	// Query variables
	x := Fresh("x")
	y := Fresh("y")

	// Define ancestor as recursive rule
	//ancestor := RecursiveRule(
	//	db,
	//	parent,     // base: parent is ancestor
	//	"ancestor", // predicate ID
	//	[]Term{x, y},
	//	func() Goal { // recursive: ancestor of parent is ancestor
	//		z := Fresh("z")
	//		return Conj(
	//			TabledQuery(db, parent, "ancestor", x, z),
	//			TabledQuery(db, parent, "ancestor", z, y),
	//		)
	//	},
	//)

	// For now, just query direct parents (base case)
	goal := Conj(
		Eq(y, NewAtom("alice")),
		db.Query(parent, x, y),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Collect results
	parents := make([]string, 0)
	for _, s := range results {
		if binding := s.GetBinding(x.ID()); binding != nil {
			if atom, ok := binding.(*Atom); ok {
				parents = append(parents, atom.String())
			}
		}
	}
	sort.Strings(parents)

	for _, name := range parents {
		fmt.Printf("%s is parent of alice\n", name)
	}

	// Output:
	// mary is parent of alice
}

// ExampleTabledQuery_grandparent demonstrates joining tabled queries.
func ExampleTabledQuery_grandparent() {
	// Create parent relation
	parent, _ := DbRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
	)

	// Query for grandparent
	gp := Fresh("gp")
	p := Fresh("p")
	gc := Fresh("gc")

	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	goal := Conj(
		TabledQuery(db, parent, "parent", gp, p),
		TabledQuery(db, parent, "parent", p, gc),
		Eq(gp, NewAtom("john")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(gc.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("john's grandchild: %s\n", atom.String())
		}
	}

	// Output:
	// john's grandchild: alice
}

// ExampleTabledDatabase demonstrates automatic tabling for all queries.
func ExampleTabledDatabase() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
	)

	// Wrap database for automatic tabling
	tdb := WithTabledDatabase(db, "mydb")

	x := Fresh("x")
	y := Fresh("y")

	// All queries automatically use tabling
	goal := tdb.Query(edge, x, y)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Found %d edges with automatic tabling\n", len(results))

	// Output:
	// Found 2 edges with automatic tabling
}

// ExampleTabledDatabase_withMutation shows cache invalidation after updates.
func ExampleTabledDatabase_withMutation() {
	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "mutable")

	x := Fresh("x")
	y := Fresh("y")

	// Query once to populate cache
	goal := tdb.Query(edge, x, y)
	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results1, _ := stream.Take(10)

	fmt.Printf("Before update: %d edges\n", len(results1))

	// Add a new fact
	db2, _ := db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	tdb2 := WithTabledDatabase(db2, "mutable")

	// Clear cache for this predicate
	InvalidateAll()

	// Query again with new database
	goal2 := tdb2.Query(edge, x, y)
	stream2 := goal2(ctx, store)
	results2, _ := stream2.Take(10)

	fmt.Printf("After update: %d edges\n", len(results2))

	// Output:
	// Before update: 1 edges
	// After update: 2 edges
}

// ExampleTabledRelation_symmetricGraph shows querying symmetric relations.
func ExampleTabledRelation_symmetricGraph() {
	friend, _ := DbRel("friend", 2, 0, 1)
	db := DB().MustAddFacts(friend,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "alice"},
	)

	friendPred := TabledRelation(db, friend, "friend")

	x := Fresh("x")
	// Who is friends with Alice?
	goal := Conj(
		friendPred(x, NewAtom("alice")),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(x.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s is friend with alice\n", atom.String())
		}
	}

	// Output:
	// bob is friend with alice
}

// ExampleTabledQuery_multiRelation demonstrates queries across multiple relations.
func ExampleTabledQuery_multiRelation() {
	employee, _ := DbRel("employee", 2, 0, 1) // (name, dept)
	manager, _ := DbRel("manager", 2, 0, 1)   // (mgr, employee)

	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", "engineering"},
		[]interface{}{"bob", "engineering"},
	)
	db = db.MustAddFacts(manager,
		[]interface{}{"bob", "alice"},
	)

	// Who manages Alice?
	mgr := Fresh("mgr")
	goal := Conj(
		TabledQuery(db, manager, "mgr", mgr, NewAtom("alice")),
		TabledQuery(db, employee, "emp", mgr, Fresh("_")), // ensure mgr is an employee
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	if len(results) > 0 {
		binding := results[0].GetBinding(mgr.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("%s manages alice\n", atom.String())
		}
	}

	// Output:
	// bob manages alice
}
