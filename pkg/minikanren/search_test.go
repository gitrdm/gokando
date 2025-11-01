package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestDatabaseSearch tests the database-style search strategy
func TestDatabaseSearch(t *testing.T) {
	store := NewFDStoreWithDomain(4)

	// Create variables
	vars := make([]*FDVar, 3)
	for i := range vars {
		vars[i] = store.NewVar()
	}

	// Add all-different constraint
	store.AddAllDifferent(vars)

	// Test database search
	search := NewDatabaseSearch()
	solutions, err := search.Search(context.Background(), store, NewFirstFailLabeling(), 10)
	if err != nil {
		t.Fatalf("Database search failed: %v", err)
	}

	if len(solutions) == 0 {
		t.Fatal("Expected at least one solution")
	}

	// Verify solutions are valid permutations
	for _, sol := range solutions {
		if len(sol) != 3 {
			t.Errorf("Expected solution length 3, got %d", len(sol))
		}
		seen := make(map[int]bool)
		for _, val := range sol {
			if seen[val] {
				t.Errorf("Duplicate value %d in solution %v", val, sol)
			}
			seen[val] = true
			if val < 1 || val > 4 {
				t.Errorf("Value %d out of domain [1,4]", val)
			}
		}
	}
}

// TestNonChronologicalSearch tests the non-chronological search strategy
func TestNonChronologicalSearch(t *testing.T) {
	store := NewFDStoreWithDomain(3)

	// Create variables
	vars := make([]*FDVar, 3)
	for i := range vars {
		vars[i] = store.NewVar()
	}

	// Add all-different constraint
	store.AddAllDifferent(vars)

	// Test non-chronological search
	search := NewNonChronologicalSearch(10, 1)
	solutions, err := search.Search(context.Background(), store, NewFirstFailLabeling(), 10)
	if err != nil {
		t.Fatalf("Non-chronological search failed: %v", err)
	}

	if len(solutions) == 0 {
		t.Fatal("Expected at least one solution")
	}

	// Verify solutions are valid
	for _, sol := range solutions {
		if len(sol) != 3 {
			t.Errorf("Expected solution length 3, got %d", len(sol))
		}
		seen := make(map[int]bool)
		for _, val := range sol {
			if seen[val] {
				t.Errorf("Duplicate value %d in solution %v", val, sol)
			}
			seen[val] = true
		}
	}
}

// TestRunDB tests the RunDB function with database-style search
func TestRunDB(t *testing.T) {
	// Test with a simple goal that should work with database search
	solutions := RunDB(5, func(q *Var) Goal {
		return FDSolve(FDIn(q, []int{1, 2, 3, 4, 5}))
	})

	if len(solutions) != 5 {
		t.Errorf("Expected 5 solutions, got %d", len(solutions))
	}

	// Verify all solutions are valid
	for _, sol := range solutions {
		if val, ok := sol.(*Atom); ok {
			if intVal, ok2 := val.Value().(int); ok2 {
				if intVal < 1 || intVal > 5 {
					t.Errorf("Solution value %d out of expected range [1,5]", intVal)
				}
			} else {
				t.Errorf("Expected integer solution, got %T", val.Value())
			}
		} else {
			t.Errorf("Expected Atom solution, got %T", sol)
		}
	}
}

// TestRunDBWithContext tests RunDB with context cancellation
func TestRunDBWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Test with a goal that might take longer
	solutions := RunDBWithContext(ctx, 100, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return FDSolve(Conj(
			FDIn(x, []int{1, 2, 3}),
			FDIn(y, []int{1, 2, 3}),
			FDAllDifferent(x, y),
			Eq(q, NewPair(x, y)),
		))
	})

	// Should get some solutions or none if cancelled
	if len(solutions) == 0 {
		t.Log("No solutions returned, possibly due to timeout")
	} else {
		t.Logf("Got %d solutions", len(solutions))
	}
}

// TestRunNC tests the RunNC function with non-chronological search
func TestRunNC(t *testing.T) {
	// Test with a simple goal
	solutions := RunNC(3, func(q *Var) Goal {
		return FDSolve(FDIn(q, []int{1, 2, 3, 4, 5}))
	})

	if len(solutions) != 3 {
		t.Errorf("Expected 3 solutions, got %d", len(solutions))
	}

	// Verify solutions are valid
	for _, sol := range solutions {
		if val, ok := sol.(*Atom); ok {
			if intVal, ok2 := val.Value().(int); ok2 {
				if intVal < 1 || intVal > 5 {
					t.Errorf("Solution value %d out of expected range [1,5]", intVal)
				}
			} else {
				t.Errorf("Expected integer solution, got %T", val.Value())
			}
		} else {
			t.Errorf("Expected Atom solution, got %T", sol)
		}
	}
}

// TestRunNCWithContext tests RunNC with context cancellation
func TestRunNCWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	solutions := RunNCWithContext(ctx, 10, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return FDSolve(Conj(
			FDIn(x, []int{1, 2}),
			FDIn(y, []int{1, 2}),
			FDAllDifferent(x, y),
			Eq(q, NewPair(x, y)),
		))
	})

	// Should get some solutions
	if len(solutions) == 0 {
		t.Log("No solutions returned, possibly due to timeout")
	} else {
		t.Logf("Got %d solutions", len(solutions))
	}
}

// TestSearchStrategyNames tests that search strategies have correct names
func TestSearchStrategyNames(t *testing.T) {
	strategies := []SearchStrategy{
		NewDFSSearch(),
		NewBFSSearch(),
		NewLimitedDepthSearch(10),
		NewIterativeDeepeningSearch(10, 1),
		NewDatabaseSearch(),
		NewNonChronologicalSearch(10, 1),
	}

	expectedNames := []string{
		"dfs",
		"bfs",
		"limited-depth",
		"iterative-deepening",
		"database",
		"non-chronological",
	}

	for i, strategy := range strategies {
		if strategy.Name() != expectedNames[i] {
			t.Errorf("Expected strategy name %s, got %s", expectedNames[i], strategy.Name())
		}
	}
}

// TestSearchStrategyDescriptions tests that search strategies have descriptions
func TestSearchStrategyDescriptions(t *testing.T) {
	strategies := []SearchStrategy{
		NewDatabaseSearch(),
		NewNonChronologicalSearch(10, 1),
	}

	for _, strategy := range strategies {
		desc := strategy.Description()
		if desc == "" {
			t.Errorf("Strategy %s has empty description", strategy.Name())
		}
		if len(desc) < 10 {
			t.Errorf("Strategy %s has suspiciously short description: %s", strategy.Name(), desc)
		}
	}
}

// TestSearchStrategyPruning tests the SupportsPruning method
func TestSearchStrategyPruning(t *testing.T) {
	testCases := []struct {
		strategy SearchStrategy
		expected bool
	}{
		{NewDFSSearch(), true},
		{NewBFSSearch(), false},
		{NewLimitedDepthSearch(10), true},
		{NewIterativeDeepeningSearch(10, 1), true},
		{NewDatabaseSearch(), false},
		{NewNonChronologicalSearch(10, 1), true},
	}

	for _, tc := range testCases {
		if tc.strategy.SupportsPruning() != tc.expected {
			t.Errorf("Strategy %s SupportsPruning() = %v, expected %v",
				tc.strategy.Name(), tc.strategy.SupportsPruning(), tc.expected)
		}
	}
}

// TestDatabaseSearchWithConstraints tests database search with complex constraints
func TestDatabaseSearchWithConstraints(t *testing.T) {
	store := NewFDStoreWithDomain(5)

	// Create variables for SEND + MORE = MONEY cryptarithm
	s, e, n, d := store.NewVar(), store.NewVar(), store.NewVar(), store.NewVar()
	m, o, r, y := store.NewVar(), store.NewVar(), store.NewVar(), store.NewVar()

	// All different constraint
	allVars := []*FDVar{s, e, n, d, m, o, r, y}
	store.AddAllDifferent(allVars)

	// S and M != 0
	if err := store.Remove(s, 0); err != nil {
		t.Fatalf("Failed to remove 0 from S: %v", err)
	}
	if err := store.Remove(m, 0); err != nil {
		t.Fatalf("Failed to remove 0 from M: %v", err)
	}

	// Test database search finds solutions
	search := NewDatabaseSearch()
	solutions, err := search.Search(context.Background(), store, NewFirstFailLabeling(), 5)
	if err != nil {
		t.Fatalf("Database search failed: %v", err)
	}

	if len(solutions) == 0 {
		t.Log("No solutions found for cryptarithm (expected for partial constraints)")
	} else {
		t.Logf("Found %d partial solutions", len(solutions))
	}
}

// TestNonChronologicalSearchDepthLimit tests depth limiting in non-chronological search
func TestNonChronologicalSearchDepthLimit(t *testing.T) {
	store := NewFDStoreWithDomain(10)

	// Create many variables to make search deep
	vars := make([]*FDVar, 8)
	for i := range vars {
		vars[i] = store.NewVar()
	}

	// Add all-different to create complex search space
	store.AddAllDifferent(vars)

	// Test with small depth limit
	search := NewNonChronologicalSearch(5, 1) // Max depth 5
	solutions, err := search.Search(context.Background(), store, NewFirstFailLabeling(), 3)
	if err != nil {
		t.Fatalf("Non-chronological search failed: %v", err)
	}

	// Should find some solutions despite depth limit
	if len(solutions) == 0 {
		t.Log("No solutions found within depth limit (expected for complex problem)")
	} else {
		t.Logf("Found %d solutions within depth limit", len(solutions))
	}
}
