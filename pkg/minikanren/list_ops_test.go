package minikanren

import (
	"context"
	"testing"
)

// TestRembero_RemoveFirst tests removing the first occurrence of an element.
func TestRembero_RemoveFirst(t *testing.T) {
	// Remove 2 from [1,2,3,2,4]
	input := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(2), NewAtom(4))
	expected := List(NewAtom(1), NewAtom(3), NewAtom(2), NewAtom(4))

	result := Run(1, func(q *Var) Goal {
		return Rembero(NewAtom(2), input, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestRembero_GenerateInputs tests generating possible input lists.
func TestRembero_GenerateInputs(t *testing.T) {
	// Given output [1,3] and element 2, what input lists could produce this?
	output := List(NewAtom(1), NewAtom(3))

	result := Run(5, func(q *Var) Goal {
		return Rembero(NewAtom(2), q, output)
	})

	// Should get: [2,1,3], [1,2,3], [1,3,2]
	if len(result) < 3 {
		t.Fatalf("Expected at least 3 results, got %d", len(result))
	}

	// Verify each result when element 2 is removed gives output
	for i, inputList := range result {
		verify := Run(1, func(check *Var) Goal {
			return Rembero(NewAtom(2), inputList, check)
		})
		if len(verify) == 0 || !verify[0].Equal(output) {
			t.Errorf("Result %d: %v doesn't produce output %v when 2 is removed", i, inputList, output)
		}
	}
}

// TestRembero_DetermineElement tests determining what element was removed.
func TestRembero_DetermineElement(t *testing.T) {
	input := List(NewAtom(1), NewAtom(2), NewAtom(3))
	output := List(NewAtom(1), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Rembero(q, input, output)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(2)) {
		t.Errorf("Expected element 2, got %v", result[0])
	}
}

// TestRembero_EmptyList tests removing from an empty list.
func TestRembero_EmptyList(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Rembero(NewAtom(1), Nil, q)
	})

	// Cannot remove from empty list
	if len(result) != 0 {
		t.Errorf("Expected no results, got %d", len(result))
	}
}

// TestReverso_Forward tests reversing a list.
func TestReverso_Forward(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))
	expected := List(NewAtom(4), NewAtom(3), NewAtom(2), NewAtom(1))

	result := Run(1, func(q *Var) Goal {
		return Reverso(list, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestReverso_Backward tests computing original from reversed.
// Note: Due to the relational nature of Reverso using Appendo, backward mode
// can be slow for longer lists. This test uses a small list.
func TestReverso_Backward(t *testing.T) {
	reversed := List(NewAtom(3), NewAtom(2), NewAtom(1))
	expected := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Reverso(q, reversed)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestReverso_Verification tests verifying two lists are reverses.
func TestReverso_Verification(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	reversed := List(NewAtom(3), NewAtom(2), NewAtom(1))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Reverso(list, reversed),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected reversal verification to succeed")
	}
}

// TestReverso_EmptyList tests reversing an empty list.
func TestReverso_EmptyList(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Reverso(Nil, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(Nil) {
		t.Errorf("Expected empty list, got %v", result[0])
	}
}

// TestPermuteo_Generate tests generating permutations.
func TestPermuteo_Generate(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(10, func(q *Var) Goal {
		return Permuteo(list, q)
	})

	// 3! = 6 permutations
	if len(result) != 6 {
		t.Errorf("Expected 6 permutations, got %d", len(result))
	}

	// Check each permutation has the same elements
	for i, perm := range result {
		// Verify it's a valid permutation by checking length and membership
		lenResult := Run(1, func(check *Var) Goal {
			return LengthoInt(perm, check)
		})
		if len(lenResult) == 0 || !lenResult[0].Equal(NewAtom(3)) {
			t.Errorf("Permutation %d has wrong length: %v", i, perm)
		}
	}
}

// TestPermuteo_Verify tests verifying a permutation.
func TestPermuteo_Verify(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	perm := List(NewAtom(3), NewAtom(1), NewAtom(2))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Permuteo(list, perm),
			Eq(q, NewAtom("valid")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected permutation to be verified")
	}
}

// TestPermuteo_InvalidPermutation tests rejecting invalid permutations.
func TestPermuteo_InvalidPermutation(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	notPerm := List(NewAtom(1), NewAtom(1), NewAtom(3)) // Duplicate 1, missing 2

	result := Run(1, func(q *Var) Goal {
		return Permuteo(list, notPerm)
	})

	if len(result) != 0 {
		t.Errorf("Expected invalid permutation to be rejected, got %d results", len(result))
	}
}

// TestSubseto_VerifySubset tests verifying subset relationship.
func TestSubseto_VerifySubset(t *testing.T) {
	subset := List(NewAtom(1), NewAtom(3))
	superset := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Subseto(subset, superset),
			Eq(q, NewAtom("valid")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected subset verification to succeed")
	}
}

// TestSubseto_GenerateSubsets tests generating all subsets.
func TestSubseto_GenerateSubsets(t *testing.T) {
	superset := List(NewAtom(1), NewAtom(2), NewAtom(3))

	// Request exactly 8 results (2^3 = 8 subsets)
	result := Run(8, func(q *Var) Goal {
		return Subseto(q, superset)
	})

	// 2^3 = 8 subsets (including empty and full set)
	if len(result) != 8 {
		t.Errorf("Expected 8 subsets, got %d", len(result))
	}

	// Verify empty set is included
	hasEmpty := false
	for _, subset := range result {
		if subset.Equal(Nil) {
			hasEmpty = true
			break
		}
	}
	if !hasEmpty {
		t.Errorf("Expected empty set to be in subsets")
	}
}

// TestSubseto_NotSubset tests rejecting non-subsets.
func TestSubseto_NotSubset(t *testing.T) {
	notSubset := List(NewAtom(1), NewAtom(5)) // 5 not in superset
	superset := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Subseto(notSubset, superset)
	})

	if len(result) != 0 {
		t.Errorf("Expected non-subset to be rejected, got %d results", len(result))
	}
}

// TestLengtho_Forward tests computing list length.
func TestLengtho_Forward(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	expected := NewPair(NewAtom("s"), NewPair(NewAtom("s"), NewPair(NewAtom("s"), Nil)))

	result := Run(1, func(q *Var) Goal {
		return Lengtho(list, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestLengtho_EmptyList tests length of empty list.
func TestLengtho_EmptyList(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Lengtho(Nil, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(Nil) {
		t.Errorf("Expected Nil (zero), got %v", result[0])
	}
}

// TestLengthoInt_Forward tests computing integer length.
func TestLengthoInt_Forward(t *testing.T) {
	list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"))

	result := Run(1, func(q *Var) Goal {
		return LengthoInt(list, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(4)) {
		t.Errorf("Expected 4, got %v", result[0])
	}
}

// TestLengthoInt_Verification tests verifying list length.
func TestLengthoInt_Verification(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			LengthoInt(list, NewAtom(3)),
			Eq(q, NewAtom("correct")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected length verification to succeed")
	}

	// Test wrong length
	wrongResult := Run(1, func(q *Var) Goal {
		return LengthoInt(list, NewAtom(5))
	})

	if len(wrongResult) != 0 {
		t.Errorf("Expected wrong length to fail")
	}
}

// TestFlatteno_NestedLists tests flattening nested structures.
func TestFlatteno_NestedLists(t *testing.T) {
	nested := List(
		List(NewAtom(1), NewAtom(2)),
		List(NewAtom(3), List(NewAtom(4), NewAtom(5))),
	)
	expected := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4), NewAtom(5))

	result := Run(1, func(q *Var) Goal {
		return Flatteno(nested, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestFlatteno_SingleAtom tests flattening a single atom.
func TestFlatteno_SingleAtom(t *testing.T) {
	atom := NewAtom(42)
	expected := List(NewAtom(42))

	result := Run(1, func(q *Var) Goal {
		return Flatteno(atom, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result[0])
	}
}

// TestDistincto_AllDistinct tests list with all distinct elements.
func TestDistincto_AllDistinct(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Distincto(list),
			Eq(q, NewAtom("distinct")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected distinct check to succeed")
	}
}

// TestDistincto_HasDuplicates tests list with duplicate elements.
func TestDistincto_HasDuplicates(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(1), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Distincto(list)
	})

	if len(result) != 0 {
		t.Errorf("Expected duplicate check to fail, got %d results", len(result))
	}
}

// TestDistincto_EmptyList tests empty list is distinct.
func TestDistincto_EmptyList(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Distincto(Nil),
			Eq(q, NewAtom("distinct")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected empty list to be distinct")
	}
}

// TestNoto_GoalFails tests Noto succeeds when goal fails.
func TestNoto_GoalFails(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Noto(Eq(NewAtom(1), NewAtom(2))), // This fails
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected Noto to succeed when goal fails")
	}
}

// TestNoto_GoalSucceeds tests Noto fails when goal succeeds.
func TestNoto_GoalSucceeds(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Noto(Eq(NewAtom(1), NewAtom(1))) // This succeeds
	})

	if len(result) != 0 {
		t.Errorf("Expected Noto to fail when goal succeeds, got %d results", len(result))
	}
}

// TestListOps_Integration tests combining multiple list operations.
func TestListOps_Integration(t *testing.T) {
	// Find lists that:
	// 1. Are subsets of [1,2,3]
	// 2. Have all distinct elements
	// This should give all 8 subsets with distinct elements (which is all 8 subsets for a set)

	superset := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(10, func(q *Var) Goal {
		return Conj(
			Subseto(q, superset),
			Distincto(q),
		)
	})

	// Should get 8 subsets: [], [1], [2], [3], [1,2], [1,3], [2,3], [1,2,3]
	if len(result) != 8 {
		t.Errorf("Expected 8 distinct subsets, got %d", len(result))
	}
}

// TestListOps_WithHybridSolver tests list operations with UnifiedStore.
func TestListOps_WithHybridSolver(t *testing.T) {
	// Use Reverso with hybrid solver
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	store := NewUnifiedStore()
	result := Fresh("result")

	goal := Reverso(list, result)
	stream := goal(context.Background(), NewUnifiedStoreAdapter(store))

	results := make([]ConstraintStore, 0)
	for {
		stores, hasMore := stream.Take(10)
		results = append(results, stores...)
		if !hasMore {
			break
		}
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Verify the result
	walked := results[0].GetSubstitution().DeepWalk(result)
	expected := List(NewAtom(3), NewAtom(2), NewAtom(1))

	if !walked.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, walked)
	}
}

// TestRembero_WithConstraints tests Rembero with FD constraints.
func TestRembero_WithConstraints(t *testing.T) {
	// Remove a number from a list, where the number must be > 2
	input := List(NewAtom(1), NewAtom(3), NewAtom(5))

	result := Run(5, func(elem *Var) Goal {
		output := Fresh("output")
		return Conj(
			Rembero(elem, input, output),
			// elem must be one of the values > 2
			Disj(
				Eq(elem, NewAtom(3)),
				Eq(elem, NewAtom(5)),
			),
		)
	})

	// Should find 3 and 5 as removable elements
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	// Verify results
	hasThree := false
	hasFive := false
	for _, r := range result {
		if r.Equal(NewAtom(3)) {
			hasThree = true
		}
		if r.Equal(NewAtom(5)) {
			hasFive = true
		}
	}

	if !hasThree || !hasFive {
		t.Errorf("Expected to find both 3 and 5, got: %v", result)
	}
}

// TestPermuteo_Performance tests permutation generation doesn't hang.
func TestPermuteo_Performance(t *testing.T) {
	// Generate permutations of 4 elements (4! = 24)
	list := List(NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4))

	result := Run(30, func(q *Var) Goal {
		return Permuteo(list, q)
	})

	if len(result) != 24 {
		t.Errorf("Expected 24 permutations, got %d", len(result))
	}
}

// TestPermuteo_LazySemantics tests that Permuteo uses lazy evaluation.
// With lazy semantics (Conde), requesting only 1 solution should be fast
// even for large lists, as it doesn't need to compute all permutations.
func TestPermuteo_LazySemantics(t *testing.T) {
	// Use a larger list that would have many permutations (8! = 40320)
	list := List(
		NewAtom(1), NewAtom(2), NewAtom(3), NewAtom(4),
		NewAtom(5), NewAtom(6), NewAtom(7), NewAtom(8),
	)

	// Request only 1 permutation
	result := Run(1, func(q *Var) Goal {
		return Permuteo(list, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// Verify it's a valid permutation by checking it has the same length
	perm := result[0]
	lenResult := Run(1, func(check *Var) Goal {
		return LengthoInt(perm, check)
	})

	if len(lenResult) == 0 || !lenResult[0].Equal(NewAtom(8)) {
		t.Errorf("Permutation has wrong length: got %v, want 8", lenResult)
	}

	// Request just 5 permutations (not all 40320)
	result5 := Run(5, func(q *Var) Goal {
		return Permuteo(list, q)
	})

	if len(result5) != 5 {
		t.Fatalf("Expected 5 results, got %d", len(result5))
	}

	// Verify all 5 are distinct permutations
	seen := make(map[string]bool)
	for i, perm := range result5 {
		key := perm.String()
		if seen[key] {
			t.Errorf("Duplicate permutation at index %d: %v", i, perm)
		}
		seen[key] = true
	}
}

// TestReverso_LargeList tests reversing larger lists.
func TestReverso_LargeList(t *testing.T) {
	// Create a list of 10 elements
	elements := make([]Term, 10)
	for i := 0; i < 10; i++ {
		elements[i] = NewAtom(i)
	}
	list := List(elements...)

	result := Run(1, func(q *Var) Goal {
		return Reverso(list, q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// Verify reversed list
	reversed := result[0]
	t.Logf("Reversed result: %v", reversed)

	lenResult := Run(1, func(check *Var) Goal {
		return LengthoInt(reversed, check)
	})

	t.Logf("Length result: %v", lenResult)
	if len(lenResult) == 0 {
		t.Errorf("Could not determine length of reversed list")
	} else if !lenResult[0].Equal(NewAtom(10)) {
		t.Errorf("Reversed list has wrong length: got %v, want 10", lenResult[0])
	}
}
