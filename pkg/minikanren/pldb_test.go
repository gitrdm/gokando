package minikanren

import (
	"context"
	"testing"
)

// TestDbRel_Creation tests relation creation with various configurations.
func TestDbRel_Creation(t *testing.T) {
	tests := []struct {
		name        string
		relName     string
		arity       int
		indexedCols []int
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid binary relation with both columns indexed",
			relName:     "parent",
			arity:       2,
			indexedCols: []int{0, 1},
			wantErr:     false,
		},
		{
			name:        "valid unary relation with index",
			relName:     "person",
			arity:       1,
			indexedCols: []int{0},
			wantErr:     false,
		},
		{
			name:        "valid relation with no indexes",
			relName:     "triple",
			arity:       3,
			indexedCols: []int{},
			wantErr:     false,
		},
		{
			name:        "valid relation with selective indexing",
			relName:     "edge",
			arity:       2,
			indexedCols: []int{0}, // only source indexed
			wantErr:     false,
		},
		{
			name:        "invalid: zero arity",
			relName:     "bad",
			arity:       0,
			indexedCols: []int{},
			wantErr:     true,
			errContains: "arity must be positive",
		},
		{
			name:        "invalid: negative arity",
			relName:     "bad",
			arity:       -1,
			indexedCols: []int{},
			wantErr:     true,
			errContains: "arity must be positive",
		},
		{
			name:        "invalid: empty name",
			relName:     "",
			arity:       2,
			indexedCols: []int{0},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name:        "invalid: index out of range (high)",
			relName:     "bad",
			arity:       2,
			indexedCols: []int{0, 2},
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "invalid: index out of range (negative)",
			relName:     "bad",
			arity:       2,
			indexedCols: []int{-1},
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "valid: duplicate index specification (idempotent)",
			relName:     "dup",
			arity:       3,
			indexedCols: []int{0, 1, 0}, // duplicate 0 is ok
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := DbRel(tt.relName, tt.arity, tt.indexedCols...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel.Name() != tt.relName {
				t.Errorf("Name() = %q, want %q", rel.Name(), tt.relName)
			}
			if rel.Arity() != tt.arity {
				t.Errorf("Arity() = %d, want %d", rel.Arity(), tt.arity)
			}
			// Verify indexes
			indexSet := make(map[int]bool)
			for _, col := range tt.indexedCols {
				indexSet[col] = true
			}
			for col := 0; col < tt.arity; col++ {
				expected := indexSet[col]
				if rel.IsIndexed(col) != expected {
					t.Errorf("IsIndexed(%d) = %v, want %v", col, rel.IsIndexed(col), expected)
				}
			}
		})
	}
}

// TestDatabase_AddFact tests adding facts to relations.
func TestDatabase_AddFact(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	t.Run("add single fact", func(t *testing.T) {
		db := NewDatabase()
		db2, err := db.AddFact(parent, alice, bob)
		if err != nil {
			t.Fatalf("AddFact failed: %v", err)
		}
		if db2 == nil {
			t.Fatal("AddFact returned nil database")
		}
		if db2.FactCount(parent) != 1 {
			t.Errorf("FactCount = %d, want 1", db2.FactCount(parent))
		}
		// Original database unchanged
		if db.FactCount(parent) != 0 {
			t.Errorf("original database modified")
		}
	})

	t.Run("add multiple facts", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.AddFact(parent, bob, charlie)
		if db.FactCount(parent) != 2 {
			t.Errorf("FactCount = %d, want 2", db.FactCount(parent))
		}
	})

	t.Run("deduplication: same fact twice", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.AddFact(parent, alice, bob) // duplicate
		if db.FactCount(parent) != 1 {
			t.Errorf("FactCount = %d, want 1 (deduplication)", db.FactCount(parent))
		}
	})

	t.Run("error: nil relation", func(t *testing.T) {
		db := NewDatabase()
		_, err := db.AddFact(nil, alice, bob)
		if err == nil || !contains(err.Error(), "relation cannot be nil") {
			t.Errorf("expected nil relation error, got %v", err)
		}
	})

	t.Run("error: arity mismatch (too few)", func(t *testing.T) {
		db := NewDatabase()
		_, err := db.AddFact(parent, alice)
		if err == nil || !contains(err.Error(), "expects 2 terms, got 1") {
			t.Errorf("expected arity error, got %v", err)
		}
	})

	t.Run("error: arity mismatch (too many)", func(t *testing.T) {
		db := NewDatabase()
		_, err := db.AddFact(parent, alice, bob, charlie)
		if err == nil || !contains(err.Error(), "expects 2 terms, got 3") {
			t.Errorf("expected arity error, got %v", err)
		}
	})

	t.Run("error: non-ground term (variable)", func(t *testing.T) {
		db := NewDatabase()
		x := Fresh("x")
		_, err := db.AddFact(parent, alice, x)
		if err == nil || !contains(err.Error(), "not ground") {
			t.Errorf("expected non-ground error, got %v", err)
		}
	})

	t.Run("copy-on-write semantics", func(t *testing.T) {
		db1 := NewDatabase()
		db2, _ := db1.AddFact(parent, alice, bob)
		db3, _ := db2.AddFact(parent, bob, charlie)

		// Each version has its own state
		if db1.FactCount(parent) != 0 {
			t.Errorf("db1.FactCount = %d, want 0", db1.FactCount(parent))
		}
		if db2.FactCount(parent) != 1 {
			t.Errorf("db2.FactCount = %d, want 1", db2.FactCount(parent))
		}
		if db3.FactCount(parent) != 2 {
			t.Errorf("db3.FactCount = %d, want 2", db3.FactCount(parent))
		}
	})
}

// TestDatabase_RemoveFact tests fact removal with tombstone semantics.
func TestDatabase_RemoveFact(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	t.Run("remove existing fact", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.AddFact(parent, bob, charlie)

		db2, err := db.RemoveFact(parent, alice, bob)
		if err != nil {
			t.Fatalf("RemoveFact failed: %v", err)
		}
		if db2.FactCount(parent) != 1 {
			t.Errorf("FactCount after removal = %d, want 1", db2.FactCount(parent))
		}
		// Original unchanged
		if db.FactCount(parent) != 2 {
			t.Errorf("original database modified")
		}
	})

	t.Run("remove non-existent fact (idempotent)", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)

		db2, err := db.RemoveFact(parent, bob, charlie)
		if err != nil {
			t.Fatalf("RemoveFact failed: %v", err)
		}
		if db2.FactCount(parent) != 1 {
			t.Errorf("FactCount = %d, want 1 (unchanged)", db2.FactCount(parent))
		}
	})

	t.Run("remove all facts", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.RemoveFact(parent, alice, bob)
		if db.FactCount(parent) != 0 {
			t.Errorf("FactCount = %d, want 0", db.FactCount(parent))
		}
	})

	t.Run("tombstone semantics: re-add after remove", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.RemoveFact(parent, alice, bob)
		db, _ = db.AddFact(parent, alice, bob)
		if db.FactCount(parent) != 1 {
			t.Errorf("FactCount after re-add = %d, want 1", db.FactCount(parent))
		}
	})

	t.Run("error: nil relation", func(t *testing.T) {
		db := NewDatabase()
		_, err := db.RemoveFact(nil, alice, bob)
		if err == nil || !contains(err.Error(), "relation cannot be nil") {
			t.Errorf("expected nil relation error, got %v", err)
		}
	})

	t.Run("error: arity mismatch", func(t *testing.T) {
		db := NewDatabase()
		_, err := db.RemoveFact(parent, alice)
		if err == nil || !contains(err.Error(), "expects 2 terms") {
			t.Errorf("expected arity error, got %v", err)
		}
	})
}

// TestDatabase_AllFacts tests retrieving all facts from a relation.
func TestDatabase_AllFacts(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	t.Run("empty relation", func(t *testing.T) {
		db := NewDatabase()
		facts := db.AllFacts(parent)
		if facts != nil {
			t.Errorf("AllFacts on empty relation = %v, want nil", facts)
		}
	})

	t.Run("single fact", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		facts := db.AllFacts(parent)
		if len(facts) != 1 {
			t.Fatalf("len(AllFacts) = %d, want 1", len(facts))
		}
		if len(facts[0]) != 2 {
			t.Fatalf("fact arity = %d, want 2", len(facts[0]))
		}
	})

	t.Run("multiple facts", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.AddFact(parent, bob, charlie)
		facts := db.AllFacts(parent)
		if len(facts) != 2 {
			t.Errorf("len(AllFacts) = %d, want 2", len(facts))
		}
	})

	t.Run("excludes tombstoned facts", func(t *testing.T) {
		db := NewDatabase()
		db, _ = db.AddFact(parent, alice, bob)
		db, _ = db.AddFact(parent, bob, charlie)
		db, _ = db.RemoveFact(parent, alice, bob)
		facts := db.AllFacts(parent)
		if len(facts) != 1 {
			t.Errorf("len(AllFacts) after removal = %d, want 1", len(facts))
		}
	})

	t.Run("nil relation", func(t *testing.T) {
		db := NewDatabase()
		facts := db.AllFacts(nil)
		if facts != nil {
			t.Errorf("AllFacts(nil) = %v, want nil", facts)
		}
	})
}

// TestDatabase_Query tests the Query→Goal functionality.
func TestDatabase_Query(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")
	dave := NewAtom("dave")

	// Build test database
	db := NewDatabase()
	db, _ = db.AddFact(parent, alice, bob)
	db, _ = db.AddFact(parent, alice, charlie)
	db, _ = db.AddFact(parent, bob, dave)

	ctx := context.Background()

	t.Run("query with all fresh variables", func(t *testing.T) {
		x := Fresh("x")
		y := Fresh("y")
		goal := db.Query(parent, x, y)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 3 {
			t.Errorf("got %d results, want 3", len(results))
		}
	})

	t.Run("query with ground first argument", func(t *testing.T) {
		child := Fresh("child")
		goal := db.Query(parent, alice, child)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 2 {
			t.Fatalf("got %d results, want 2 (bob and charlie)", len(results))
		}

		// Verify bindings
		for _, r := range results {
			val := r.GetBinding(child.ID())
			if val.Equal(bob) || val.Equal(charlie) {
				// expected
			} else {
				t.Errorf("unexpected binding: %v", val)
			}
		}
	})

	t.Run("query with ground second argument", func(t *testing.T) {
		p := Fresh("parent")
		goal := db.Query(parent, p, bob)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Fatalf("got %d results, want 1 (alice)", len(results))
		}

		val := results[0].GetBinding(p.ID())
		if !val.Equal(alice) {
			t.Errorf("parent of bob = %v, want alice", val)
		}
	})

	t.Run("query with both ground (exists)", func(t *testing.T) {
		goal := db.Query(parent, alice, bob)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Errorf("got %d results, want 1 (fact exists)", len(results))
		}
	})

	t.Run("query with both ground (not exists)", func(t *testing.T) {
		goal := db.Query(parent, bob, alice)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 0 {
			t.Errorf("got %d results, want 0 (fact doesn't exist)", len(results))
		}
	})

	t.Run("repeated variable: same value", func(t *testing.T) {
		// Create self-loop
		edge, _ := DbRel("edge", 2, 0, 1)
		db2 := NewDatabase()
		db2, _ = db2.AddFact(edge, alice, alice) // self-loop
		db2, _ = db2.AddFact(edge, alice, bob)
		db2, _ = db2.AddFact(edge, bob, bob) // self-loop

		x := Fresh("x")
		goal := db2.Query(edge, x, x) // find self-loops

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 2 {
			t.Errorf("got %d self-loops, want 2", len(results))
		}
	})

	t.Run("repeated variable: filters non-matching", func(t *testing.T) {
		edge, _ := DbRel("edge", 2, 0, 1)
		db2 := NewDatabase()
		db2, _ = db2.AddFact(edge, alice, bob)
		db2, _ = db2.AddFact(edge, bob, charlie)

		x := Fresh("x")
		goal := db2.Query(edge, x, x)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 0 {
			t.Errorf("got %d results, want 0 (no self-loops)", len(results))
		}
	})

	t.Run("empty relation", func(t *testing.T) {
		person, _ := DbRel("person", 1, 0)
		db2 := NewDatabase()

		x := Fresh("x")
		goal := db2.Query(person, x)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 0 {
			t.Errorf("got %d results, want 0 (empty relation)", len(results))
		}
	})

	t.Run("nil relation returns Failure", func(t *testing.T) {
		x := Fresh("x")
		goal := db.Query(nil, x)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 0 {
			t.Errorf("got %d results, want 0 (nil relation)", len(results))
		}
	})

	t.Run("arity mismatch returns Failure", func(t *testing.T) {
		x := Fresh("x")
		goal := db.Query(parent, x) // wrong arity

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 0 {
			t.Errorf("got %d results, want 0 (arity mismatch)", len(results))
		}
	})
}

// TestDatabase_Query_Indexing tests that indexes are used correctly.
func TestDatabase_Query_Indexing(t *testing.T) {
	// Create relation with only first column indexed
	edge, _ := DbRel("edge", 2, 0)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	db := NewDatabase()
	db, _ = db.AddFact(edge, alice, bob)
	db, _ = db.AddFact(edge, alice, charlie)
	db, _ = db.AddFact(edge, bob, charlie)

	ctx := context.Background()

	t.Run("indexed column query (first)", func(t *testing.T) {
		y := Fresh("y")
		goal := db.Query(edge, alice, y)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		// Should use index on first column
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("non-indexed column query (second)", func(t *testing.T) {
		x := Fresh("x")
		goal := db.Query(edge, x, charlie)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		// Should fall back to full scan
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})
}

// TestDatabase_Query_WithTombstones tests that queries skip tombstoned facts.
func TestDatabase_Query_WithTombstones(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	db := NewDatabase()
	db, _ = db.AddFact(parent, alice, bob)
	db, _ = db.AddFact(parent, alice, charlie)
	db, _ = db.RemoveFact(parent, alice, bob) // tombstone this

	ctx := context.Background()

	t.Run("tombstoned facts excluded from results", func(t *testing.T) {
		child := Fresh("child")
		goal := db.Query(parent, alice, child)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Fatalf("got %d results, want 1 (only charlie)", len(results))
		}

		val := results[0].GetBinding(child.ID())
		if !val.Equal(charlie) {
			t.Errorf("child = %v, want charlie", val)
		}
	})
}

// TestDatabase_Integration tests pldb with constraint stores and goals.
func TestDatabase_Integration(t *testing.T) {
	parent, _ := DbRel("parent", 2, 0, 1)
	grandparent, _ := DbRel("grandparent", 2, 0, 1)

	alice := NewAtom("alice")
	bob := NewAtom("bob")
	charlie := NewAtom("charlie")

	db := NewDatabase()
	db, _ = db.AddFact(parent, alice, bob)
	db, _ = db.AddFact(parent, bob, charlie)

	ctx := context.Background()

	t.Run("compose with Conj", func(t *testing.T) {
		// Find x such that alice is parent of x AND x is parent of charlie
		x := Fresh("x")
		goal := Conj(
			db.Query(parent, alice, x),
			db.Query(parent, x, charlie),
		)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}

		val := results[0].GetBinding(x.ID())
		if !val.Equal(bob) {
			t.Errorf("x = %v, want bob", val)
		}
	})

	t.Run("transitive closure (grandparent)", func(t *testing.T) {
		// grandparent(X, Z) :- parent(X, Y), parent(Y, Z)
		gp := Fresh("gp")
		gc := Fresh("gc")
		y := Fresh("y")

		goal := Conj(
			db.Query(parent, gp, y),
			db.Query(parent, y, gc),
		)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Fatalf("got %d grandparent relations, want 1", len(results))
		}

		gpVal := results[0].GetBinding(gp.ID())
		gcVal := results[0].GetBinding(gc.ID())
		if !gpVal.Equal(alice) || !gcVal.Equal(charlie) {
			t.Errorf("grandparent = (%v, %v), want (alice, charlie)", gpVal, gcVal)
		}
	})

	t.Run("compose with Disj", func(t *testing.T) {
		// Find children of alice OR bob
		child := Fresh("child")
		goal := Disj(
			db.Query(parent, alice, child),
			db.Query(parent, bob, child),
		)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		// alice has 1 child (bob), bob has 1 child (charlie)
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
	})

	t.Run("materialize derived relation", func(t *testing.T) {
		// Compute grandparent relation and add to db
		gp := Fresh("gp")
		gc := Fresh("gc")
		y := Fresh("y")

		goal := Conj(
			db.Query(parent, gp, y),
			db.Query(parent, y, gc),
		)

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		db2 := NewDatabase()
		for _, r := range results {
			gpVal := r.GetBinding(gp.ID())
			gcVal := r.GetBinding(gc.ID())
			db2, _ = db2.AddFact(grandparent, gpVal, gcVal)
		}

		// Now query the materialized grandparent relation
		g := Fresh("grandchild")
		gpGoal := db2.Query(grandparent, alice, g)
		stream2 := gpGoal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
		results2, _ := stream2.Take(10)

		if len(results2) != 1 {
			t.Fatalf("got %d grandchildren, want 1", len(results2))
		}

		val := results2[0].GetBinding(g.ID())
		if !val.Equal(charlie) {
			t.Errorf("grandchild = %v, want charlie", val)
		}
	})
}

// TestDatabase_LargeScale tests performance with many facts.
func TestDatabase_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-scale test in short mode")
	}

	edge, _ := DbRel("edge", 2, 0, 1)
	db := NewDatabase()

	// Add 1000 facts
	n := 1000
	for i := 0; i < n; i++ {
		src := NewAtom(i)
		dst := NewAtom(i + 1)
		db, _ = db.AddFact(edge, src, dst)
	}

	if db.FactCount(edge) != n {
		t.Errorf("FactCount = %d, want %d", db.FactCount(edge), n)
	}

	t.Run("indexed lookup", func(t *testing.T) {
		target := NewAtom(500)
		y := Fresh("y")
		goal := db.Query(edge, target, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(10)

		if len(results) != 1 {
			t.Errorf("indexed lookup returned %d results, want 1", len(results))
		}
	})

	t.Run("full scan", func(t *testing.T) {
		x := Fresh("x")
		y := Fresh("y")
		goal := db.Query(edge, x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(n + 100)

		if len(results) != n {
			t.Errorf("full scan returned %d results, want %d", len(results), n)
		}
	})

	t.Run("tombstone performance", func(t *testing.T) {
		// Remove half the facts
		for i := 0; i < n/2; i++ {
			src := NewAtom(i * 2)
			dst := NewAtom(i*2 + 1)
			db, _ = db.RemoveFact(edge, src, dst)
		}

		if db.FactCount(edge) != n/2 {
			t.Errorf("FactCount after removal = %d, want %d", db.FactCount(edge), n/2)
		}

		// Query should still work efficiently
		x := Fresh("x")
		y := Fresh("y")
		goal := db.Query(edge, x, y)

		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		results, _ := stream.Take(n)

		if len(results) != n/2 {
			t.Errorf("query after tombstones returned %d results, want %d", len(results), n/2)
		}
	})
}
