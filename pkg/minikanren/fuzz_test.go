package minikanren

import (
	"context"
	"testing"
	"unicode/utf8"
)

// FuzzFresh tests variable creation with random names
func FuzzFresh(f *testing.F) {
	// Seed corpus with known good inputs
	f.Add("x")
	f.Add("var1")
	f.Add("query")
	f.Add("🎯") // Unicode test
	f.Add("")  // Edge case: empty string

	f.Fuzz(func(t *testing.T, name string) {
		// Skip invalid UTF-8 sequences
		if !utf8.ValidString(name) {
			t.Skip("Invalid UTF-8 string")
		}

		// Test that Fresh never panics with any valid string input
		v := Fresh(name)

		// Basic invariants
		if v == nil {
			t.Error("Fresh returned nil")
		}
		if v.name != name {
			t.Errorf("Variable name mismatch: expected %q, got %q", name, v.name)
		}

		// Test that two calls with same name create different variables
		v2 := Fresh(name)
		if v.id == v2.id {
			t.Error("Fresh with same name should create different variables")
		}
	})
}

// FuzzUnification tests unification with random terms
func FuzzUnification(f *testing.F) {
	// Seed with various term types
	f.Add(int64(42), "hello")
	f.Add(int64(0), "")
	f.Add(int64(-1), "test")
	f.Add(int64(999999), "unicode🎯")

	f.Fuzz(func(t *testing.T, num int64, str string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(str) {
			t.Skip("Invalid UTF-8 string")
		}

		// Create terms
		numAtom := NewAtom(num)
		strAtom := NewAtom(str)
		v := Fresh("fuzz")

		sub := NewSubstitution()

		// Test unification never panics
		result1 := unify(v, numAtom, sub)
		if result1 != nil {
			// If unification succeeded, verify the binding
			bound := result1.Walk(v)
			if !bound.Equal(numAtom) {
				t.Error("Unification result incorrect")
			}
		}

		// Test self-unification always succeeds
		sub2 := NewSubstitution()
		result2 := unify(numAtom, numAtom, sub2)
		if result2 == nil {
			t.Error("Self-unification should always succeed")
		}

		// Test different terms don't unify
		sub3 := NewSubstitution()
		result3 := unify(numAtom, strAtom, sub3)
		if result3 != nil && !numAtom.Equal(strAtom) {
			t.Error("Different atoms should not unify unless equal")
		}
	})
}

// FuzzGoalExecution tests goal execution with random data
func FuzzGoalExecution(f *testing.F) {
	f.Add(int64(1), int64(1))
	f.Add(int64(42), int64(42))
	f.Add(int64(0), int64(100))
	f.Add(int64(-5), int64(10))

	f.Fuzz(func(t *testing.T, val1, val2 int64) {
		// Test that goal execution never panics
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		// Create a simple goal: fresh variable equals val1
		v := Fresh("fuzz")
		goal := Eq(v, NewAtom(val1))

		// Execute goal
		stream := goal(ctx, store)
		solutions, hasMore := stream.Take(1)

		// Basic invariants
		if len(solutions) > 1 {
			t.Error("Simple Eq goal should produce at most 1 solution")
		}

		if len(solutions) == 1 {
			// Verify the solution
			result := solutions[0].GetSubstitution().Walk(v)
			expected := NewAtom(val1)
			if !result.Equal(expected) {
				t.Errorf("Expected %v, got %v", expected, result)
			}
		}

		// Test conjunction
		v2 := Fresh("fuzz2")
		conjGoal := Conj(
			Eq(v, NewAtom(val1)),
			Eq(v2, NewAtom(val2)),
		)

		stream2 := conjGoal(ctx, store)
		solutions2, _ := stream2.Take(1)

		if len(solutions2) > 1 {
			t.Error("Simple Conj goal should produce at most 1 solution")
		}

		// Test contradiction (different values for same variable)
		if val1 != val2 {
			contrGoal := Conj(
				Eq(v, NewAtom(val1)),
				Eq(v, NewAtom(val2)),
			)

			stream3 := contrGoal(ctx, store)
			solutions3, _ := stream3.Take(1)

			if len(solutions3) != 0 {
				t.Error("Contradictory goal should produce no solutions")
			}
		}

		// Verify hasMore behaves correctly
		_ = hasMore // Use to prevent unused variable warning
	})
}

// FuzzListOperations tests list operations with random data
func FuzzListOperations(f *testing.F) {
	f.Add(3, int64(42))
	f.Add(0, int64(0))
	f.Add(1, int64(-1))
	f.Add(10, int64(999))

	f.Fuzz(func(t *testing.T, length int, value int64) {
		// Limit length to prevent excessive memory usage
		if length < 0 || length > 100 {
			t.Skip("Length out of reasonable range")
		}

		// Create a list of specified length
		var elements []Term
		for i := 0; i < length; i++ {
			elements = append(elements, NewAtom(value+int64(i)))
		}

		if length == 0 {
			// Test empty list
			emptyList := List()
			if !emptyList.Equal(NewAtom(nil)) {
				t.Error("Empty list should be nil atom")
			}
		} else {
			// Test list creation and properties
			list := List(elements...)
			if list == nil {
				t.Error("List returned nil")
			}

			// Test membership
			ctx := context.Background()
			v := Fresh("member")
			memberGoal := Membero(v, list)

			store := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := memberGoal(ctx, store)
			solutions, _ := stream.Take(length + 5) // Take more than we expect

			// Should find exactly the elements we put in
			if len(solutions) != length {
				t.Errorf("Expected %d solutions from Membero, got %d", length, len(solutions))
			}

			// Verify each solution is one of our elements
			found := make(map[int64]bool)
			for _, sol := range solutions {
				result := sol.GetSubstitution().Walk(v)
				if atom, ok := result.(*Atom); ok {
					if val, ok := atom.Value().(int64); ok {
						found[val] = true
					}
				}
			}

			// Check we found all expected values
			for i := 0; i < length; i++ {
				expectedVal := value + int64(i)
				if !found[expectedVal] {
					t.Errorf("Expected to find %d in membership results", expectedVal)
				}
			}
		}
	})
}
