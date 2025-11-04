package minikanren

import (
	"context"
	"fmt"
)

// ExampleUnifiedStoreAdapter_basicQuery demonstrates using the UnifiedStoreAdapter
// to query a pldb database with the hybrid solver's UnifiedStore.
//
// This is the simplest pattern: create an adapter, query the database, and
// retrieve results. The adapter enables pldb queries (which expect ConstraintStore)
// to work with UnifiedStore (used by the hybrid solver).
func ExampleUnifiedStoreAdapter_basicQuery() {
	// Create a database of people with names and ages
	person, _ := DbRel("person", 2, 0) // name is indexed
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))
	db, _ = db.AddFact(person, NewAtom("bob"), NewAtom(25))
	db, _ = db.AddFact(person, NewAtom("carol"), NewAtom(35))

	// Create UnifiedStore and adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for all people
	name := Fresh("name")
	age := Fresh("age")

	goal := db.Query(person, name, age)
	stream := goal(context.Background(), adapter)

	// Retrieve results
	results, _ := stream.Take(10)

	// Print number of results (order may vary due to map iteration)
	fmt.Printf("Found %d people\n", len(results))

	// Output:
	// Found 3 people
}

// ExampleUnifiedStoreAdapter_fdConstrainedQuery demonstrates combining pldb
// queries with FD domain constraints using manual filtering.
//
// This pattern shows how to:
//  1. Create an FD domain for a variable
//  2. Query the database normally
//  3. Manually filter results by FD domain membership
//
// This explicit integration gives you full control over when and how
// FD constraints affect query results.
func ExampleUnifiedStoreAdapter_fdConstrainedQuery() {
	// Create a database of employees with ages (compact via HLAPI)
	employee, _ := DbRel("employee", 2, 0) // name is indexed
	db := DB().MustAddFacts(employee,
		[]interface{}{"alice", 28},
		[]interface{}{"bob", 32},
		[]interface{}{"carol", 45},
		[]interface{}{"dave", 29},
	)

	// Create FD model with age restricted to [25, 35]
	model := NewModel()
	// ageVar := model.NewVariableWithName(
	//     NewBitSetDomainFromValues(100, []int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}),
	//     "age",
	// )
	ageVar := model.IntVarValues([]int{25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35}, "age")

	// Create store with FD domain and adapter
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Define variables
	name := Fresh("name")
	age := Fresh("age")

	// Use HLAPI FDFilteredQuery to combine the DB query and FD-domain filtering
	// FDFilteredQuery(db, rel, fdVar, filterVar, queryTerms...)
	goal := FDFilteredQuery(db, employee, ageVar, age, name, age)

	// Execute query
	stream := goal(context.Background(), adapter)
	results, _ := stream.Take(10)

	// Print count (order may vary)
	fmt.Printf("Found %d employees aged 25-35\n", len(results))

	// Output:
	// Found 3 employees aged 25-35
}

// ExampleUnifiedStoreAdapter_hybridPropagation demonstrates full integration
// between pldb queries and hybrid solver propagation.
//
// This pattern shows:
//  1. Querying the database to get relational bindings
//  2. Mapping logical variables to FD variables
//  3. Running hybrid solver propagation
//  4. Observing how relational bindings prune FD domains
//
// This is useful when you want database facts to constrain FD variables,
// which then participate in constraint propagation with other FD constraints.
func ExampleUnifiedStoreAdapter_hybridPropagation() {
	// Create database of people with ages
	person, _ := DbRel("person", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(person, NewAtom("alice"), NewAtom(30))

	// Create FD model with age variable (domain 0-100)
	model := NewModel()
	ageValues := make([]int, 101)
	for i := range ageValues {
		ageValues[i] = i
	}
	// ageVar := model.NewVariableWithName(NewBitSetDomainFromValues(101, ageValues), "age")
	ageVar := model.IntVarValues(ageValues, "age")

	// Set up hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(relPlugin, fdPlugin)

	// Create store with FD domain
	store := NewUnifiedStore()
	store, _ = store.SetDomain(ageVar.ID(), ageVar.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	// Query for alice's age
	age := Fresh("age")
	goal := db.Query(person, NewAtom("alice"), age)
	stream := goal(context.Background(), adapter)

	results, _ := stream.Take(1)
	if len(results) > 0 {
		resultAdapter := results[0].(*UnifiedStoreAdapter)

		// Link logical variable to FD variable
		resultStore := resultAdapter.UnifiedStore()
		ageBinding := resultAdapter.GetBinding(age.ID())
		if ageAtom, ok := ageBinding.(*Atom); ok {
			if ageInt, ok := ageAtom.value.(int); ok {
				// Bind FD variable to the same value
				resultStore, _ = resultStore.AddBinding(int64(ageVar.ID()), NewAtom(ageInt))
				resultAdapter.SetUnifiedStore(resultStore)

				// Run propagation
				propagated, err := solver.Propagate(resultAdapter.UnifiedStore())
				if err == nil {
					// FD domain should now be singleton {30}
					ageDomain := propagated.GetDomain(ageVar.ID())
					if ageDomain.IsSingleton() {
						fmt.Printf("FD domain pruned to: {%d}\n", ageDomain.SingletonValue())
					}
				}
			}
		}
	}

	// Output:
	// FD domain pruned to: {30}
}

// ExampleUnifiedStoreAdapter_parallelSearch demonstrates that adapter cloning
// preserves search branch independence for parallel miniKanren execution.
//
// When searching in parallel (e.g., via Conj or Disj), each search branch
// gets its own copy of the adapter and UnifiedStore. This ensures:
//  1. No shared mutable state between branches
//  2. Each branch can independently bind variables
//  3. No race conditions in parallel search
//
// The adapter's Clone() method creates deep copies suitable for concurrent use.
func ExampleUnifiedStoreAdapter_parallelSearch() {
	// Create database
	color, _ := DbRel("color", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(color, NewAtom("apple"), NewAtom("red"))
	db, _ = db.AddFact(color, NewAtom("banana"), NewAtom("yellow"))

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Simulate parallel search: clone adapter for each branch
	branch1 := adapter.Clone().(*UnifiedStoreAdapter)
	branch2 := adapter.Clone().(*UnifiedStoreAdapter)

	// Each branch queries independently
	item := Fresh("item")

	goal1 := db.Query(color, item, NewAtom("red"))
	stream1 := goal1(context.Background(), branch1)
	results1, _ := stream1.Take(1)

	goal2 := db.Query(color, item, NewAtom("yellow"))
	stream2 := goal2(context.Background(), branch2)
	results2, _ := stream2.Take(1)

	// Print results from each independent branch
	if len(results1) > 0 {
		itemBinding := results1[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 1: %s is red\n", atom.value)
		}
	}

	if len(results2) > 0 {
		itemBinding := results2[0].GetBinding(item.ID())
		if atom, ok := itemBinding.(*Atom); ok {
			fmt.Printf("Branch 2: %s is yellow\n", atom.value)
		}
	}

	// Output:
	// Branch 1: apple is red
	// Branch 2: banana is yellow
}

// ExampleUnifiedStoreAdapter_performance demonstrates that the adapter
// maintains efficient indexed access for large databases.
//
// pldb uses indexing for O(1) lookups on indexed fields. The adapter
// preserves this performance characteristic - queries remain fast even
// with thousands of facts.
func ExampleUnifiedStoreAdapter_performance() {
	// Create large database with 1000 people
	person, _ := DbRel("person", 3, 0, 1, 2) // all fields indexed
	db := NewDatabase()

	for i := 0; i < 1000; i++ {
		name := NewAtom(fmt.Sprintf("person%d", i))
		age := NewAtom(20 + (i % 50))
		score := NewAtom(50 + (i % 50))
		db, _ = db.AddFact(person, name, age, score)
	}

	// Create adapter
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	// Query for specific age (indexed lookup is O(1))
	name := Fresh("name")
	score := Fresh("score")

	goal := db.Query(person, name, NewAtom(30), score)
	stream := goal(context.Background(), adapter)

	// Fast retrieval even from large database
	results, _ := stream.Take(100)

	fmt.Printf("Found %d people with age 30 (from 1000 total)\n", len(results))

	// Output:
	// Found 20 people with age 30 (from 1000 total)
}
