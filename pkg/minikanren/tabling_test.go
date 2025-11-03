package minikanren

import (
	"fmt"
	"sync"
	"testing"
)

// TestCallPattern_Basic tests basic call pattern creation and comparison.
func TestCallPattern_Basic(t *testing.T) {
	// Test with simple atoms
	args1 := []Term{NewAtom("a"), NewAtom("b")}
	pattern1 := NewCallPattern("edge", args1)

	if pattern1.PredicateID() != "edge" {
		t.Errorf("Expected predicateID 'edge', got '%s'", pattern1.PredicateID())
	}

	expected := "atom(a),atom(b)"
	if pattern1.ArgStructure() != expected {
		t.Errorf("Expected argStructure '%s', got '%s'", expected, pattern1.ArgStructure())
	}

	// Test equality
	args2 := []Term{NewAtom("a"), NewAtom("b")}
	pattern2 := NewCallPattern("edge", args2)

	if !pattern1.Equal(pattern2) {
		t.Errorf("Expected patterns to be equal: %s vs %s", pattern1, pattern2)
	}

	if pattern1.Hash() != pattern2.Hash() {
		t.Errorf("Expected equal patterns to have same hash")
	}
}

// TestCallPattern_VariableAbstraction tests variable canonicalization.
func TestCallPattern_VariableAbstraction(t *testing.T) {
	// Create two calls with different variable IDs but same structure
	v1 := &Var{id: 42, name: "x"}
	v2 := &Var{id: 73, name: "y"}
	args1 := []Term{v1, NewAtom("a"), v2}
	pattern1 := NewCallPattern("path", args1)

	v3 := &Var{id: 100, name: "p"}
	v4 := &Var{id: 200, name: "q"}
	args2 := []Term{v3, NewAtom("a"), v4}
	pattern2 := NewCallPattern("path", args2)

	// Should have same structure (X0, atom(a), X1)
	if !pattern1.Equal(pattern2) {
		t.Errorf("Expected patterns with same structure to be equal:\n  %s\n  %s",
			pattern1.ArgStructure(), pattern2.ArgStructure())
	}
}

// TestCallPattern_VariableReuse tests that repeated variables are recognized.
func TestCallPattern_VariableReuse(t *testing.T) {
	v1 := &Var{id: 42, name: "x"}
	args := []Term{v1, NewAtom("a"), v1} // Same variable appears twice
	pattern := NewCallPattern("test", args)

	expected := "X0,atom(a),X0" // X0 repeated
	if pattern.ArgStructure() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, pattern.ArgStructure())
	}
}

// TestCallPattern_DifferentPredicates tests that different predicates produce different patterns.
func TestCallPattern_DifferentPredicates(t *testing.T) {
	args := []Term{NewAtom("a")}
	pattern1 := NewCallPattern("pred1", args)
	pattern2 := NewCallPattern("pred2", args)

	if pattern1.Equal(pattern2) {
		t.Errorf("Expected patterns with different predicates to be unequal")
	}
}

// TestCallPattern_Pairs tests call patterns with pair structures.
func TestCallPattern_Pairs(t *testing.T) {
	pair := NewPair(NewAtom("a"), NewAtom("b"))
	args := []Term{pair}
	pattern := NewCallPattern("test", args)

	expected := "pair(atom(a),atom(b))"
	if pattern.ArgStructure() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, pattern.ArgStructure())
	}
}

// TestCallPattern_NilHandling tests call patterns with nil terms.
func TestCallPattern_NilHandling(t *testing.T) {
	args := []Term{nil, NewAtom("a")}
	pattern := NewCallPattern("test", args)

	expected := "nil,atom(a)"
	if pattern.ArgStructure() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, pattern.ArgStructure())
	}
}

// TestCallPattern_String tests the string representation.
func TestCallPattern_String(t *testing.T) {
	args := []Term{NewAtom("a"), NewAtom("b")}
	pattern := NewCallPattern("edge", args)

	str := pattern.String()
	expected := "edge(atom(a),atom(b))"
	if str != expected {
		t.Errorf("Expected '%s', got '%s'", expected, str)
	}
}

// TestSubgoalEntry_Basic tests basic subgoal entry operations.
func TestSubgoalEntry_Basic(t *testing.T) {
	args := []Term{NewAtom("a")}
	pattern := NewCallPattern("test", args)
	entry := NewSubgoalEntry(pattern)

	if entry.Pattern() != pattern {
		t.Errorf("Expected pattern to match")
	}

	if entry.Status() != StatusActive {
		t.Errorf("Expected initial status Active, got %s", entry.Status())
	}

	if entry.Answers() == nil {
		t.Errorf("Expected answers trie to be initialized")
	}

	if entry.ConsumptionCount() != 0 {
		t.Errorf("Expected initial consumption count 0, got %d", entry.ConsumptionCount())
	}

	if entry.DerivationCount() != 0 {
		t.Errorf("Expected initial derivation count 0, got %d", entry.DerivationCount())
	}
}

// TestSubgoalEntry_StatusTransitions tests status changes.
func TestSubgoalEntry_StatusTransitions(t *testing.T) {
	args := []Term{NewAtom("a")}
	pattern := NewCallPattern("test", args)
	entry := NewSubgoalEntry(pattern)

	entry.SetStatus(StatusComplete)
	if entry.Status() != StatusComplete {
		t.Errorf("Expected status Complete, got %s", entry.Status())
	}

	entry.SetStatus(StatusFailed)
	if entry.Status() != StatusFailed {
		t.Errorf("Expected status Failed, got %s", entry.Status())
	}

	entry.SetStatus(StatusInvalidated)
	if entry.Status() != StatusInvalidated {
		t.Errorf("Expected status Invalidated, got %s", entry.Status())
	}
}

// TestSubgoalEntry_Dependencies tests dependency tracking.
func TestSubgoalEntry_Dependencies(t *testing.T) {
	pattern1 := NewCallPattern("test1", []Term{NewAtom("a")})
	pattern2 := NewCallPattern("test2", []Term{NewAtom("b")})
	pattern3 := NewCallPattern("test3", []Term{NewAtom("c")})

	entry1 := NewSubgoalEntry(pattern1)
	entry2 := NewSubgoalEntry(pattern2)
	entry3 := NewSubgoalEntry(pattern3)

	// Add dependencies
	entry1.AddDependency(entry2)
	entry1.AddDependency(entry3)

	deps := entry1.Dependencies()
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	if deps[0] != entry2 || deps[1] != entry3 {
		t.Errorf("Dependencies not in expected order")
	}
}

// TestSubgoalEntry_ReferenceCount tests reference counting.
func TestSubgoalEntry_ReferenceCount(t *testing.T) {
	pattern := NewCallPattern("test", []Term{NewAtom("a")})
	entry := NewSubgoalEntry(pattern)

	// Initial refcount should be 1
	if entry.refCount.Load() != 1 {
		t.Errorf("Expected initial refcount 1, got %d", entry.refCount.Load())
	}

	// Retain
	entry.Retain()
	if entry.refCount.Load() != 2 {
		t.Errorf("Expected refcount 2 after Retain, got %d", entry.refCount.Load())
	}

	// Release
	if entry.Release() {
		t.Errorf("Expected Release to return false (refcount > 0)")
	}
	if entry.refCount.Load() != 1 {
		t.Errorf("Expected refcount 1 after Release, got %d", entry.refCount.Load())
	}

	// Final release
	if !entry.Release() {
		t.Errorf("Expected Release to return true (refcount = 0)")
	}
	if entry.refCount.Load() != 0 {
		t.Errorf("Expected refcount 0 after final Release, got %d", entry.refCount.Load())
	}
}

// TestSubgoalTable_GetOrCreate tests basic table operations.
func TestSubgoalTable_GetOrCreate(t *testing.T) {
	table := NewSubgoalTable()

	args := []Term{NewAtom("a")}
	pattern := NewCallPattern("test", args)

	// First call should create
	entry1, created := table.GetOrCreate(pattern)
	if !created {
		t.Errorf("Expected first GetOrCreate to create entry")
	}
	if entry1 == nil {
		t.Fatalf("Expected non-nil entry")
	}

	// Second call should retrieve
	entry2, created := table.GetOrCreate(pattern)
	if created {
		t.Errorf("Expected second GetOrCreate to retrieve existing entry")
	}
	if entry1 != entry2 {
		t.Errorf("Expected same entry on second call")
	}

	// Total subgoals should be 1
	if table.TotalSubgoals() != 1 {
		t.Errorf("Expected total subgoals 1, got %d", table.TotalSubgoals())
	}
}

// TestSubgoalTable_Get tests retrieval of existing entries.
func TestSubgoalTable_Get(t *testing.T) {
	table := NewSubgoalTable()

	pattern := NewCallPattern("test", []Term{NewAtom("a")})

	// Get non-existent should return nil
	if entry := table.Get(pattern); entry != nil {
		t.Errorf("Expected nil for non-existent entry")
	}

	// Create entry
	created, _ := table.GetOrCreate(pattern)

	// Get should now return it
	retrieved := table.Get(pattern)
	if retrieved != created {
		t.Errorf("Expected Get to return created entry")
	}
}

// TestSubgoalTable_DifferentPatterns tests multiple patterns in table.
func TestSubgoalTable_DifferentPatterns(t *testing.T) {
	table := NewSubgoalTable()

	pattern1 := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})
	pattern2 := NewCallPattern("edge", []Term{NewAtom("b"), NewAtom("c")})
	pattern3 := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("c")})

	entry1, _ := table.GetOrCreate(pattern1)
	entry2, _ := table.GetOrCreate(pattern2)
	entry3, _ := table.GetOrCreate(pattern3)

	// All should be different
	if entry1 == entry2 || entry1 == entry3 || entry2 == entry3 {
		t.Errorf("Expected different entries for different patterns")
	}

	if table.TotalSubgoals() != 3 {
		t.Errorf("Expected 3 total subgoals, got %d", table.TotalSubgoals())
	}
}

// TestSubgoalTable_AllEntries tests retrieving all entries.
func TestSubgoalTable_AllEntries(t *testing.T) {
	table := NewSubgoalTable()

	// Create multiple entries
	patterns := []*CallPattern{
		NewCallPattern("test1", []Term{NewAtom("a")}),
		NewCallPattern("test2", []Term{NewAtom("b")}),
		NewCallPattern("test3", []Term{NewAtom("c")}),
	}

	for _, pattern := range patterns {
		table.GetOrCreate(pattern)
	}

	allEntries := table.AllEntries()
	if len(allEntries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(allEntries))
	}
}

// TestSubgoalTable_Clear tests clearing the table.
func TestSubgoalTable_Clear(t *testing.T) {
	table := NewSubgoalTable()

	// Add entries
	for i := 0; i < 5; i++ {
		pattern := NewCallPattern("test", []Term{NewAtom(fmt.Sprintf("a%d", i))})
		table.GetOrCreate(pattern)
	}

	if table.TotalSubgoals() != 5 {
		t.Errorf("Expected 5 subgoals before clear, got %d", table.TotalSubgoals())
	}

	// Clear
	table.Clear()

	if table.TotalSubgoals() != 0 {
		t.Errorf("Expected 0 subgoals after clear, got %d", table.TotalSubgoals())
	}

	// AllEntries should be empty
	if len(table.AllEntries()) != 0 {
		t.Errorf("Expected no entries after clear")
	}
}

// TestSubgoalTable_Concurrent tests concurrent access to table.
func TestSubgoalTable_Concurrent(t *testing.T) {
	table := NewSubgoalTable()
	pattern := NewCallPattern("test", []Term{NewAtom("a")})

	// Multiple goroutines trying to create same pattern
	const numGoroutines = 10
	var wg sync.WaitGroup
	entries := make([]*SubgoalEntry, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			entry, _ := table.GetOrCreate(pattern)
			entries[idx] = entry
		}(i)
	}

	wg.Wait()

	// All goroutines should get the same entry
	first := entries[0]
	for i := 1; i < numGoroutines; i++ {
		if entries[i] != first {
			t.Errorf("Goroutine %d got different entry", i)
		}
	}

	// Only one subgoal should exist
	if table.TotalSubgoals() != 1 {
		t.Errorf("Expected 1 subgoal after concurrent creation, got %d", table.TotalSubgoals())
	}
}

// TestAnswerTrie_Insert tests inserting answers into the trie.
func TestAnswerTrie_Insert(t *testing.T) {
	trie := NewAnswerTrie()

	// Insert first answer: {1: atom(a), 2: atom(b)}
	bindings1 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	if !trie.Insert(bindings1) {
		t.Errorf("Expected first insert to return true (new answer)")
	}

	if trie.Count() != 1 {
		t.Errorf("Expected count 1 after first insert, got %d", trie.Count())
	}

	// Insert duplicate answer
	if trie.Insert(bindings1) {
		t.Errorf("Expected duplicate insert to return false")
	}

	if trie.Count() != 1 {
		t.Errorf("Expected count still 1 after duplicate insert, got %d", trie.Count())
	}

	// Insert different answer: {1: atom(a), 2: atom(c)}
	bindings2 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("c"),
	}
	if !trie.Insert(bindings2) {
		t.Errorf("Expected second insert to return true (new answer)")
	}

	if trie.Count() != 2 {
		t.Errorf("Expected count 2 after second insert, got %d", trie.Count())
	}
}

// TestAnswerTrie_InsertEmpty tests inserting empty answer (all vars unbound).
func TestAnswerTrie_InsertEmpty(t *testing.T) {
	trie := NewAnswerTrie()

	// Empty bindings
	bindings := map[int64]Term{}
	if !trie.Insert(bindings) {
		t.Errorf("Expected empty answer insert to return true")
	}

	if trie.Count() != 1 {
		t.Errorf("Expected count 1, got %d", trie.Count())
	}

	// Duplicate empty
	if trie.Insert(bindings) {
		t.Errorf("Expected duplicate empty answer to return false")
	}
}

// TestAnswerTrie_InsertSameStructure tests answers with same structure.
func TestAnswerTrie_InsertSameStructure(t *testing.T) {
	trie := NewAnswerTrie()

	// Create variable
	v2 := &Var{id: 2}

	// Answer 1: {1: atom(a), 2: atom(b)}
	bindings1 := map[int64]Term{
		1: NewAtom("a"),
		2: NewAtom("b"),
	}
	trie.Insert(bindings1)

	// Answer 2: {1: atom(a), 2: var(2)} - different because var is unbound
	bindings2 := map[int64]Term{
		1: NewAtom("a"),
		2: v2,
	}
	if !trie.Insert(bindings2) {
		t.Errorf("Expected different answer to be inserted")
	}

	if trie.Count() != 2 {
		t.Errorf("Expected count 2, got %d", trie.Count())
	}
}

// TestAnswerTrie_Iterator tests iterating over answers.
func TestAnswerTrie_Iterator(t *testing.T) {
	trie := NewAnswerTrie()

	// Insert multiple answers
	expected := []map[int64]Term{
		{1: NewAtom("a"), 2: NewAtom("b")},
		{1: NewAtom("a"), 2: NewAtom("c")},
		{1: NewAtom("x"), 2: NewAtom("y")},
	}

	for _, bindings := range expected {
		trie.Insert(bindings)
	}

	// Iterate and collect
	iter := trie.Iterator()
	found := make([]map[int64]Term, 0)

	for {
		answer, ok := iter.Next()
		if !ok {
			break
		}
		found = append(found, answer)
	}

	if len(found) != len(expected) {
		t.Errorf("Expected %d answers, got %d", len(expected), len(found))
	}

	// Check all expected answers are found (order may differ)
	for _, exp := range expected {
		foundMatch := false
		for _, f := range found {
			if answersEqual(exp, f) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			t.Errorf("Expected answer not found: %v", exp)
		}
	}
}

// TestAnswerTrie_IteratorEmpty tests iterating over empty trie.
func TestAnswerTrie_IteratorEmpty(t *testing.T) {
	trie := NewAnswerTrie()
	iter := trie.Iterator()

	answer, ok := iter.Next()
	if ok {
		t.Errorf("Expected Next() to return false for empty trie, got %v", answer)
	}
}

// TestAnswerTrie_IteratorSingleAnswer tests iterator with one answer.
func TestAnswerTrie_IteratorSingleAnswer(t *testing.T) {
	trie := NewAnswerTrie()

	bindings := map[int64]Term{1: NewAtom("a")}
	trie.Insert(bindings)

	iter := trie.Iterator()

	// First call should return answer
	answer, ok := iter.Next()
	if !ok {
		t.Errorf("Expected Next() to return true for first answer")
	}
	if !answersEqual(answer, bindings) {
		t.Errorf("Expected answer %v, got %v", bindings, answer)
	}

	// Second call should return false
	answer, ok = iter.Next()
	if ok {
		t.Errorf("Expected Next() to return false after exhausting answers")
	}
}

// TestAnswerTrie_ConcurrentInsert tests concurrent insertions.
func TestAnswerTrie_ConcurrentInsert(t *testing.T) {
	trie := NewAnswerTrie()

	const numGoroutines = 10
	const answersPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < answersPerGoroutine; j++ {
				bindings := map[int64]Term{
					int64(goroutineID): NewAtom(fmt.Sprintf("g%d", goroutineID)),
					int64(j):           NewAtom(fmt.Sprintf("a%d", j)),
				}
				trie.Insert(bindings)
			}
		}(i)
	}

	wg.Wait()

	// Should have numGoroutines * answersPerGoroutine unique answers
	expected := int64(numGoroutines * answersPerGoroutine)
	if trie.Count() != expected {
		t.Errorf("Expected count %d, got %d", expected, trie.Count())
	}
}

// TestAnswerTrie_ConcurrentReadWrite tests concurrent reads and writes.
func TestAnswerTrie_ConcurrentReadWrite(t *testing.T) {
	trie := NewAnswerTrie()

	// Pre-populate with some answers
	for i := 0; i < 10; i++ {
		bindings := map[int64]Term{1: NewAtom(fmt.Sprintf("a%d", i))}
		trie.Insert(bindings)
	}

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				bindings := map[int64]Term{
					1: NewAtom(fmt.Sprintf("w%d", writerID)),
					2: NewAtom(fmt.Sprintf("a%d", j)),
				}
				trie.Insert(bindings)
			}
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				iter := trie.Iterator()
				count := 0
				for {
					_, ok := iter.Next()
					if !ok {
						break
					}
					count++
				}
				// Just verify we can iterate without crashing
			}
		}()
	}

	wg.Wait()

	// Verify final count makes sense (at least initial + writers)
	if trie.Count() < 10 {
		t.Errorf("Expected count >= 10, got %d", trie.Count())
	}
}

// TestHashTerm tests term hashing consistency.
func TestHashTerm(t *testing.T) {
	// Same atoms should hash the same
	atom1 := NewAtom("test")
	atom2 := NewAtom("test")
	if hashTerm(atom1) != hashTerm(atom2) {
		t.Errorf("Expected same atoms to have same hash")
	}

	// Different atoms should (likely) hash differently
	atom3 := NewAtom("different")
	if hashTerm(atom1) == hashTerm(atom3) {
		t.Logf("Warning: Different atoms have same hash (collision possible)")
	}

	// Variables with same ID should hash the same
	v1 := &Var{id: 42}
	v2 := &Var{id: 42}
	if hashTerm(v1) != hashTerm(v2) {
		t.Errorf("Expected same variable IDs to have same hash")
	}

	// Pairs should be consistent
	pair1 := NewPair(NewAtom("a"), NewAtom("b"))
	pair2 := NewPair(NewAtom("a"), NewAtom("b"))
	if hashTerm(pair1) != hashTerm(pair2) {
		t.Errorf("Expected same pairs to have same hash")
	}

	// Nil should hash consistently
	if hashTerm(nil) != 0 {
		t.Errorf("Expected nil to hash to 0")
	}
}

// Helper function to check if two answer maps are equal.
func answersEqual(a, b map[int64]Term) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !v.Equal(bv) {
			return false
		}
	}
	return true
}

// Benchmark answer trie insertion.
func BenchmarkAnswerTrie_Insert(b *testing.B) {
	trie := NewAnswerTrie()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindings := map[int64]Term{
			1: NewAtom(fmt.Sprintf("a%d", i)),
			2: NewAtom("b"),
		}
		trie.Insert(bindings)
	}
}

// Benchmark answer trie iteration.
func BenchmarkAnswerTrie_Iterator(b *testing.B) {
	trie := NewAnswerTrie()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		bindings := map[int64]Term{
			1: NewAtom(fmt.Sprintf("a%d", i)),
		}
		trie.Insert(bindings)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := trie.Iterator()
		for {
			_, ok := iter.Next()
			if !ok {
				break
			}
		}
	}
}

// Benchmark subgoal table concurrent access.
func BenchmarkSubgoalTable_GetOrCreate(b *testing.B) {
	table := NewSubgoalTable()
	patterns := make([]*CallPattern, 100)
	for i := 0; i < 100; i++ {
		patterns[i] = NewCallPattern("test", []Term{NewAtom(fmt.Sprintf("a%d", i))})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pattern := patterns[i%len(patterns)]
			table.GetOrCreate(pattern)
			i++
		}
	})
}
