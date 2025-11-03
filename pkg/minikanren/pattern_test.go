package minikanren

import (
	"context"
	"testing"
)

// TestMatche_EmptyListPattern tests exhaustive matching with empty list pattern.
func TestMatche_EmptyListPattern(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matche(Nil,
			NewClause(Nil, Eq(q, NewAtom("matched"))),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("matched")) {
		t.Errorf("Expected 'matched', got %v", result[0])
	}
}

// TestMatche_SingletonListPattern tests matching a single-element list.
func TestMatche_SingletonListPattern(t *testing.T) {
	list := List(NewAtom(42))

	result := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return Matche(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(x, Nil), Eq(q, x)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(42)) {
		t.Errorf("Expected 42, got %v", result[0])
	}
}

// TestMatche_MultipleMatches tests that all matching clauses produce results.
func TestMatche_MultipleMatches(t *testing.T) {
	// A pair matches both "pair" pattern and "any" pattern
	pair := NewPair(NewAtom(1), NewAtom(2))

	result := Run(10, func(q *Var) Goal {
		return Matche(pair,
			NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
			NewClause(Fresh("_"), Eq(q, NewAtom("any"))),
		)
	})

	if len(result) != 2 {
		t.Fatalf("Expected 2 results (exhaustive matching), got %d", len(result))
	}

	// Results can be in any order due to Disj
	hasAny := false
	hasPair := false
	for _, r := range result {
		if r.Equal(NewAtom("any")) {
			hasAny = true
		}
		if r.Equal(NewAtom("pair")) {
			hasPair = true
		}
	}

	if !hasAny || !hasPair {
		t.Errorf("Expected both 'any' and 'pair', got %v", result)
	}
}

// TestMatche_ListClassification tests classifying lists by length.
func TestMatche_ListClassification(t *testing.T) {
	tests := []struct {
		name     string
		list     Term
		expected []string
	}{
		{
			name:     "empty list",
			list:     Nil,
			expected: []string{"empty", "any"},
		},
		{
			name:     "singleton list",
			list:     List(NewAtom(1)),
			expected: []string{"singleton", "any"},
		},
		{
			name:     "pair list",
			list:     List(NewAtom(1), NewAtom(2)),
			expected: []string{"multiple", "any"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Run(10, func(q *Var) Goal {
				return Matche(tt.list,
					NewClause(Nil, Eq(q, NewAtom("empty"))),
					NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
					NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
					NewClause(Fresh("_"), Eq(q, NewAtom("any"))),
				)
			})

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d results, got %d", len(tt.expected), len(result))
			}

			// Convert results to strings for comparison
			gotStrings := make([]string, len(result))
			for i, r := range result {
				if atom, ok := r.(*Atom); ok {
					if s, ok := atom.value.(string); ok {
						gotStrings[i] = s
					}
				}
			}

			// Check all expected values are present
			for _, exp := range tt.expected {
				found := false
				for _, got := range gotStrings {
					if got == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected result '%s' not found in %v", exp, gotStrings)
				}
			}
		})
	}
}

// TestMatche_WithGoalSequence tests executing multiple goals in a clause.
func TestMatche_WithGoalSequence(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Matche(NewPair(NewAtom(1), NewAtom(2)),
			NewClause(
				NewPair(x, y),
				Eq(x, NewAtom(1)),
				Eq(y, NewAtom(2)),
				Eq(q, NewAtom("success")),
			),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("success")) {
		t.Errorf("Expected 'success', got %v", result[0])
	}
}

// TestMatche_NoMatchingClause tests behavior when no clause matches.
func TestMatche_NoMatchingClause(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matche(NewAtom(42),
			NewClause(Nil, Eq(q, NewAtom("list"))),
			NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
		)
	})

	if len(result) != 0 {
		t.Fatalf("Expected 0 results when no clause matches, got %d", len(result))
	}
}

// TestMatcha_FirstMatchOnly tests committed choice behavior.
func TestMatcha_FirstMatchOnly(t *testing.T) {
	// Both clauses would match, but only first is tried
	pair := NewPair(NewAtom(1), NewAtom(2))

	result := Run(10, func(q *Var) Goal {
		return Matcha(pair,
			NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("first"))),
			NewClause(Fresh("_"), Eq(q, NewAtom("second"))),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (committed choice), got %d", len(result))
	}

	if !result[0].Equal(NewAtom("first")) {
		t.Errorf("Expected 'first' (committed to first match), got %v", result[0])
	}
}

// TestMatcha_SkipToNextOnMismatch tests that Matcha tries clauses in order.
func TestMatcha_SkipToNextOnMismatch(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matcha(NewAtom(42),
			NewClause(Nil, Eq(q, NewAtom("wrong1"))),
			NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("wrong2"))),
			NewClause(NewAtom(42), Eq(q, NewAtom("correct"))),
			NewClause(Fresh("_"), Eq(q, NewAtom("wrong3"))),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("correct")) {
		t.Errorf("Expected 'correct', got %v", result[0])
	}
}

// TestMatcha_CommitEvenIfGoalsFail tests that Matcha commits even if goals fail.
func TestMatcha_CommitEvenIfGoalsFail(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matcha(NewAtom(1),
			// First clause matches but goal fails
			NewClause(NewAtom(1), Eq(NewAtom(1), NewAtom(2))),
			// Second clause would match but is never tried
			NewClause(Fresh("_"), Eq(q, NewAtom("fallback"))),
		)
	})

	// Should get no results because first clause matched but its goal failed
	if len(result) != 0 {
		t.Errorf("Expected 0 results (committed but failed), got %d", len(result))
	}
}

// TestMatchu_ExactlyOneMatch tests unique matching success case.
func TestMatchu_ExactlyOneMatch(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matchu(NewAtom(2),
			NewClause(NewAtom(1), Eq(q, NewAtom("one"))),
			NewClause(NewAtom(2), Eq(q, NewAtom("two"))),
			NewClause(NewAtom(3), Eq(q, NewAtom("three"))),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("two")) {
		t.Errorf("Expected 'two', got %v", result[0])
	}
}

// TestMatchu_NoMatches tests unique matching with zero matches fails.
func TestMatchu_NoMatches(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matchu(NewAtom(99),
			NewClause(NewAtom(1), Eq(q, NewAtom("one"))),
			NewClause(NewAtom(2), Eq(q, NewAtom("two"))),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (no unique match), got %d", len(result))
	}
}

// TestMatchu_MultipleMatches tests unique matching with multiple matches fails.
func TestMatchu_MultipleMatches(t *testing.T) {
	pair := NewPair(NewAtom(1), NewAtom(2))

	result := Run(10, func(q *Var) Goal {
		return Matchu(pair,
			NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
			NewClause(Fresh("_"), Eq(q, NewAtom("any"))),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (multiple matches, not unique), got %d", len(result))
	}
}

// TestMatchu_WithFreshVariablesInPattern tests unique matching with variable binding.
func TestMatchu_WithFreshVariablesInPattern(t *testing.T) {
	list := List(NewAtom(10), NewAtom(20))

	result := Run(10, func(q *Var) Goal {
		head := Fresh("head")
		tail := Fresh("tail")
		return Matchu(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(head, tail), Eq(q, head)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(10)) {
		t.Errorf("Expected 10 (head of list), got %v", result[0])
	}
}

// TestNewClause_EmptyGoals tests clause with no goals (pattern check only).
func TestNewClause_EmptyGoals(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Conj(
			Matche(NewAtom(42),
				NewClause(NewAtom(42)), // No goals - just check pattern
			),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("success")) {
		t.Errorf("Expected 'success', got %v", result[0])
	}
}

// TestMatcheList_ValidListPatterns tests list-specific matching.
func TestMatcheList_ValidListPatterns(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return MatcheList(List(NewAtom(1), NewAtom(2)),
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("one"))),
			NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Nil)), Eq(q, NewAtom("two"))),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("two")) {
		t.Errorf("Expected 'two', got %v", result[0])
	}
}

// TestMatcheList_VariablePattern tests that variables are valid list patterns.
func TestMatcheList_VariablePattern(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		anyList := Fresh("anyList")
		return MatcheList(List(NewAtom(1)),
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(anyList, Eq(q, NewAtom("nonempty"))),
		)
	})

	// Should match both clauses since variable matches anything
	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// The non-empty clause matches since list is not nil
	if !result[0].Equal(NewAtom("nonempty")) {
		t.Errorf("Expected 'nonempty', got %v", result[0])
	}
}

// TestPatternMatching_Integration tests pattern matching with hybrid solver.
func TestPatternMatching_Integration(t *testing.T) {
	// Test that pattern matching works with UnifiedStore (hybrid solver)
	store := NewUnifiedStore()
	adapter := NewUnifiedStoreAdapter(store)

	q := Fresh("q")
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	goal := Matche(list,
		NewClause(Nil, Eq(q, NewAtom("empty"))),
		NewClause(NewPair(NewAtom(1), Fresh("rest")), Eq(q, NewAtom("matched-one"))),
		NewClause(Fresh("_"), Eq(q, NewAtom("matched-any"))),
	)

	ctx := context.Background()
	stream := goal(ctx, adapter)
	results, _ := stream.Take(10)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results with hybrid solver, got %d", len(results))
	}

	// Should match both the specific pattern and the wildcard
	hasMatchedOne := false
	hasMatchedAny := false
	for _, r := range results {
		binding := r.GetBinding(q.ID())
		if binding == nil {
			continue
		}
		if binding.Equal(NewAtom("matched-one")) {
			hasMatchedOne = true
		}
		if binding.Equal(NewAtom("matched-any")) {
			hasMatchedAny = true
		}
	}

	if !hasMatchedOne {
		t.Error("Expected 'matched-one' in results")
	}
	if !hasMatchedAny {
		t.Error("Expected 'matched-any' in results")
	}
}

// TestPatternMatching_EmptyClauses tests behavior with no clauses.
func TestPatternMatching_EmptyClauses(t *testing.T) {
	result := Run(10, func(q *Var) Goal {
		return Matche(NewAtom(42))
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results with no clauses, got %d", len(result))
	}
}

// TestPatternMatching_ComplexPatterns tests nested pattern matching.
func TestPatternMatching_ComplexPatterns(t *testing.T) {
	// Nested list: ((1 2) (3 4))
	nested := List(
		List(NewAtom(1), NewAtom(2)),
		List(NewAtom(3), NewAtom(4)),
	)

	result := Run(10, func(q *Var) Goal {
		second := Fresh("second")
		a := Fresh("a")
		b := Fresh("b")

		return Matche(nested,
			NewClause(
				NewPair(
					NewPair(a, NewPair(b, Nil)),
					NewPair(second, Nil),
				),
				Eq(q, List(a, b)),
			),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	expected := List(NewAtom(1), NewAtom(2))
	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestPatternMatching_WithConstraints tests pattern matching with FD constraints.
func TestPatternMatching_WithConstraints(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))

	// Constrain x to be in {5, 6, 7}
	domain := NewBitSetDomainFromValues(10, []int{5, 6, 7})

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), domain)
	adapter := NewUnifiedStoreAdapter(store)

	q := Fresh("q")
	val := Fresh("val")

	// Pattern match with variable that should unify with FD variable domain
	goal := Conj(
		Eq(val, NewAtom(5)),
		Matche(val,
			NewClause(NewAtom(5), Eq(q, NewAtom("small"))),
			NewClause(NewAtom(10), Eq(q, NewAtom("large"))),
		),
	)

	ctx := context.Background()
	stream := goal(ctx, adapter)
	results, _ := stream.Take(10)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	binding := results[0].GetBinding(q.ID())
	if binding == nil || !binding.Equal(NewAtom("small")) {
		t.Errorf("Expected 'small', got %v", binding)
	}
}
