package minikanren

import (
	"fmt"
)

// ExampleNewCallPattern demonstrates creating normalized call patterns.
//
// Call patterns abstract away variable identities, allowing different calls
// with the same structure to share cached answers in tabling.
func ExampleNewCallPattern() {
	// Create a call pattern for edge(a, b)
	args := []Term{NewAtom("a"), NewAtom("b")}
	pattern := NewCallPattern("edge", args)

	fmt.Printf("Predicate: %s\n", pattern.PredicateID())
	fmt.Printf("Structure: %s\n", pattern.ArgStructure())
	fmt.Printf("Full pattern: %s\n", pattern.String())

	// Output:
	// Predicate: edge
	// Structure: atom(a),atom(b)
	// Full pattern: edge(atom(a),atom(b))
}

// ExampleNewCallPattern_variables demonstrates variable abstraction.
//
// Variables are abstracted to canonical positions (X0, X1, ...) based on
// their first occurrence, allowing calls with different variable IDs but
// the same structure to be recognized as equivalent.
func ExampleNewCallPattern_variables() {
	// Two calls with different variable IDs but same structure
	v1 := &Var{id: 42, name: "x"}
	v2 := &Var{id: 73, name: "y"}
	pattern1 := NewCallPattern("path", []Term{v1, v2})

	v3 := &Var{id: 100, name: "p"}
	v4 := &Var{id: 200, name: "q"}
	pattern2 := NewCallPattern("path", []Term{v3, v4})

	fmt.Printf("Pattern 1: %s\n", pattern1.ArgStructure())
	fmt.Printf("Pattern 2: %s\n", pattern2.ArgStructure())
	fmt.Printf("Are equal: %v\n", pattern1.Equal(pattern2))

	// Output:
	// Pattern 1: X0,X1
	// Pattern 2: X0,X1
	// Are equal: true
}

// ExampleNewCallPattern_variableReuse demonstrates repeated variable detection.
//
// When the same variable appears multiple times in the arguments,
// it's represented with the same canonical position.
func ExampleNewCallPattern_variableReuse() {
	v := &Var{id: 42, name: "x"}
	// path(X, X) - same variable twice
	pattern := NewCallPattern("path", []Term{v, v})

	fmt.Printf("Structure: %s\n", pattern.ArgStructure())

	// Output:
	// Structure: X0,X0
}

// ExampleSubgoalTable demonstrates managing tabled subgoals.
//
// The SubgoalTable provides lock-free concurrent access to subgoal entries,
// making it safe for parallel tabling implementations.
func ExampleSubgoalTable() {
	table := NewSubgoalTable()

	// Create a call pattern
	pattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})

	// Get or create a subgoal entry
	entry, created := table.GetOrCreate(pattern)
	fmt.Printf("Created new entry: %v\n", created)
	fmt.Printf("Entry status: %s\n", entry.Status())

	// Subsequent calls return the same entry
	entry2, created2 := table.GetOrCreate(pattern)
	fmt.Printf("Created on second call: %v\n", created2)
	fmt.Printf("Same entry: %v\n", entry == entry2)

	fmt.Printf("Total subgoals: %d\n", table.TotalSubgoals())

	// Output:
	// Created new entry: true
	// Entry status: Active
	// Created on second call: false
	// Same entry: true
	// Total subgoals: 1
}

// ExampleSubgoalEntry demonstrates managing subgoal lifecycle.
//
// SubgoalEntry tracks the evaluation state, dependencies, and statistics
// for a tabled subgoal.
func ExampleSubgoalEntry() {
	pattern := NewCallPattern("fib", []Term{NewAtom(5)})
	entry := NewSubgoalEntry(pattern)

	fmt.Printf("Initial status: %s\n", entry.Status())
	fmt.Printf("Answer count: %d\n", entry.Answers().Count())

	// Add an answer
	bindings := map[int64]Term{1: NewAtom(8)} // fib(5) = 8
	entry.Answers().Insert(bindings)

	fmt.Printf("After insertion: %d answers\n", entry.Answers().Count())

	// Mark as complete
	entry.SetStatus(StatusComplete)
	fmt.Printf("Final status: %s\n", entry.Status())

	// Output:
	// Initial status: Active
	// Answer count: 0
	// After insertion: 1 answers
	// Final status: Complete
}

// ExampleAnswerTrie demonstrates efficient answer storage.
//
// AnswerTrie uses structural sharing to minimize memory overhead
// when storing many similar answers.
func ExampleAnswerTrie() {
	trie := NewAnswerTrie()

	// Insert first answer: {1: a, 2: b}
	answer1 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted := trie.Insert(answer1)
	fmt.Printf("First answer inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert duplicate
	duplicate := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	inserted = trie.Insert(duplicate)
	fmt.Printf("Duplicate inserted: %v\n", inserted)
	fmt.Printf("Count: %d\n", trie.Count())

	// Insert different answer: {1: a, 2: c}
	answer2 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("c"),
	}
	inserted = trie.Insert(answer2)
	fmt.Printf("Different answer inserted: %v\n", inserted)
	fmt.Printf("Final count: %d\n", trie.Count())

	// Output:
	// First answer inserted: true
	// Count: 1
	// Duplicate inserted: false
	// Count: 1
	// Different answer inserted: true
	// Final count: 2
}

// ExampleAnswerTrie_Iterator demonstrates iterating over cached answers.
//
// The iterator provides a consistent snapshot of answers, safe for
// concurrent use with ongoing insertions.
func ExampleAnswerTrie_Iterator() {
	trie := NewAnswerTrie()

	// Insert multiple answers
	for i := 1; i <= 3; i++ {
		bindings := map[int64]Term{
			1: NewAtom(fmt.Sprintf("value%d", i)),
		}
		trie.Insert(bindings)
	}

	// Iterate over all answers
	iter := trie.Iterator()
	count := 0
	for {
		answer, ok := iter.Next()
		if !ok {
			break
		}
		count++
		// Note: iteration order is not guaranteed
		fmt.Printf("Answer has %d bindings\n", len(answer))
	}

	fmt.Printf("Total answers iterated: %d\n", count)

	// Output:
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Answer has 1 bindings
	// Total answers iterated: 3
}

// ExampleSubgoalEntry_dependencies demonstrates dependency tracking.
//
// Dependencies are used for cycle detection and fixpoint computation
// in SLG resolution.
func ExampleSubgoalEntry_dependencies() {
	// Create a dependency chain: path depends on edge
	edgePattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})
	pathPattern := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")})

	edgeEntry := NewSubgoalEntry(edgePattern)
	pathEntry := NewSubgoalEntry(pathPattern)

	// path depends on edge
	pathEntry.AddDependency(edgeEntry)

	deps := pathEntry.Dependencies()
	fmt.Printf("Number of dependencies: %d\n", len(deps))
	fmt.Printf("Depends on: %s\n", deps[0].Pattern().String())

	// Output:
	// Number of dependencies: 1
	// Depends on: edge(atom(a),atom(b))
}

// ExampleSubgoalStatus demonstrates status transitions.
//
// Subgoal status tracks the evaluation lifecycle: Active, Complete,
// Failed, or Invalidated (for incremental tabling).
func ExampleSubgoalStatus() {
	statuses := []SubgoalStatus{
		StatusActive,
		StatusComplete,
		StatusFailed,
		StatusInvalidated,
	}

	for _, status := range statuses {
		fmt.Printf("%s\n", status.String())
	}

	// Output:
	// Active
	// Complete
	// Failed
	// Invalidated
}
