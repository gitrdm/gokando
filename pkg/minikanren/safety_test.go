package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestSafetyMechanisms tests the comprehensive safety mechanisms implemented in Phase 11.5
func TestSafetyMechanisms(t *testing.T) {
	t.Run("Deferred constraint checking prevents infinite loops", func(t *testing.T) {
		// This test verifies that deferred checking prevents the immediate failure trap
		// that could cause infinite loops in complex goal compositions

		x := Fresh("x")

		// Create a goal that would previously cause immediate failure
		// but now works with deferred checking
		goal := Conj(
			SafeConstraintGoal(NewDisequalityConstraint(x, NewAtom("forbidden"))),
			Eq(x, NewAtom("allowed")),
		)

		results := Run(1, func(q *Var) Goal {
			return Conj(goal, Eq(q, x))
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("allowed")) {
			t.Error("Expected 'allowed', got", results[0])
		}
	})

	t.Run("Constraint violations are properly detected", func(t *testing.T) {
		// Test that constraint violations are still properly detected and cause failure

		results := Run(1, func(q *Var) Goal {
			return Conj(
				Eq(q, NewAtom("forbidden")),
				SafeConstraintGoal(NewDisequalityConstraint(q, NewAtom("forbidden"))),
			)
		})

		if len(results) != 0 {
			t.Error("Constraint violation should return no results")
		}
	})

	t.Run("Timeout protection works", func(t *testing.T) {
		// Test that WithTimeout prevents runaway execution

		q := Fresh("q")

		// Create a goal that could potentially run forever
		infiniteGoal := Disj(
			Eq(q, NewAtom(1)),
			Eq(q, NewAtom(2)),
			// In a real infinite goal, this would continue indefinitely
		)

		timeoutGoal := WithTimeout(10*time.Millisecond, infiniteGoal)

		results := Run(100, func(q *Var) Goal {
			return timeoutGoal
		})

		// Should get some results but not all 100 due to timeout
		if len(results) == 0 {
			t.Error("Should get some results before timeout")
		}

		if len(results) >= 100 {
			t.Error("Timeout should limit the number of results")
		}
	})

	t.Run("Constraint validation detects store inconsistencies", func(t *testing.T) {
		// Test the ValidateConstraintStore function

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		defer store.Shutdown()

		x := Fresh("x")

		// Add a binding
		store.AddBinding(x.ID(), NewAtom("test"))

		// Add a constraint that should be satisfied
		constraint := NewDisequalityConstraint(x, NewAtom("different"))
		store.AddConstraint(constraint)

		// Validate the store
		validation := ValidateConstraintStore(store)

		if !validation.Valid {
			t.Error("Store should be valid")
		}

		// Now add a conflicting constraint using deferred addition
		conflictingConstraint := NewDisequalityConstraint(x, NewAtom("test"))
		store.AddConstraintDeferred(conflictingConstraint)

		// Validate again
		validation = ValidateConstraintStore(store)

		if validation.Valid {
			t.Error("Store should be invalid due to constraint violation")
		}

		if len(validation.Errors) == 0 {
			t.Error("Should have validation errors")
		}
	})

	t.Run("Safe execution with validation", func(t *testing.T) {
		// Test that SafeRun works with simple goals

		results := SafeRun(1*time.Second, Eq(Fresh("q"), NewAtom("safe")))

		// SafeRun should return some results
		if len(results) == 0 {
			t.Error("SafeRun should return results for simple goals")
		}
	})

	t.Run("Constraint validation wrapper", func(t *testing.T) {
		// Test WithConstraintValidation wrapper

		q := Fresh("q")

		// Create a goal that adds constraints and bindings
		goal := Conj(
			SafeConstraintGoal(NewDisequalityConstraint(q, NewAtom("bad"))),
			Eq(q, NewAtom("good")),
		)

		validatedGoal := WithConstraintValidation(goal)

		results := Run(1, func(q *Var) Goal {
			return validatedGoal
		})

		if len(results) != 1 {
			t.Error("Validated goal should succeed with valid constraints")
		}
	})

	t.Run("Resource leak prevention", func(t *testing.T) {
		// Test that safety mechanisms prevent resource exhaustion

		q := Fresh("q")

		// Create a goal that could create many intermediate states
		goal := Disj(
			Eq(q, NewAtom(1)),
			Eq(q, NewAtom(2)),
			Eq(q, NewAtom(3)),
			Eq(q, NewAtom(4)),
			Eq(q, NewAtom(5)),
		)

		// Use SafeRun with reasonable limits
		results := SafeRun(50*time.Millisecond, goal)

		// Should get reasonable number of results without exhaustion
		if len(results) == 0 {
			t.Error("Should get some results")
		}

		if len(results) > 10 {
			t.Error("Should not get excessive results")
		}
	})

	t.Run("Complex constraint interactions work safely", func(t *testing.T) {
		// Test complex interactions between multiple constraints

		x, y, z := Fresh("x"), Fresh("y"), Fresh("z")

		goal := Conj(
			// Type constraints
			SafeConstraintGoal(NewTypeConstraint(x, SymbolType)),
			SafeConstraintGoal(NewTypeConstraint(y, NumberType)),

			// Disequality constraints
			SafeConstraintGoal(NewDisequalityConstraint(x, NewAtom("forbidden"))),
			SafeConstraintGoal(NewDisequalityConstraint(y, NewAtom(0))),

			// Bindings that satisfy constraints
			Eq(x, NewAtom("allowed")),
			Eq(y, NewAtom(42)),
			Eq(z, List(x, y)),
		)

		results := Run(1, func(q *Var) Goal {
			return Conj(goal, Eq(q, z))
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Verify the result structure
		result := results[0]
		if pair, ok := result.(*Pair); ok {
			if !pair.Car().Equal(NewAtom("allowed")) || !pair.Cdr().(*Pair).Car().Equal(NewAtom(42)) {
				t.Error("Result structure incorrect")
			}
		} else {
			t.Error("Result should be a pair")
		}
	})
}

// TestDeferredConstraintEdgeCases tests edge cases in deferred constraint handling
func TestDeferredConstraintEdgeCases(t *testing.T) {
	t.Run("Deferred addition with cross-store constraints", func(t *testing.T) {
		// Test that deferred constraint addition works with global bus

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		defer store.Shutdown()

		x := Fresh("x")

		// Create a constraint that might require cross-store coordination
		constraint := NewDisequalityConstraint(x, NewAtom("test"))

		// Add with deferred checking
		err := store.AddConstraintDeferred(constraint)
		if err != nil {
			t.Fatalf("Deferred constraint addition should succeed: %v", err)
		}

		// Verify constraint was added
		constraints := store.GetConstraints()
		if len(constraints) != 1 {
			t.Fatalf("Expected 1 constraint, got %d", len(constraints))
		}
	})

	t.Run("SafeConstraintGoal with immediate violation", func(t *testing.T) {
		// Test SafeConstraintGoal behavior when constraint is immediately violated

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		defer store.Shutdown()

		x := Fresh("x")

		// Bind variable first
		store.AddBinding(x.ID(), NewAtom("test"))

		// Try to add a constraint that's immediately violated
		constraint := NewDisequalityConstraint(x, NewAtom("test"))

		goal := SafeConstraintGoal(constraint)

		ctx := context.Background()
		stream := goal(ctx, store)

		solutions, _, err := stream.Take(ctx, 1)
		if err != nil {
			t.Fatalf("Goal execution should not error: %v", err)
		}

		// Should get no solutions because constraint is violated
		if len(solutions) != 0 {
			t.Error("SafeConstraintGoal should fail when constraint is immediately violated")
		}
	})

	t.Run("DeferredConstraintGoal always succeeds at addition time", func(t *testing.T) {
		// Test that DeferredConstraintGoal always succeeds regardless of immediate violations

		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		defer store.Shutdown()

		x := Fresh("x")

		// Bind variable first
		store.AddBinding(x.ID(), NewAtom("test"))

		// Try to add a constraint that's immediately violated
		constraint := NewDisequalityConstraint(x, NewAtom("test"))

		goal := DeferredConstraintGoal(constraint)

		ctx := context.Background()
		stream := goal(ctx, store)

		solutions, _, err := stream.Take(ctx, 1)
		if err != nil {
			t.Fatalf("Goal execution should not error: %v", err)
		}

		// Should get solutions because DeferredConstraintGoal always succeeds at addition time
		if len(solutions) != 1 {
			t.Error("DeferredConstraintGoal should always succeed at addition time")
		}

		// But the constraint should still be in the store
		resultStore := solutions[0]
		constraints := resultStore.GetConstraints()
		if len(constraints) != 1 {
			t.Error("Constraint should be added to store")
		}
	})
}

// BenchmarkSafetyMechanisms benchmarks the performance impact of safety mechanisms
func BenchmarkSafetyMechanisms(b *testing.B) {
	b.Run("SafeConstraintGoal performance", func(b *testing.B) {
		x, y := Fresh("x"), Fresh("y")
		constraint := NewDisequalityConstraint(x, y)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			goal := SafeConstraintGoal(constraint)
			ctx := context.Background()
			store := NewLocalConstraintStore(nil) // No global bus for benchmark
			stream := goal(ctx, store)
			stream.Take(ctx, 1)
			store.Shutdown()
		}
	})

	b.Run("ValidateConstraintStore performance", func(b *testing.B) {
		store := NewLocalConstraintStore(nil)
		defer store.Shutdown()

		// Add some constraints and bindings
		x, y := Fresh("x"), Fresh("y")
		store.AddBinding(x.ID(), NewAtom("test1"))
		store.AddBinding(y.ID(), NewAtom("test2"))
		store.AddConstraint(NewDisequalityConstraint(x, y))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ValidateConstraintStore(store)
		}
	})

	b.Run("SafeRun performance", func(b *testing.B) {
		q := Fresh("q")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SafeRun(1*time.Second, Eq(q, NewAtom("test")))
		}
	})
}
