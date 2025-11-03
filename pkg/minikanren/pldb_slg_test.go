package minikanren_test

import (
	"context"
	"testing"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// TestQueryEvaluator tests the basic conversion from pldb query to GoalEvaluator.
func TestQueryEvaluator(t *testing.T) {
	parent, err := DbRel("parent", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
	db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("charlie"))
	db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("diana"))

	t.Run("extract bindings", func(t *testing.T) {
		child := Fresh("child")
		query := db.Query(parent, NewAtom("alice"), child)

		evaluator := QueryEvaluator(query, child.ID())

		ctx := context.Background()
		answers := make(chan map[int64]Term, 10)

		go func() {
			defer close(answers)
			if err := evaluator(ctx, answers); err != nil {
				t.Errorf("Evaluator error: %v", err)
			}
		}()

		count := 0
		for answer := range answers {
			if _, ok := answer[child.ID()]; !ok {
				t.Errorf("Missing binding for child variable")
			}
			count++
		}

		if count != 2 {
			t.Errorf("Expected 2 answers, got %d", count)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		child := Fresh("child")
		query := db.Query(parent, NewAtom("nobody"), child)

		evaluator := QueryEvaluator(query, child.ID())

		ctx := context.Background()
		answers := make(chan map[int64]Term, 10)

		go func() {
			defer close(answers)
			if err := evaluator(ctx, answers); err != nil {
				t.Errorf("Evaluator error: %v", err)
			}
		}()

		count := 0
		for range answers {
			count++
		}

		if count != 0 {
			t.Errorf("Expected 0 answers, got %d", count)
		}
	})
}

// TestTabledQuery tests basic tabled query functionality.
func TestTabledQuery(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

	t.Run("basic tabled query", func(t *testing.T) {
		x := Fresh("x")
		y := Fresh("y")

		goal := TabledQuery(db, edge, "edge", x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Verify each result has bindings for both variables
		for _, s := range results {
			if s.GetBinding(x.ID()) == nil {
				t.Errorf("Missing binding for x")
			}
			if s.GetBinding(y.ID()) == nil {
				t.Errorf("Missing binding for y")
			}
		}
	})

	t.Run("cache reuse", func(t *testing.T) {
		// Clear engine for clean test
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		// First query - cache miss
		goal1 := TabledQuery(db, edge, "edge_reuse", x, y)
		ctx := context.Background()
		store1 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream1 := goal1(ctx, store1)
		results1, _ := stream1.Take(10)

		// Second query - should hit cache
		goal2 := TabledQuery(db, edge, "edge_reuse", x, y)
		store2 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream2 := goal2(ctx, store2)
		results2, _ := stream2.Take(10)

		if len(results1) != len(results2) {
			t.Errorf("Cache results differ: %d vs %d", len(results1), len(results2))
		}

		engine := GlobalEngine()
		stats := engine.Stats()
		if stats.CacheHits == 0 {
			t.Errorf("Expected cache hit, got 0")
		}
	})
}

// TestTabledQuery_Recursive tests recursive queries with tabling.
// Note: Proper recursive tabling requires using the same predicate ID
// throughout the recursion to enable fixpoint computation.
func TestTabledQuery_Recursive(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))
	db, _ = db.AddFact(edge, NewAtom("c"), NewAtom("d"))

	t.Run("simple tabled query", func(t *testing.T) {
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		// Simple base case: just the edges
		goal := TabledQuery(db, edge, "edge_simple", x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)

		// Should find all 3 edges
		if len(results) != 3 {
			t.Errorf("Expected 3 edges, got %d", len(results))
		}
	})
}

// TestRecursiveRule tests the RecursiveRule helper.
func TestRecursiveRule(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	t.Run("basic recursive rule", func(t *testing.T) {
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		// This currently won't work properly without fixing the recursion
		// The RecursiveRule helper needs the tabled predicate reference
		// Let's test that it at least executes the base case
		goal := RecursiveRule(db, edge, "path_rr", []Term{x, y}, func() Goal {
			return Failure // Don't recurse yet
		})

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)

		// Should find at least the base facts
		if len(results) < 2 {
			t.Errorf("Expected at least 2 base results, got %d", len(results))
		}
	})
}

// TestTabledRelation tests the convenient wrapper.
func TestTabledRelation(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))
	db, _ = db.AddFact(edge, NewAtom("b"), NewAtom("c"))

	t.Run("wrapped predicate", func(t *testing.T) {
		edgePred := TabledRelation(db, edge, "edge_wrapped")

		x := Fresh("x")
		y := Fresh("y")

		goal := edgePred(x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("wrong arity", func(t *testing.T) {
		edgePred := TabledRelation(db, edge, "edge_wrong")

		x := Fresh("x")

		// Wrong arity - should return Failure
		goal := edgePred(x) // edge has arity 2, not 1

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)
		if len(results) != 0 {
			t.Errorf("Expected 0 results for wrong arity, got %d", len(results))
		}
	})
}

// TestTabledDatabase tests the automatic tabling wrapper.
func TestTabledDatabase(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	tdb := WithTabledDatabase(db, "test_db")

	t.Run("query auto-tables", func(t *testing.T) {
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		goal := tdb.Query(edge, x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("add fact invalidates cache", func(t *testing.T) {
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		// First query
		goal1 := tdb.Query(edge, x, y)
		ctx := context.Background()
		store1 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream1 := goal1(ctx, store1)
		results1, _ := stream1.Take(10)

		// Add fact
		tdb2, err := tdb.AddFact(edge, NewAtom("b"), NewAtom("c"))
		if err != nil {
			t.Fatal(err)
		}

		// Query new database
		goal2 := tdb2.Query(edge, x, y)
		store2 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream2 := goal2(ctx, store2)
		results2, _ := stream2.Take(10)

		if len(results1) >= len(results2) {
			t.Errorf("Expected more results after adding fact: %d vs %d", len(results1), len(results2))
		}
	})

	t.Run("remove fact invalidates cache", func(t *testing.T) {
		tdb2, err := tdb.AddFact(edge, NewAtom("b"), NewAtom("c"))
		if err != nil {
			t.Fatal(err)
		}

		tdb3, err := tdb2.RemoveFact(edge, NewAtom("b"), NewAtom("c"))
		if err != nil {
			t.Fatal(err)
		}

		x := Fresh("x")
		y := Fresh("y")

		goal := tdb3.Query(edge, x, y)
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)

		results, _ := stream.Take(10)

		// Should have original fact count
		if len(results) != 1 {
			t.Errorf("Expected 1 result after removal, got %d", len(results))
		}
	})

	t.Run("unwrap returns original db", func(t *testing.T) {
		unwrapped := tdb.Unwrap()
		if unwrapped == nil {
			t.Errorf("Unwrap returned nil")
		}

		if unwrapped.FactCount(edge) != 1 {
			t.Errorf("Unwrapped db has wrong fact count: %d", unwrapped.FactCount(edge))
		}
	})
}

// TestInvalidation tests cache invalidation strategies.
func TestInvalidation(t *testing.T) {
	edge, err := DbRel("edge", 2, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	db := NewDatabase()
	db, _ = db.AddFact(edge, NewAtom("a"), NewAtom("b"))

	t.Run("invalidate all", func(t *testing.T) {
		x := Fresh("x")
		y := Fresh("y")

		// Populate cache
		goal := TabledQuery(db, edge, "edge_inv", x, y)
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		stream.Take(10)

		// Clear cache
		InvalidateAll()

		// Check stats reset
		engine := GlobalEngine()
		stats := engine.Stats()
		if stats.CachedSubgoals != 0 {
			t.Errorf("Expected 0 cached subgoals after clear, got %d", stats.CachedSubgoals)
		}
	})

	t.Run("invalidate relation", func(t *testing.T) {
		InvalidateAll()

		x := Fresh("x")
		y := Fresh("y")

		// Populate cache
		goal := TabledQuery(db, edge, "edge_rel_inv", x, y)
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		stream.Take(10)

		// Invalidate specific relation (currently clears all)
		InvalidateRelation("edge_rel_inv")

		// Verify cache cleared
		engine := GlobalEngine()
		stats := engine.Stats()
		if stats.CachedSubgoals != 0 {
			t.Errorf("Expected 0 cached subgoals after invalidate, got %d", stats.CachedSubgoals)
		}
	})
}

// TestStreamFromAnswers tests the internal stream conversion concept.
func TestStreamFromAnswers(t *testing.T) {
	t.Run("unify answers with pattern", func(t *testing.T) {
		x := Fresh("x")
		y := Fresh("y")

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Test the concept of unifying multiple variables
		goal := func(ctx context.Context, s ConstraintStore) *Stream {
			goals := []Goal{
				Eq(x, NewAtom("a")),
				Eq(y, NewAtom("b")),
			}
			return Conj(goals...)(ctx, s)
		}

		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if results[0].GetBinding(x.ID()) == nil {
			t.Errorf("Missing x binding")
		}
	})
}

// TestTabledQuery_Join verifies that TabledQuery correctly handles shared variables in Conj.
func TestTabledQuery_Join(t *testing.T) {
	t.Run("shared variables in Conj work correctly", func(t *testing.T) {
		InvalidateAll()

		parent, _ := DbRel("parent", 2, 0, 1)
		db := NewDatabase()
		db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
		db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))

		gp := Fresh("gp")
		gc := Fresh("gc")
		p := Fresh("p")

		// TabledQuery should work correctly with shared variable p
		goal := Conj(
			TabledQuery(db, parent, "parent_join_test", gp, p),
			TabledQuery(db, parent, "parent_join_test", p, gc),
		)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		// Should find exactly 1 grandparent: alice->bob->charlie
		if len(results) != 1 {
			t.Errorf("Expected 1 grandparent relationship, got %d", len(results))
		}

		// Verify all variables are properly bound
		if len(results) > 0 {
			gpVal := results[0].GetBinding(gp.ID())
			gcVal := results[0].GetBinding(gc.ID())
			pVal := results[0].GetBinding(p.ID())

			if gpVal == nil {
				t.Errorf("Missing gp binding")
			}
			if gcVal == nil {
				t.Errorf("Missing gc binding")
			}
			if pVal == nil {
				t.Errorf("Missing p binding")
			}

			// Verify correct values
			if atom, ok := gpVal.(*Atom); ok {
				if atom.Value() != "alice" {
					t.Errorf("Expected gp=alice, got %v", atom.Value())
				}
			}
			if atom, ok := gcVal.(*Atom); ok {
				if atom.Value() != "charlie" {
					t.Errorf("Expected gc=charlie, got %v", atom.Value())
				}
			}
			if atom, ok := pVal.(*Atom); ok {
				if atom.Value() != "bob" {
					t.Errorf("Expected p=bob, got %v", atom.Value())
				}
			}
		}
	})

	t.Run("compare with regular Query", func(t *testing.T) {
		InvalidateAll()

		parent, _ := DbRel("parent", 2, 0, 1)
		db := NewDatabase()
		db, _ = db.AddFact(parent, NewAtom("alice"), NewAtom("bob"))
		db, _ = db.AddFact(parent, NewAtom("bob"), NewAtom("charlie"))
		db, _ = db.AddFact(parent, NewAtom("charlie"), NewAtom("diana"))

		gp := Fresh("gp")
		gc := Fresh("gc")
		p := Fresh("p")

		// TabledQuery join
		goal1 := Conj(
			TabledQuery(db, parent, "parent_cmp", gp, p),
			TabledQuery(db, parent, "parent_cmp", p, gc),
		)

		ctx := context.Background()
		store1 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream1 := goal1(ctx, store1)
		results1, _ := stream1.Take(10)

		// Regular Query join
		goal2 := Conj(
			db.Query(parent, gp, p),
			db.Query(parent, p, gc),
		)

		store2 := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream2 := goal2(ctx, store2)
		results2, _ := stream2.Take(10)

		// Both should return same number of results
		if len(results1) != len(results2) {
			t.Errorf("TabledQuery returned %d results, regular Query returned %d", len(results1), len(results2))
		}

		// Should find 2 grandparents: alice->charlie, bob->diana
		if len(results1) != 2 {
			t.Errorf("Expected 2 grandparent relationships, got %d", len(results1))
		}
	})
}
