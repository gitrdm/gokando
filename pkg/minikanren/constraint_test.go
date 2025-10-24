package minikanren

import (
	"testing"
)

// TestOrderIndependence verifies that constraints are evaluated order-independently.
// This is the core requirement - constraints should behave the same regardless
// of when they are applied during the unification process.
func TestOrderIndependence(t *testing.T) {
	tests := []struct {
		name     string
		goal1    func(*Var) Goal // Constraint first, then unification
		goal2    func(*Var) Goal // Unification first, then constraint
		expected int             // Expected number of solutions
	}{
		{
			name: "disequality constraint order independence",
			goal1: func(x *Var) Goal {
				return Conj(
					Neq(x, NewAtom("forbidden")),
					Eq(x, NewAtom("allowed")),
				)
			},
			goal2: func(x *Var) Goal {
				return Conj(
					Eq(x, NewAtom("allowed")),
					Neq(x, NewAtom("forbidden")),
				)
			},
			expected: 1,
		},
		{
			name: "conflicting disequality should fail",
			goal1: func(x *Var) Goal {
				return Conj(
					Neq(x, NewAtom("forbidden")),
					Eq(x, NewAtom("forbidden")),
				)
			},
			goal2: func(x *Var) Goal {
				return Conj(
					Eq(x, NewAtom("forbidden")),
					Neq(x, NewAtom("forbidden")),
				)
			},
			expected: 0,
		},
		{
			name: "type constraint order independence",
			goal1: func(x *Var) Goal {
				return Conj(
					Symbolo(x),
					Eq(x, NewAtom("symbol")),
				)
			},
			goal2: func(x *Var) Goal {
				return Conj(
					Eq(x, NewAtom("symbol")),
					Symbolo(x),
				)
			},
			expected: 1,
		},
		{
			name: "absence constraint order independence",
			goal1: func(x *Var) Goal {
				return Conj(
					Absento(NewAtom("bad"), x),
					Eq(x, NewAtom("good")),
				)
			},
			goal2: func(x *Var) Goal {
				return Conj(
					Eq(x, NewAtom("good")),
					Absento(NewAtom("bad"), x),
				)
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test first ordering
			results1 := Run(10, tt.goal1)

			// Test second ordering
			results2 := Run(10, tt.goal2)

			// Verify both have the expected number of solutions
			if len(results1) != tt.expected {
				t.Errorf("Goal1 returned %d solutions, expected %d", len(results1), tt.expected)
			}

			if len(results2) != tt.expected {
				t.Errorf("Goal2 returned %d solutions, expected %d", len(results2), tt.expected)
			}

			// Verify the solutions are equivalent
			if len(results1) != len(results2) {
				t.Errorf("Different number of solutions: goal1=%d, goal2=%d", len(results1), len(results2))
				return
			}

			// For positive cases, verify the solutions are the same
			if tt.expected > 0 {
				for i, result1 := range results1 {
					if !result1.Equal(results2[i]) {
						t.Errorf("Solution %d differs: goal1=%v, goal2=%v", i, result1, results2[i])
					}
				}
			}
		})
	}
}

// TestConstraintStoreIsolation verifies that constraint stores properly isolate
// their constraints and don't interfere with each other.
func TestConstraintStoreIsolation(t *testing.T) {
	globalBus := NewGlobalConstraintBus()

	// Create two independent constraint stores
	store1 := NewLocalConstraintStore(globalBus)
	store2 := NewLocalConstraintStore(globalBus)

	x := Fresh("x")
	y := Fresh("y")

	// Add different constraints to each store
	constraint1 := NewDisequalityConstraint(x, NewAtom("value1"))
	constraint2 := NewDisequalityConstraint(y, NewAtom("value2"))

	err1 := store1.AddConstraint(constraint1)
	err2 := store2.AddConstraint(constraint2)

	if err1 != nil {
		t.Errorf("Failed to add constraint to store1: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Failed to add constraint to store2: %v", err2)
	}

	// Verify each store only has its own constraint
	constraints1 := store1.GetConstraints()
	constraints2 := store2.GetConstraints()

	if len(constraints1) != 1 {
		t.Errorf("Store1 should have 1 constraint, has %d", len(constraints1))
	}
	if len(constraints2) != 1 {
		t.Errorf("Store2 should have 1 constraint, has %d", len(constraints2))
	}

	// Verify the constraints are different
	if len(constraints1) > 0 && len(constraints2) > 0 {
		if constraints1[0].ID() == constraints2[0].ID() {
			t.Error("Stores should have different constraints")
		}
	}
}

// TestConstraintCloning verifies that constraint stores can be properly cloned
// for parallel execution without sharing mutable state.
func TestConstraintCloning(t *testing.T) {
	originalStore := NewLocalConstraintStore(NewGlobalConstraintBus())

	x := Fresh("x")

	// Add a constraint to the original store
	constraint := NewDisequalityConstraint(x, NewAtom("forbidden"))
	err := originalStore.AddConstraint(constraint)
	if err != nil {
		t.Fatalf("Failed to add constraint: %v", err)
	}

	// Add a binding to the original store
	err = originalStore.AddBinding(x.id, NewAtom("allowed"))
	if err != nil {
		t.Fatalf("Failed to add binding: %v", err)
	}

	// Clone the store
	clonedStore := originalStore.Clone()

	// Verify the clone has the same constraints and bindings
	originalConstraints := originalStore.GetConstraints()
	clonedConstraints := clonedStore.GetConstraints()

	if len(originalConstraints) != len(clonedConstraints) {
		t.Errorf("Cloned store has different number of constraints: original=%d, clone=%d",
			len(originalConstraints), len(clonedConstraints))
	}

	originalSub := originalStore.GetSubstitution()
	clonedSub := clonedStore.GetSubstitution()

	originalValue := originalSub.Walk(x)
	clonedValue := clonedSub.Walk(x)

	if !originalValue.Equal(clonedValue) {
		t.Errorf("Cloned store has different binding: original=%v, clone=%v",
			originalValue, clonedValue)
	}

	// Verify modifying the clone doesn't affect the original
	y := Fresh("y")
	err = clonedStore.AddBinding(y.id, NewAtom("clone-only"))
	if err != nil {
		t.Fatalf("Failed to add binding to clone: %v", err)
	}

	// Original should not have the new binding
	originalValue2 := originalStore.GetSubstitution().Walk(y)
	if !originalValue2.Equal(y) { // Should still be unbound
		t.Error("Modifying clone affected original store")
	}
}

// TestConstraintViolationDetection verifies that constraint violations
// are properly detected and cause goals to fail.
func TestConstraintViolationDetection(t *testing.T) {
	tests := []struct {
		name          string
		goal          func(*Var) Goal
		shouldSucceed bool
	}{
		{
			name: "valid disequality constraint",
			goal: func(x *Var) Goal {
				return Conj(
					Neq(x, NewAtom("forbidden")),
					Eq(x, NewAtom("allowed")),
				)
			},
			shouldSucceed: true,
		},
		{
			name: "violated disequality constraint",
			goal: func(x *Var) Goal {
				return Conj(
					Neq(x, NewAtom("forbidden")),
					Eq(x, NewAtom("forbidden")),
				)
			},
			shouldSucceed: false,
		},
		{
			name: "valid type constraint",
			goal: func(x *Var) Goal {
				return Conj(
					Symbolo(x),
					Eq(x, NewAtom("symbol")),
				)
			},
			shouldSucceed: true,
		},
		{
			name: "violated type constraint",
			goal: func(x *Var) Goal {
				return Conj(
					Symbolo(x),
					Eq(x, NewAtom(42)),
				)
			},
			shouldSucceed: false,
		},
		{
			name: "valid absence constraint",
			goal: func(x *Var) Goal {
				return Conj(
					Absento(NewAtom("bad"), x),
					Eq(x, NewAtom("good")),
				)
			},
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Run(1, tt.goal)

			if tt.shouldSucceed && len(results) == 0 {
				t.Error("Goal should have succeeded but produced no results")
			}

			if !tt.shouldSucceed && len(results) > 0 {
				t.Errorf("Goal should have failed but produced %d results", len(results))
			}
		})
	}
}

// TestConstraintGlobalCoordination verifies that the global constraint bus
// properly coordinates constraints across multiple stores when needed.
func TestConstraintGlobalCoordination(t *testing.T) {
	globalBus := NewGlobalConstraintBus()

	// Create multiple stores sharing the same global bus
	store1 := NewLocalConstraintStore(globalBus)
	store2 := NewLocalConstraintStore(globalBus)

	// This test verifies the infrastructure is in place
	// More complex global coordination tests would require
	// cross-store constraints, which are part of future work

	if store1.ID() == store2.ID() {
		t.Error("Stores should have different IDs")
	}

	// Verify each store can operate independently
	x := Fresh("x")
	y := Fresh("y")

	err1 := store1.AddBinding(x.id, NewAtom("value1"))
	err2 := store2.AddBinding(y.id, NewAtom("value2"))

	if err1 != nil || err2 != nil {
		t.Error("Stores should be able to operate independently")
	}
}
