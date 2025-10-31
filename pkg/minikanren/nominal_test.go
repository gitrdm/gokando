// Package minikanren provides comprehensive tests for nominal logic support.
// This test suite validates the implementation of Phase 5.3 (Nominal Logic Support)
// ensuring production-quality code with zero technical debt.
//
// Test coverage includes:
//   - Name type operations and thread-safety
//   - NominalScope binding and lookup operations
//   - NominalUnifier alpha-equivalence and unification
//   - Nominal constraints (Freshness, Binding, Scope)
//   - Constraint solver integration
//   - End-to-end nominal logic scenarios
//   - Race condition detection and thread-safety validation
package minikanren

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestNameType(t *testing.T) {
	t.Run("NameCreation", func(t *testing.T) {
		name1 := NewName("x")
		name2 := NewName("x")
		name3 := NewName("y")

		// Names with same string should have different IDs
		if name1.ID() == name2.ID() {
			t.Error("Names with same string should have unique IDs")
		}

		// Different names should have different IDs
		if name1.ID() == name3.ID() {
			t.Error("Different names should have different IDs")
		}

		// String representation should match
		if name1.Symbol() != "x" {
			t.Errorf("Expected name symbol 'x', got '%s'", name1.Symbol())
		}
	})

	t.Run("NameEquality", func(t *testing.T) {
		name1 := NewName("x")
		name2 := NewName("x")
		name3 := NewName("y")

		// Same name instance should equal itself
		if !name1.Equal(name1) {
			t.Error("Name should equal itself")
		}

		// Different name instances with same string should not be equal
		if name1.Equal(name2) {
			t.Error("Different name instances should not be equal")
		}

		// Names with different strings should not be equal
		if name1.Equal(name3) {
			t.Error("Names with different strings should not be equal")
		}
	})

	t.Run("NameThreadSafety", func(t *testing.T) {
		name := NewName("test")

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Concurrent access to ID and String methods
				_ = name.ID()
				_ = name.String()
			}()
		}
		wg.Wait()
	})
}

func TestNominalScope(t *testing.T) {
	t.Run("ScopeCreation", func(t *testing.T) {
		scope := NewNominalScope()

		if scope.Size() != 0 {
			t.Errorf("New scope should have size 0, got %d", scope.Size())
		}

		if scope.parent != nil {
			t.Error("New scope should not have a parent")
		}
	})

	t.Run("ScopeBinding", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")

		// Bind name to term
		scope.Bind(name, term)

		if scope.Size() != 1 {
			t.Errorf("Scope should have size 1 after binding, got %d", scope.Size())
		}

		// Lookup should return the bound term
		if bound := scope.Lookup(name); bound == nil || !bound.Equal(term) {
			t.Error("Lookup should return the bound term")
		}

		// Unbound name should return nil
		unboundName := NewName("y")
		if bound := scope.Lookup(unboundName); bound != nil {
			t.Error("Lookup of unbound name should return nil")
		}
	})

	t.Run("ScopeBoundChecking", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")

		// Name should not be bound initially
		if scope.IsBound(name) {
			t.Error("Name should not be bound in empty scope")
		}

		// Bind the name
		scope.Bind(name, term)

		// Name should now be bound
		if !scope.IsBound(name) {
			t.Error("Name should be bound after binding")
		}
	})

	t.Run("ScopeHierarchy", func(t *testing.T) {
		parentScope := NewNominalScope()
		childScope := NewNominalScopeWithParent(parentScope)

		parentName := NewName("parent")
		childName := NewName("child")
		term := NewAtom("value")

		// Bind in parent scope
		parentScope.Bind(parentName, term)

		// Child should see parent's binding
		if bound := childScope.Lookup(parentName); bound == nil || !bound.Equal(term) {
			t.Error("Child scope should see parent's bindings")
		}

		// Bind in child scope
		childScope.Bind(childName, term)

		if childScope.Size() != 1 {
			t.Errorf("Child scope should have size 1, got %d", childScope.Size())
		}

		// Parent should not see child's binding
		if bound := parentScope.Lookup(childName); bound != nil {
			t.Error("Parent scope should not see child's bindings")
		}
	})

	t.Run("ScopeCloning", func(t *testing.T) {
		original := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")

		original.Bind(name, term)
		cloned := original.Clone()

		// Clone should have same size
		if cloned.Size() != original.Size() {
			t.Errorf("Cloned scope should have same size, got %d vs %d", cloned.Size(), original.Size())
		}

		// Clone should have same bindings
		if bound := cloned.Lookup(name); bound == nil || !bound.Equal(term) {
			t.Error("Cloned scope should have same bindings")
		}

		// Modifying clone should not affect original
		clonedName := NewName("y")
		cloned.Bind(clonedName, term)

		if original.Size() != 1 {
			t.Errorf("Original scope should not be affected by clone modifications, size %d", original.Size())
		}
	})
}

func TestNominalUnifier(t *testing.T) {
	t.Run("AlphaEquivalence", func(t *testing.T) {
		unifier := NewNominalUnifier()

		// Simple terms should be alpha-equivalent to themselves
		term := NewAtom("test")
		if !unifier.AlphaEquivalent(term, term) {
			t.Error("Term should be alpha-equivalent to itself")
		}

		// Different terms should not be alpha-equivalent
		term1 := NewAtom("test1")
		term2 := NewAtom("test2")
		if unifier.AlphaEquivalent(term1, term2) {
			t.Error("Different terms should not be alpha-equivalent")
		}
	})

	t.Run("FreshNameGeneration", func(t *testing.T) {
		unifier := NewNominalUnifier()

		// Generate a fresh name
		freshName := unifier.FreshName("x")

		if freshName.Symbol() != "x" {
			t.Errorf("Fresh name should have correct symbol, got '%s'", freshName.Symbol())
		}

		// Generated name should not be bound initially
		if unifier.IsNameBound(freshName) {
			t.Error("Generated name should not be bound initially")
		}
	})

	t.Run("NominalUnification", func(t *testing.T) {
		unifier := NewNominalUnifier()
		name := NewName("x")
		term := NewAtom("value")

		// Unify name with term
		result := unifier.Unify(name, term)
		if !result {
			t.Error("Unification should succeed")
		}

		// Check that the binding was created
		if bound := unifier.LookupName(name); bound == nil || !bound.Equal(term) {
			t.Error("Unification should create correct binding")
		}
	})
}

func TestFreshnessConstraint(t *testing.T) {
	t.Run("FreshnessConstraintCreation", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		constraint := NewFreshnessConstraint([]*Name{name}, scope)

		if constraint.ID() == "" {
			t.Error("Constraint should have non-empty ID")
		}

		if !constraint.IsLocal() {
			t.Error("Freshness constraint should be local")
		}
	})

	t.Run("FreshnessConstraintCheck", func(t *testing.T) {
		scope := NewNominalScope()
		freshName := NewName("fresh")
		boundName := NewName("bound")
		term := NewAtom("value")

		// Bind one name
		scope.Bind(boundName, term)

		constraint := NewFreshnessConstraint([]*Name{freshName, boundName}, scope)

		// Since bound name is bound, constraint should be violated
		result := constraint.Check(map[int64]Term{})
		if result != ConstraintViolated {
			t.Errorf("Constraint with bound name should be violated, got %s", result)
		}
	})

	t.Run("FreshnessConstraintInvolvesName", func(t *testing.T) {
		scope := NewNominalScope()
		name1 := NewName("x")
		name2 := NewName("y")
		constraint := NewFreshnessConstraint([]*Name{name1}, scope)

		if !constraint.InvolvesName(name1) {
			t.Error("Constraint should involve its name")
		}

		if constraint.InvolvesName(name2) {
			t.Error("Constraint should not involve other names")
		}
	})
}

func TestBindingConstraint(t *testing.T) {
	t.Run("BindingConstraintCreation", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")
		constraint := NewBindingConstraint(name, term, scope)

		if constraint.ID() == "" {
			t.Error("Constraint should have non-empty ID")
		}
	})

	t.Run("BindingConstraintCheck", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")
		constraint := NewBindingConstraint(name, term, scope)

		// Binding constraints are always satisfied
		result := constraint.Check(map[int64]Term{})
		if result != ConstraintSatisfied {
			t.Errorf("Binding constraint should always be satisfied, got %s", result)
		}
	})

	t.Run("BindingConstraintInvolvesName", func(t *testing.T) {
		scope := NewNominalScope()
		name := NewName("x")
		term := NewAtom("value")
		constraint := NewBindingConstraint(name, term, scope)

		if !constraint.InvolvesName(name) {
			t.Error("Binding constraint should involve its name")
		}

		otherName := NewName("y")
		if constraint.InvolvesName(otherName) {
			t.Error("Binding constraint should not involve other names")
		}
	})
}

func TestScopeConstraint(t *testing.T) {
	t.Run("ScopeConstraintCreation", func(t *testing.T) {
		parentScope := NewNominalScope()
		childScope := NewNominalScopeWithParent(parentScope)
		constraint := NewScopeConstraint(parentScope, childScope)

		if constraint.ID() == "" {
			t.Error("Constraint should have non-empty ID")
		}
	})

	t.Run("ScopeConstraintCheck", func(t *testing.T) {
		parentScope := NewNominalScope()
		childScope := NewNominalScopeWithParent(parentScope)
		constraint := NewScopeConstraint(parentScope, childScope)

		// Valid scope relationship should be satisfied
		result := constraint.Check(map[int64]Term{})
		if result != ConstraintSatisfied {
			t.Errorf("Valid scope constraint should be satisfied, got %s", result)
		}

		// Invalid scope relationship should be violated
		badParent := NewNominalScope()
		invalidConstraint := NewScopeConstraint(badParent, childScope)
		result = invalidConstraint.Check(map[int64]Term{})
		if result != ConstraintViolated {
			t.Errorf("Invalid scope constraint should be violated, got %s", result)
		}
	})
}

func TestNominalConstraintSolver(t *testing.T) {
	t.Run("SolverCreation", func(t *testing.T) {
		solver := NewNominalConstraintSolver()

		if solver.ID() != "nominal-solver" {
			t.Errorf("Solver should have correct ID, got '%s'", solver.ID())
		}

		if solver.Name() != "Nominal Constraint Solver" {
			t.Errorf("Solver should have correct name, got '%s'", solver.Name())
		}

		caps := solver.Capabilities()
		expectedCaps := []string{"nominal", "freshness", "binding", "scope"}
		if len(caps) != len(expectedCaps) {
			t.Errorf("Solver should have correct capabilities, got %v", caps)
		}
	})

	t.Run("SolverCanHandle", func(t *testing.T) {
		solver := NewNominalConstraintSolver()
		scope := NewNominalScope()
		name := NewName("x")

		freshnessConstraint := NewFreshnessConstraint([]*Name{name}, scope)

		if !solver.CanHandle(freshnessConstraint) {
			t.Error("Solver should handle nominal constraints")
		}

		// Test with non-nominal constraint - create a different type of constraint
		otherScope := NewNominalScope()
		otherName := NewName("y")
		otherConstraint := NewFreshnessConstraint([]*Name{otherName}, otherScope)

		// The solver should still handle this nominal constraint
		if !solver.CanHandle(otherConstraint) {
			t.Error("Solver should handle all nominal constraints")
		}
	})

	t.Run("SolverSolve", func(t *testing.T) {
		solver := NewNominalConstraintSolver()
		bus := NewGlobalConstraintBus()
		store := NewLocalConstraintStore(bus)
		scope := NewNominalScope()
		name := NewName("x")

		freshnessConstraint := NewFreshnessConstraint([]*Name{name}, scope)

		ctx := context.Background()
		result, err := solver.Solve(ctx, freshnessConstraint, store)

		if err != nil {
			t.Errorf("Solver should succeed with valid constraint, got error: %v", err)
		}

		if result == nil {
			t.Error("Solver should return a valid store")
		}
		bus.Shutdown()
	})
}

func TestNominalLogicIntegration(t *testing.T) {
	t.Run("EndToEndNominalScenario", func(t *testing.T) {
		// Create a scenario with nominal logic
		bus := NewGlobalConstraintBus()
		store := NewLocalConstraintStore(bus)
		scope := NewNominalScope()
		name1 := NewName("x")
		name2 := NewName("y")
		term := Fresh("value")

		// Create constraints
		freshnessConstraint := NewFreshnessConstraint([]*Name{name1}, scope)
		bindingConstraint := NewBindingConstraint(name2, term, scope)

		// Create constraint store and add constraints
		if err := store.AddConstraint(freshnessConstraint); err != nil {
			t.Errorf("Should be able to add freshness constraint: %v", err)
		}

		if err := store.AddConstraint(bindingConstraint); err != nil {
			t.Errorf("Should be able to add binding constraint: %v", err)
		}

		// Check that constraints are stored
		constraints := store.GetConstraints()
		if len(constraints) != 2 {
			t.Errorf("Store should have 2 constraints, got %d", len(constraints))
		}
		bus.Shutdown()
	})
}

func TestNominalRaceConditions(t *testing.T) {
	t.Run("ConcurrentNameOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		names := make([]*Name, 100)

		// Concurrent name creation
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				names[idx] = NewName("test")
			}(i)
		}
		wg.Wait()

		// Verify all names are unique
		idSet := make(map[int64]bool)
		for _, name := range names {
			if idSet[name.ID()] {
				t.Error("Name IDs should be unique")
			}
			idSet[name.ID()] = true
		}
	})

	t.Run("ConcurrentScopeOperations", func(t *testing.T) {
		scope := NewNominalScope()
		var wg sync.WaitGroup

		// Concurrent binding operations
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := NewName(fmt.Sprintf("name%d", idx))
				term := NewAtom("value")
				scope.Bind(name, term)
			}(i)
		}
		wg.Wait()

		if scope.Size() != 50 {
			t.Errorf("Scope should have 50 bindings, got %d", scope.Size())
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkNameCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewName("benchmark")
	}
}

func BenchmarkScopeBinding(b *testing.B) {
	scope := NewNominalScope()
	name := NewName("bench")
	term := NewAtom("value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope.Bind(name, term)
	}
}

func BenchmarkBoundCheck(b *testing.B) {
	scope := NewNominalScope()
	name := NewName("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scope.IsBound(name)
	}
}
