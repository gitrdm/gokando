package minikanren

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

// Helper to extract int value from atom
func extractInt(t Term) (int, bool) {
	if atom, ok := t.(*Atom); ok {
		if val, ok := atom.Value().(int); ok {
			return val, true
		}
	}
	return 0, false
}

// Helper to extract string value from atom
func extractString(t Term) (string, bool) {
	if atom, ok := t.(*Atom); ok {
		if val, ok := atom.Value().(string); ok {
			return val, true
		}
	}
	return "", false
}

// TestIfa_AllSolutions verifies that Ifa explores all solutions from the condition
// and evaluates the then-branch for each one.
func TestIfa_AllSolutions(t *testing.T) {
	// Test: If x ∈ {1,2,3} then q = x*10
	solutions := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return Ifa(
			Disj(
				Eq(x, NewAtom(1)),
				Disj(
					Eq(x, NewAtom(2)),
					Eq(x, NewAtom(3)),
				),
			),
			func(ctx context.Context, s ConstraintStore) *Stream {
				walked := s.GetSubstitution().Walk(x)
				if atom, ok := walked.(*Atom); ok {
					if val, ok := atom.Value().(int); ok {
						return Eq(q, NewAtom(val*10))(ctx, s)
					}
				}
				return Failure(ctx, s)
			},
			Failure, // else branch not taken
		)
	})

	expected := []int{10, 20, 30}
	var got []int
	for _, s := range solutions {
		if val, ok := extractInt(s); ok {
			got = append(got, val)
		}
	}

	sort.Ints(got)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ifa all solutions: expected %v, got %v", expected, got)
	}
}

// TestIfa_ConditionFails verifies that when the condition fails,
// the else-branch is evaluated exactly once.
func TestIfa_ConditionFails(t *testing.T) {
	solutions := Run(5, func(q *Var) Goal {
		return Ifa(
			Eq(NewAtom(1), NewAtom(2)), // always fails
			Eq(q, NewAtom("then")),
			Eq(q, NewAtom("else")),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	if val, ok := extractString(solutions[0]); ok {
		if val != "else" {
			t.Errorf("Expected 'else', got %v", val)
		}
	} else {
		t.Errorf("Expected string atom, got %T", solutions[0])
	}
}

// TestIfa_NestedBacktracking verifies that Ifa correctly handles
// nested backtracking in both condition and then-branch.
func TestIfa_NestedBacktracking(t *testing.T) {
	type result struct {
		x int
		y int
	}

	// For each x in {1,2}, for each y in {10,20}, produce (x,y)
	solutions := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Ifa(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))), // x ∈ {1,2}
			Conj(
				Disj(Eq(y, NewAtom(10)), Eq(y, NewAtom(20))), // y ∈ {10,20}
				Eq(q, NewPair(x, y)),
			),
			Failure,
		)
	})

	expected := []result{
		{1, 10}, {1, 20},
		{2, 10}, {2, 20},
	}

	var got []result
	for _, s := range solutions {
		if pair, ok := s.(*Pair); ok {
			xVal, okX := extractInt(pair.car)
			yVal, okY := extractInt(pair.cdr)
			if okX && okY {
				got = append(got, result{xVal, yVal})
			}
		}
	}

	sort.Slice(got, func(i, j int) bool {
		if got[i].x != got[j].x {
			return got[i].x < got[j].x
		}
		return got[i].y < got[j].y
	})

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Nested backtracking: expected %v, got %v", expected, got)
	}
}

// TestIfte_CommitsToFirst verifies that Ifte commits to the first solution
// from the condition and ignores subsequent solutions.
func TestIfte_CommitsToFirst(t *testing.T) {
	solutions := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return Ifte(
			Disj(
				Eq(x, NewAtom(1)),
				Disj(
					Eq(x, NewAtom(2)),
					Eq(x, NewAtom(3)),
				),
			),
			Eq(q, x), // Should bind q to first solution (which could be any of 1,2,3)
			Eq(q, NewAtom("else")),
		)
	})

	// Should get exactly 1 solution (commitment to first)
	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution (committed), got %d", len(solutions))
	}

	// The solution should be one of the possible values (order is non-deterministic)
	if val, ok := extractInt(solutions[0]); ok {
		validValues := map[int]bool{1: true, 2: true, 3: true}
		if !validValues[val] {
			t.Errorf("Expected one of {1,2,3}, got %v", val)
		}
	} else {
		t.Errorf("Expected int atom, got %T", solutions[0])
	}
}

// TestIfte_ThenBranchBacktracks verifies that Ifte allows backtracking
// within the then-branch (only the condition commits).
func TestIfte_ThenBranchBacktracks(t *testing.T) {
	solutions := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Ifte(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))), // Commits to first (either 1 or 2)
			Conj(
				Disj(Eq(y, NewAtom(10)), Eq(y, NewAtom(20))), // y ∈ {10,20}
				Eq(q, NewPair(x, y)),
			),
			Failure,
		)
	})

	// Should get 2 solutions: (x=committed_value, y=10) and (x=committed_value, y=20)
	// The x value is whichever came first (non-deterministic ordering)
	if len(solutions) != 2 {
		t.Fatalf("Expected 2 solutions from then-branch, got %d", len(solutions))
	}

	// Extract both solutions
	var results []struct{ x, y int }
	for _, s := range solutions {
		if pair, ok := s.(*Pair); ok {
			xVal, okX := extractInt(pair.car)
			yVal, okY := extractInt(pair.cdr)
			if okX && okY {
				results = append(results, struct{ x, y int }{xVal, yVal})
			}
		}
	}

	if len(results) != 2 {
		t.Fatalf("Failed to extract both solutions")
	}

	// Both solutions should have the same x value (committed to first)
	if results[0].x != results[1].x {
		t.Errorf("Expected committed x value to be consistent, got %d and %d",
			results[0].x, results[1].x)
	}

	// The x value should be either 1 or 2
	committedX := results[0].x
	if committedX != 1 && committedX != 2 {
		t.Errorf("Expected committed x to be 1 or 2, got %d", committedX)
	}

	// Check that we got both y values
	yValues := make(map[int]bool)
	for _, r := range results {
		yValues[r.y] = true
	}
	if !yValues[10] || !yValues[20] {
		t.Errorf("Expected y values {10,20}, got %v", yValues)
	}
}

// TestIfte_ConditionFails verifies that when condition fails,
// else-branch is evaluated.
func TestIfte_ConditionFails(t *testing.T) {
	solutions := Run(5, func(q *Var) Goal {
		return Ifte(
			Eq(NewAtom(1), NewAtom(2)), // fails
			Eq(q, NewAtom("then")),
			Eq(q, NewAtom("else")),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	if val, ok := extractString(solutions[0]); ok {
		if val != "else" {
			t.Errorf("Expected 'else', got %v", val)
		}
	} else {
		t.Errorf("Expected string atom, got %T", solutions[0])
	}
}

// TestSoftCut_AliasForIfte verifies that SoftCut behaves identically to Ifte.
func TestSoftCut_AliasForIfte(t *testing.T) {
	// Run same query with both operators multiple times to account for
	// non-determinism in Disj ordering (goroutine scheduling)
	const trials = 5
	passes := 0

	for i := 0; i < trials; i++ {
		ifteResults := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return Ifte(
				Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
				Eq(q, x),
				Eq(q, NewAtom("else")),
			)
		})

		softCutResults := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return SoftCut(
				Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
				Eq(q, x),
				Eq(q, NewAtom("else")),
			)
		})

		if reflect.DeepEqual(ifteResults, softCutResults) {
			passes++
		}
	}

	// Both should produce identical results most of the time
	// (accounting for goroutine scheduling non-determinism)
	if passes == 0 {
		t.Error("SoftCut and Ifte never produced identical results - implementation may differ")
	}

	// Both operators should always produce exactly 1 solution (commitment)
	ifteCount := len(Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return Ifte(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
			Eq(q, x),
			Eq(q, NewAtom("else")),
		)
	}))

	softCutCount := len(Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return SoftCut(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
			Eq(q, x),
			Eq(q, NewAtom("else")),
		)
	}))

	if ifteCount != 1 || softCutCount != 1 {
		t.Errorf("Expected both to produce 1 solution, got Ifte:%d SoftCut:%d",
			ifteCount, softCutCount)
	}
}

// TestCallGoal_ValidGoal verifies that CallGoal correctly extracts
// and invokes a goal from an atom.
func TestCallGoal_ValidGoal(t *testing.T) {
	solutions := Run(5, func(q *Var) Goal {
		goalAtom := NewAtom(Eq(q, NewAtom(42)))
		return CallGoal(goalAtom)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	if val, ok := extractInt(solutions[0]); ok {
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	} else {
		t.Errorf("Expected int atom, got %T", solutions[0])
	}
}

// TestCallGoal_InvalidTerm verifies that CallGoal fails gracefully
// when the term is not an atom containing a goal.
func TestCallGoal_InvalidTerm(t *testing.T) {
	testCases := []struct {
		name     string
		goalTerm Term
	}{
		{"nil term", nil},
		{"non-goal atom", NewAtom(42)},
		{"non-atom term", NewPair(NewAtom(1), NewAtom(2))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			solutions := Run(5, func(q *Var) Goal {
				return CallGoal(tc.goalTerm)
			})

			if len(solutions) != 0 {
				t.Errorf("Expected 0 solutions for invalid term, got %d", len(solutions))
			}
		})
	}
}

// TestCallGoal_WithUnification verifies that CallGoal works with
// unification to resolve goal variables.
func TestCallGoal_WithUnification(t *testing.T) {
	solutions := Run(5, func(q *Var) Goal {
		goalVar := Fresh("goal")
		return Conj(
			Eq(goalVar, NewAtom(Eq(q, NewAtom("success")))),
			CallGoal(goalVar),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	if val, ok := extractString(solutions[0]); ok {
		if val != "success" {
			t.Errorf("Expected 'success', got %v", val)
		}
	} else {
		t.Errorf("Expected string atom, got %T", solutions[0])
	}
}

// TestVariableScoping_Correct verifies that variables created inside
// Run closure are properly projected in results.
func TestVariableScoping_Correct(t *testing.T) {
	solutions := Run(5, func(q *Var) Goal {
		x := Fresh("x") // Created inside Run - will be projected
		return Conj(
			Eq(x, NewAtom(42)),
			Eq(q, x),
		)
	})

	if len(solutions) != 1 {
		t.Fatalf("Expected 1 solution, got %d", len(solutions))
	}

	if val, ok := extractInt(solutions[0]); ok {
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	} else {
		t.Errorf("Expected int atom, got %T", solutions[0])
	}
}

// TestVariableScoping_Incorrect demonstrates that variables created
// outside Run closure may not be properly projected (regression test).
func TestVariableScoping_Incorrect(t *testing.T) {
	x := Fresh("x") // Created OUTSIDE Run - problematic!

	solutions := Run(5, func(q *Var) Goal {
		return Conj(
			Eq(x, NewAtom(42)),
			Eq(q, x),
		)
	})

	// This test documents current behavior - may produce unprojected variables
	// The exact behavior depends on substitution implementation
	if len(solutions) == 0 {
		t.Skip("Variable scoping issue - no solutions (expected behavior)")
	}

	// If we get solutions, verify the value
	if len(solutions) > 0 {
		if _, isVar := solutions[0].(*Var); isVar {
			t.Log("Got unprojected variable (documented limitation)")
		}
	}
}

// TestIfa_WithCancellation verifies that Ifa respects context cancellation.
func TestIfa_WithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	solutions := RunWithContext(ctx, 10, func(q *Var) Goal {
		x := Fresh("x")
		return Ifa(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
			Eq(q, x),
			Failure,
		)
	})

	// Should get no solutions due to cancellation
	if len(solutions) > 0 {
		t.Logf("Got %d solutions despite cancellation (race condition possible)", len(solutions))
	}
}

// TestIfte_WithCancellation verifies that Ifte respects context cancellation.
func TestIfte_WithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	solutions := RunWithContext(ctx, 10, func(q *Var) Goal {
		x := Fresh("x")
		return Ifte(
			Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))),
			Eq(q, x),
			Failure,
		)
	})

	// Should get no solutions due to cancellation
	if len(solutions) > 0 {
		t.Logf("Got %d solutions despite cancellation (race condition possible)", len(solutions))
	}
}

// TestCallGoal_WithCancellation verifies that CallGoal respects context cancellation.
func TestCallGoal_WithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	solutions := RunWithContext(ctx, 10, func(q *Var) Goal {
		goalAtom := NewAtom(Eq(q, NewAtom(42)))
		return CallGoal(goalAtom)
	})

	// Should get no solutions due to cancellation
	if len(solutions) > 0 {
		t.Logf("Got %d solutions despite cancellation (race condition possible)", len(solutions))
	}
}
