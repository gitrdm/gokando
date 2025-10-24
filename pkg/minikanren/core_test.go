package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestVar tests variable creation and methods.
func TestVar(t *testing.T) {
	t.Run("Fresh creates unique variables", func(t *testing.T) {
		v1 := Fresh("x")
		v2 := Fresh("x")

		if v1.Equal(v2) {
			t.Error("Fresh should create unique variables")
		}

		if v1.id == v2.id {
			t.Error("Fresh variables should have unique IDs")
		}
	})

	t.Run("Variable string representation", func(t *testing.T) {
		v1 := Fresh("test")
		v2 := Fresh("")

		str1 := v1.String()
		str2 := v2.String()

		if str1 == str2 {
			t.Error("Different variables should have different string representations")
		}

		if str1 == "" || str2 == "" {
			t.Error("Variable string representation should not be empty")
		}
	})

	t.Run("Variable equality", func(t *testing.T) {
		v1 := Fresh("x")
		v2 := v1.Clone().(*Var)
		v3 := Fresh("x")

		if !v1.Equal(v2) {
			t.Error("Variable should equal its clone")
		}

		if v1.Equal(v3) {
			t.Error("Different variables should not be equal")
		}
	})

	t.Run("IsVar returns true", func(t *testing.T) {
		v := Fresh("x")
		if !v.IsVar() {
			t.Error("Variable should return true for IsVar()")
		}
	})
}

// TestAtom tests atomic values.
func TestAtom(t *testing.T) {
	t.Run("Atom creation and equality", func(t *testing.T) {
		a1 := NewAtom("hello")
		a2 := NewAtom("hello")
		a3 := NewAtom("world")

		if !a1.Equal(a2) {
			t.Error("Atoms with same value should be equal")
		}

		if a1.Equal(a3) {
			t.Error("Atoms with different values should not be equal")
		}
	})

	t.Run("Atom string representation", func(t *testing.T) {
		a := NewAtom(42)
		if a.String() != "42" {
			t.Errorf("Expected '42', got '%s'", a.String())
		}
	})

	t.Run("IsVar returns false", func(t *testing.T) {
		a := NewAtom("test")
		if a.IsVar() {
			t.Error("Atom should return false for IsVar()")
		}
	})

	t.Run("Atom value access", func(t *testing.T) {
		value := "test"
		a := NewAtom(value)

		if a.Value() != value {
			t.Error("Atom should return its original value")
		}
	})
}

// TestPair tests pair/cons cell functionality.
func TestPair(t *testing.T) {
	t.Run("Pair creation and access", func(t *testing.T) {
		car := NewAtom(1)
		cdr := NewAtom(2)
		pair := NewPair(car, cdr)

		if !pair.Car().Equal(car) {
			t.Error("Pair car should equal original car")
		}

		if !pair.Cdr().Equal(cdr) {
			t.Error("Pair cdr should equal original cdr")
		}
	})

	t.Run("Pair equality", func(t *testing.T) {
		p1 := NewPair(NewAtom(1), NewAtom(2))
		p2 := NewPair(NewAtom(1), NewAtom(2))
		p3 := NewPair(NewAtom(1), NewAtom(3))

		if !p1.Equal(p2) {
			t.Error("Pairs with same structure should be equal")
		}

		if p1.Equal(p3) {
			t.Error("Pairs with different structure should not be equal")
		}
	})

	t.Run("IsVar returns false", func(t *testing.T) {
		p := NewPair(NewAtom(1), NewAtom(2))
		if p.IsVar() {
			t.Error("Pair should return false for IsVar()")
		}
	})

	t.Run("Pair cloning", func(t *testing.T) {
		original := NewPair(NewAtom(1), NewAtom(2))
		cloned := original.Clone().(*Pair)

		if !original.Equal(cloned) {
			t.Error("Cloned pair should equal original")
		}

		// Verify deep copy
		if original.Car() == cloned.Car() {
			t.Error("Clone should be a deep copy")
		}
	})
}

// TestSubstitution tests substitution operations.
func TestSubstitution(t *testing.T) {
	t.Run("Empty substitution", func(t *testing.T) {
		sub := NewSubstitution()

		if sub.Size() != 0 {
			t.Error("Empty substitution should have size 0")
		}

		v := Fresh("x")
		if sub.Lookup(v) != nil {
			t.Error("Lookup in empty substitution should return nil")
		}
	})

	t.Run("Binding and lookup", func(t *testing.T) {
		sub := NewSubstitution()
		v := Fresh("x")
		term := NewAtom("hello")

		newSub := sub.Bind(v, term)

		if newSub == nil {
			t.Fatal("Bind should not return nil for valid binding")
		}

		if newSub.Size() != 1 {
			t.Error("Substitution should have size 1 after binding")
		}

		result := newSub.Lookup(v)
		if !result.Equal(term) {
			t.Error("Lookup should return bound term")
		}

		// Original substitution should be unchanged
		if sub.Size() != 0 {
			t.Error("Original substitution should be unchanged")
		}
	})

	t.Run("Walk follows bindings", func(t *testing.T) {
		sub := NewSubstitution()
		v1 := Fresh("x")
		v2 := Fresh("y")
		term := NewAtom("hello")

		// Bind v1 -> v2, v2 -> term
		sub = sub.Bind(v1, v2)
		sub = sub.Bind(v2, term)

		result := sub.Walk(v1)
		if !result.Equal(term) {
			t.Error("Walk should follow binding chain")
		}
	})

	t.Run("Walk with unbound variable", func(t *testing.T) {
		sub := NewSubstitution()
		v := Fresh("x")

		result := sub.Walk(v)
		if !result.Equal(v) {
			t.Error("Walk should return unbound variable unchanged")
		}
	})

	t.Run("Walk with non-variable", func(t *testing.T) {
		sub := NewSubstitution()
		term := NewAtom("hello")

		result := sub.Walk(term)
		if !result.Equal(term) {
			t.Error("Walk should return non-variable unchanged")
		}
	})
}

// TestUnification tests the unification algorithm.
func TestUnification(t *testing.T) {
	t.Run("Unify atoms", func(t *testing.T) {
		sub := NewSubstitution()
		a1 := NewAtom("hello")
		a2 := NewAtom("hello")
		a3 := NewAtom("world")

		// Same atoms should unify
		result := unify(a1, a2, sub)
		if result == nil {
			t.Error("Same atoms should unify")
		}

		// Different atoms should not unify
		result = unify(a1, a3, sub)
		if result != nil {
			t.Error("Different atoms should not unify")
		}
	})

	t.Run("Unify variable with atom", func(t *testing.T) {
		sub := NewSubstitution()
		v := Fresh("x")
		a := NewAtom("hello")

		result := unify(v, a, sub)
		if result == nil {
			t.Fatal("Variable should unify with atom")
		}

		bound := result.Lookup(v)
		if !bound.Equal(a) {
			t.Error("Variable should be bound to atom")
		}
	})

	t.Run("Unify pairs", func(t *testing.T) {
		sub := NewSubstitution()
		p1 := NewPair(NewAtom(1), NewAtom(2))
		p2 := NewPair(NewAtom(1), NewAtom(2))
		p3 := NewPair(NewAtom(1), NewAtom(3))

		// Same pairs should unify
		result := unify(p1, p2, sub)
		if result == nil {
			t.Error("Same pairs should unify")
		}

		// Different pairs should not unify
		result = unify(p1, p3, sub)
		if result != nil {
			t.Error("Different pairs should not unify")
		}
	})

	t.Run("Unify pairs with variables", func(t *testing.T) {
		sub := NewSubstitution()
		v1 := Fresh("x")
		v2 := Fresh("y")
		p1 := NewPair(v1, v2)
		p2 := NewPair(NewAtom(1), NewAtom(2))

		result := unify(p1, p2, sub)
		if result == nil {
			t.Fatal("Pairs with variables should unify")
		}

		if !result.Lookup(v1).Equal(NewAtom(1)) {
			t.Error("v1 should be bound to 1")
		}

		if !result.Lookup(v2).Equal(NewAtom(2)) {
			t.Error("v2 should be bound to 2")
		}
	})
}

// TestGoals tests basic goal operations.
func TestGoals(t *testing.T) {
	t.Run("Success goal", func(t *testing.T) {
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		stream := Success(ctx, store)
		solutions, hasMore := stream.Take(1)

		if len(solutions) != 1 {
			t.Error("Success should return one solution")
		}

		if hasMore {
			t.Error("Success should not have more solutions")
		}

		if len(solutions[0].GetSubstitution().bindings) != len(store.GetSubstitution().bindings) {
			t.Error("Success should return the original substitution")
		}
	})

	t.Run("Failure goal", func(t *testing.T) {
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())

		stream := Failure(ctx, store)
		solutions, hasMore := stream.Take(1)

		if len(solutions) != 0 {
			t.Error("Failure should return no solutions")
		}

		if hasMore {
			t.Error("Failure should not have more solutions")
		}
	})

	t.Run("Eq goal success", func(t *testing.T) {
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a := NewAtom("hello")

		goal := Eq(v, a)
		stream := goal(ctx, store)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Fatal("Eq should return one solution")
		}

		result := solutions[0].GetBinding(v.ID())
		if result == nil || !result.Equal(a) {
			t.Error("Variable should be bound to atom")
		}
	})

	t.Run("Eq goal failure", func(t *testing.T) {
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		a1 := NewAtom("hello")
		a2 := NewAtom("world")

		goal := Eq(a1, a2)
		stream := goal(ctx, store)
		solutions, _ := stream.Take(1)

		if len(solutions) != 0 {
			t.Error("Eq with different atoms should fail")
		}
	})
}

// TestConjunction tests goal conjunction.
func TestConjunction(t *testing.T) {
	t.Run("Empty conjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())

		goal := Conj()
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Error("Empty conjunction should succeed")
		}
	})

	t.Run("Single goal conjunction", func(t *testing.T) {
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a := NewAtom("hello")

		goal := Eq(v, a)
		stream := goal(ctx, store)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Fatal("Single goal conjunction should succeed")
		}

		result := solutions[0].GetBinding(v.ID())
		if result == nil || !result.Equal(a) {
			t.Error("Variable should be bound to atom")
		}
	})

	t.Run("Multiple goal conjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())
		v1 := Fresh("x")
		v2 := Fresh("y")
		a1 := NewAtom("hello")
		a2 := NewAtom("world")

		goal := Conj(Eq(v1, a1), Eq(v2, a2))
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Fatal("Multiple goal conjunction should succeed")
		}

		if !solutions[0].GetBinding(v1.ID()).Equal(a1) {
			t.Error("v1 should be bound correctly")
		}

		if !solutions[0].GetBinding(v2.ID()).Equal(a2) {
			t.Error("v2 should be bound correctly")
		}
	})

	t.Run("Failing conjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a1 := NewAtom("hello")
		a2 := NewAtom("world")

		goal := Conj(Eq(v, a1), Eq(v, a2))
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 0 {
			t.Error("Contradictory conjunction should fail")
		}
	})
}

// TestDisjunction tests goal disjunction.
func TestDisjunction(t *testing.T) {
	t.Run("Empty disjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())

		goal := Disj()
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 0 {
			t.Error("Empty disjunction should fail")
		}
	})

	t.Run("Single goal disjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a := NewAtom("hello")

		goal := Eq(v, a)
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(1)

		if len(solutions) != 1 {
			t.Fatal("Single goal disjunction should succeed")
		}

		result := solutions[0].GetBinding(v.ID())
		if result == nil || !result.Equal(a) {
			t.Error("Variable should be bound correctly")
		}
	})

	t.Run("Multiple choice disjunction", func(t *testing.T) {
		ctx := context.Background()
		sub := NewLocalConstraintStore(NewGlobalConstraintBus())
		v := Fresh("x")
		a1 := NewAtom("hello")
		a2 := NewAtom("world")

		goal := Disj(Eq(v, a1), Eq(v, a2))
		stream := goal(ctx, sub)
		solutions, _ := stream.Take(2)

		if len(solutions) != 2 {
			t.Fatalf("Disjunction should return 2 solutions, got %d", len(solutions))
		}

		// Check that we get both bindings (order may vary due to concurrency)
		values := make(map[string]bool)
		for _, sol := range solutions {
			val := sol.GetBinding(v.ID())
			if atom, ok := val.(*Atom); ok {
				if str, ok := atom.Value().(string); ok {
					values[str] = true
				}
			}
		}

		if !values["hello"] || !values["world"] {
			t.Error("Should get both 'hello' and 'world' as solutions")
		}
	})
}

// TestRun tests the Run function.
func TestRun(t *testing.T) {
	t.Run("Simple run", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Eq(q, NewAtom("hello"))
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("hello")) {
			t.Error("Result should be 'hello'")
		}
	})

	t.Run("Multiple solutions", func(t *testing.T) {
		results := Run(3, func(q *Var) Goal {
			return Disj(
				Eq(q, NewAtom(1)),
				Eq(q, NewAtom(2)),
				Eq(q, NewAtom(3)),
			)
		})

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		// Verify we got all expected values
		expected := map[int]bool{1: false, 2: false, 3: false}
		for _, result := range results {
			if atom, ok := result.(*Atom); ok {
				if val, ok := atom.Value().(int); ok {
					expected[val] = true
				}
			}
		}

		for val, found := range expected {
			if !found {
				t.Errorf("Expected to find value %d", val)
			}
		}
	})

	t.Run("Run with context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// This would run forever without timeout
		results := RunWithContext(ctx, 1000, func(q *Var) Goal {
			return Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
		})

		// Should get some results but not 1000 due to timeout
		if len(results) > 100 {
			t.Error("Context timeout should limit results")
		}
	})
}

// TestList tests list operations.
func TestList(t *testing.T) {
	t.Run("Empty list", func(t *testing.T) {
		lst := List()

		if !lst.Equal(NewAtom(nil)) {
			t.Error("Empty list should be nil atom")
		}
	})

	t.Run("Single element list", func(t *testing.T) {
		lst := List(NewAtom(1))
		expected := NewPair(NewAtom(1), NewAtom(nil))

		if !lst.Equal(expected) {
			t.Error("Single element list should be (1 . nil)")
		}
	})

	t.Run("Multiple element list", func(t *testing.T) {
		lst := List(NewAtom(1), NewAtom(2), NewAtom(3))

		// Should be (1 . (2 . (3 . nil)))
		if pair, ok := lst.(*Pair); ok {
			if !pair.Car().Equal(NewAtom(1)) {
				t.Error("First element should be 1")
			}

			if cdr, ok := pair.Cdr().(*Pair); ok {
				if !cdr.Car().Equal(NewAtom(2)) {
					t.Error("Second element should be 2")
				}
			} else {
				t.Error("Cdr should be a pair")
			}
		} else {
			t.Error("List should be a pair")
		}
	})
}

// TestConcurrentAccess tests thread safety.
func TestConcurrentAccess(t *testing.T) {
	t.Run("Concurrent variable creation", func(t *testing.T) {
		const numGoroutines = 100
		vars := make([]*Var, numGoroutines)

		// Create variables concurrently
		done := make(chan int, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				vars[index] = Fresh("concurrent")
				done <- index
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all variables are unique
		ids := make(map[int64]bool)
		for _, v := range vars {
			if v == nil {
				t.Error("Variable should not be nil")
				continue
			}
			if ids[v.id] {
				t.Error("Duplicate variable ID found")
			}
			ids[v.id] = true
		}
	})

	t.Run("Concurrent substitution access", func(t *testing.T) {
		sub := NewSubstitution()
		v := Fresh("x")
		term := NewAtom("hello")
		sub = sub.Bind(v, term)

		const numGoroutines = 100
		results := make([]Term, numGoroutines)

		// Access substitution concurrently
		done := make(chan int, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				results[index] = sub.Lookup(v)
				done <- index
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all results are correct
		for i, result := range results {
			if !result.Equal(term) {
				t.Errorf("Result %d should equal bound term", i)
			}
		}
	})
}

// Benchmark tests for performance analysis.
func BenchmarkFresh(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Fresh("bench")
		}
	})
}

func BenchmarkUnification(b *testing.B) {
	sub := NewSubstitution()
	v := Fresh("x")
	term := NewAtom("hello")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unify(v, term, sub)
	}
}

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Run(1, func(q *Var) Goal {
			return Eq(q, NewAtom(i))
		})
	}
}

func BenchmarkDisjunction(b *testing.B) {
	goals := make([]Goal, 10)
	for i := 0; i < 10; i++ {
		val := i
		goals[i] = func(ctx context.Context, store ConstraintStore) *Stream {
			v := Fresh("x")
			return Eq(v, NewAtom(val))(ctx, store)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goal := Disj(goals...)
		ctx := context.Background()
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		stream := goal(ctx, store)
		stream.Take(10)
	}
}
