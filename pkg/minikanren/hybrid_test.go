// Package minikanren provides tests for the hybrid solver framework (Phase 3).
// These tests ensure the plugin architecture works correctly and that
// relational and FD solvers can cooperate on hybrid problems.
package minikanren

import (
	"testing"
)

// ============================================================================
// UnifiedStore Tests
// ============================================================================

func TestNewUnifiedStore(t *testing.T) {
	store := NewUnifiedStore()

	if store.parent != nil {
		t.Error("NewUnifiedStore should have nil parent")
	}
	if store.depth != 0 {
		t.Errorf("depth = %d, want 0", store.depth)
	}
	if len(store.relationalBindings) != 0 {
		t.Errorf("relationalBindings = %d, want 0", len(store.relationalBindings))
	}
	if len(store.fdDomains) != 0 {
		t.Errorf("fdDomains = %d, want 0", len(store.fdDomains))
	}
}

func TestUnifiedStore_Clone(t *testing.T) {
	original := NewUnifiedStore()
	clone := original.Clone()

	if clone.parent != original {
		t.Error("Clone parent should point to original")
	}
	if clone.depth != 1 {
		t.Errorf("Clone depth = %d, want 1", clone.depth)
	}
	if len(clone.relationalBindings) != 0 {
		t.Error("Clone should have empty local bindings")
	}
}

func TestUnifiedStore_AddBinding(t *testing.T) {
	t.Run("add single binding", func(t *testing.T) {
		store := NewUnifiedStore()
		term := NewAtom(42)

		newStore, err := store.AddBinding(1, term)
		if err != nil {
			t.Fatalf("AddBinding failed: %v", err)
		}

		// Original store unchanged
		if store.GetBinding(1) != nil {
			t.Error("Original store should be unchanged")
		}

		// New store has binding
		if newStore.GetBinding(1) != term {
			t.Error("New store should have binding")
		}
	})

	t.Run("shadowing parent binding", func(t *testing.T) {
		store1 := NewUnifiedStore()
		term1 := NewAtom(1)
		store2, _ := store1.AddBinding(1, term1)

		// Shadow with new binding
		term2 := NewAtom(2)
		store3, _ := store2.AddBinding(1, term2)

		// store2 still has original binding
		if !store2.GetBinding(1).Equal(term1) {
			t.Error("Parent store binding changed")
		}

		// store3 has new binding
		if !store3.GetBinding(1).Equal(term2) {
			t.Error("Child store has wrong binding")
		}
	})
}

func TestUnifiedStore_GetBinding(t *testing.T) {
	t.Run("unbound variable", func(t *testing.T) {
		store := NewUnifiedStore()
		if store.GetBinding(999) != nil {
			t.Error("Unbound variable should return nil")
		}
	})

	t.Run("walks parent chain", func(t *testing.T) {
		store1 := NewUnifiedStore()
		term := NewAtom("hello")
		store2, _ := store1.AddBinding(1, term)
		store3 := store2.Clone()

		// store3 should find binding in parent
		if !store3.GetBinding(1).Equal(term) {
			t.Error("Failed to walk parent chain")
		}
	})
}

func TestUnifiedStore_SetDomain(t *testing.T) {
	t.Run("set valid domain", func(t *testing.T) {
		store := NewUnifiedStore()
		domain := NewBitSetDomain(10)

		newStore, err := store.SetDomain(1, domain)
		if err != nil {
			t.Fatalf("SetDomain failed: %v", err)
		}

		// Original unchanged
		if store.GetDomain(1) != nil {
			t.Error("Original store should be unchanged")
		}

		// New store has domain
		if !newStore.GetDomain(1).Equal(domain) {
			t.Error("New store should have domain")
		}
	})

	t.Run("empty domain returns error", func(t *testing.T) {
		store := NewUnifiedStore()
		emptyDomain := NewBitSetDomainFromValues(10, []int{})

		_, err := store.SetDomain(1, emptyDomain)
		if err == nil {
			t.Error("SetDomain should reject empty domain")
		}
	})

	t.Run("domain shadowing", func(t *testing.T) {
		store1 := NewUnifiedStore()
		domain1 := NewBitSetDomain(10)
		store2, _ := store1.SetDomain(1, domain1)

		domain2 := NewBitSetDomainFromValues(10, []int{5})
		store3, _ := store2.SetDomain(1, domain2)

		// Parent unchanged
		if !store2.GetDomain(1).Equal(domain1) {
			t.Error("Parent domain changed")
		}

		// Child has new domain
		if !store3.GetDomain(1).Equal(domain2) {
			t.Error("Child has wrong domain")
		}
	})
}

func TestUnifiedStore_GetDomain(t *testing.T) {
	t.Run("no domain", func(t *testing.T) {
		store := NewUnifiedStore()
		if store.GetDomain(999) != nil {
			t.Error("Variable with no domain should return nil")
		}
	})

	t.Run("walks parent chain", func(t *testing.T) {
		store1 := NewUnifiedStore()
		domain := NewBitSetDomain(10)
		store2, _ := store1.SetDomain(1, domain)
		store3 := store2.Clone()

		// store3 should find domain in parent
		if !store3.GetDomain(1).Equal(domain) {
			t.Error("Failed to walk parent chain for domain")
		}
	})
}

func TestUnifiedStore_AddConstraint(t *testing.T) {
	store := NewUnifiedStore()
	constraint := NewTypeConstraint(Fresh("x"), NumberType)

	newStore := store.AddConstraint(constraint)

	// Original unchanged
	if len(store.GetConstraints()) != 0 {
		t.Error("Original store constraints changed")
	}

	// New store has constraint
	constraints := newStore.GetConstraints()
	if len(constraints) != 1 {
		t.Fatalf("len(constraints) = %d, want 1", len(constraints))
	}

	if constraints[0] != constraint {
		t.Error("Constraint not stored correctly")
	}
}

func TestUnifiedStore_GetSubstitution(t *testing.T) {
	t.Run("empty store", func(t *testing.T) {
		store := NewUnifiedStore()
		sub := store.GetSubstitution()

		if sub.Size() != 0 {
			t.Errorf("substitution size = %d, want 0", sub.Size())
		}
	})

	t.Run("with bindings", func(t *testing.T) {
		store := NewUnifiedStore()
		v1 := Fresh("x")
		term1 := NewAtom(42)
		store, _ = store.AddBinding(v1.id, term1)

		v2 := Fresh("y")
		term2 := NewAtom("hello")
		store, _ = store.AddBinding(v2.id, term2)

		sub := store.GetSubstitution()

		if sub.Size() != 2 {
			t.Errorf("substitution size = %d, want 2", sub.Size())
		}

		if !sub.Lookup(v1).Equal(term1) {
			t.Error("v1 binding incorrect in substitution")
		}

		if !sub.Lookup(v2).Equal(term2) {
			t.Error("v2 binding incorrect in substitution")
		}
	})
}

func TestUnifiedStore_String(t *testing.T) {
	store := NewUnifiedStore()
	v := Fresh("x")
	store, _ = store.AddBinding(v.id, NewAtom(5))
	store, _ = store.SetDomain(1, NewBitSetDomain(10))

	str := store.String()

	// Should contain depth, bindings count, domains count
	if str == "" {
		t.Error("String() returned empty string")
	}
}

// ============================================================================
// HybridSolver Tests
// ============================================================================

func TestNewHybridSolver(t *testing.T) {
	solver := NewHybridSolver()

	if len(solver.GetPlugins()) != 0 {
		t.Errorf("NewHybridSolver should have no plugins")
	}

	if solver.config == nil {
		t.Error("NewHybridSolver should have default config")
	}
}

func TestNewHybridSolverWithConfig(t *testing.T) {
	config := &HybridSolverConfig{
		MaxPropagationIterations: 50,
		EnablePropagation:        false,
	}

	solver := NewHybridSolverWithConfig(config)

	if solver.config.MaxPropagationIterations != 50 {
		t.Error("Config not applied correctly")
	}
}

func TestHybridSolver_RegisterPlugin(t *testing.T) {
	solver := NewHybridSolver()
	plugin := NewRelationalPlugin()

	solver.RegisterPlugin(plugin)

	plugins := solver.GetPlugins()
	if len(plugins) != 1 {
		t.Fatalf("len(plugins) = %d, want 1", len(plugins))
	}

	if plugins[0] != plugin {
		t.Error("Plugin not registered correctly")
	}
}

func TestHybridSolver_SetConfig(t *testing.T) {
	solver := NewHybridSolver()
	newConfig := &HybridSolverConfig{
		MaxPropagationIterations: 100,
		EnablePropagation:        false,
	}

	solver.SetConfig(newConfig)

	if solver.config != newConfig {
		t.Error("SetConfig did not update config")
	}
}

func TestHybridSolver_Propagate_Disabled(t *testing.T) {
	solver := NewHybridSolver()
	solver.config.EnablePropagation = false

	store := NewUnifiedStore()
	newStore, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	if newStore != store {
		t.Error("Propagate should return original store when disabled")
	}
}

func TestHybridSolver_Propagate_NoPlugins(t *testing.T) {
	solver := NewHybridSolver()
	store := NewUnifiedStore()

	newStore, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	if newStore != store {
		t.Error("Propagate with no plugins should return original store")
	}
}

func TestHybridSolver_CanHandle(t *testing.T) {
	solver := NewHybridSolver()
	relPlugin := NewRelationalPlugin()
	solver.RegisterPlugin(relPlugin)

	// Create a relational constraint
	constraint := NewTypeConstraint(Fresh("x"), NumberType)

	handlers := solver.CanHandle(constraint)

	if len(handlers) != 1 {
		t.Fatalf("len(handlers) = %d, want 1", len(handlers))
	}

	if handlers[0].Name() != "Relational" {
		t.Errorf("handler name = %s, want Relational", handlers[0].Name())
	}
}

func TestHybridSolver_String(t *testing.T) {
	solver := NewHybridSolver()
	solver.RegisterPlugin(NewRelationalPlugin())

	str := solver.String()

	if str == "" {
		t.Error("String() returned empty")
	}
}

// ============================================================================
// FDPlugin Tests
// ============================================================================

func TestFDPlugin_Name(t *testing.T) {
	model := NewModel()
	plugin := NewFDPlugin(model)

	if plugin.Name() != "FD" {
		t.Errorf("Name() = %s, want FD", plugin.Name())
	}
}

func TestFDPlugin_CanHandle(t *testing.T) {
	model := NewModel()
	plugin := NewFDPlugin(model)

	t.Run("handles PropagationConstraint", func(t *testing.T) {
		v1 := model.NewVariable(NewBitSetDomain(10))
		v2 := model.NewVariable(NewBitSetDomain(10))
		constraint, _ := NewArithmetic(v1, v2, 1)

		if !plugin.CanHandle(constraint) {
			t.Error("Should handle PropagationConstraint")
		}
	})

	t.Run("rejects relational Constraint", func(t *testing.T) {
		constraint := NewTypeConstraint(Fresh("x"), NumberType)

		if plugin.CanHandle(constraint) {
			t.Error("Should not handle relational Constraint")
		}
	})

	t.Run("rejects non-constraint", func(t *testing.T) {
		if plugin.CanHandle("not a constraint") {
			t.Error("Should not handle string")
		}
	})
}

func TestFDPlugin_Propagate_NoConstraints(t *testing.T) {
	model := NewModel()
	plugin := NewFDPlugin(model)
	store := NewUnifiedStore()

	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	if newStore != store {
		t.Error("Propagate with no constraints should return original store")
	}
}

func TestFDPlugin_Propagate_WithConstraints(t *testing.T) {
	model := NewModel()
	v1 := model.NewVariable(NewBitSetDomain(10))
	v2 := model.NewVariable(NewBitSetDomain(10))

	// Add constraint: v1 + 5 = v2
	constraint, _ := NewArithmetic(v1, v2, 5)
	model.AddConstraint(constraint)

	plugin := NewFDPlugin(model)
	store := NewUnifiedStore()

	// Set initial domains in store
	store, _ = store.SetDomain(v1.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	store, _ = store.SetDomain(v2.ID(), NewBitSetDomain(10))

	// Propagate
	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// v2 should be pruned to {6, 7, 8}
	v2Domain := newStore.GetDomain(v2.ID())
	if v2Domain == nil {
		t.Fatal("v2 domain should be set")
	}

	if !v2Domain.Has(6) || !v2Domain.Has(7) || !v2Domain.Has(8) {
		t.Error("v2 domain should contain {6, 7, 8}")
	}

	if v2Domain.Count() != 3 {
		t.Errorf("v2 domain count = %d, want 3", v2Domain.Count())
	}
}

func TestFDPlugin_GetModel(t *testing.T) {
	model := NewModel()
	plugin := NewFDPlugin(model)

	if plugin.GetModel() != model {
		t.Error("GetModel() should return original model")
	}
}

func TestFDPlugin_GetSolver(t *testing.T) {
	model := NewModel()
	plugin := NewFDPlugin(model)

	if plugin.GetSolver() == nil {
		t.Error("GetSolver() should return non-nil solver")
	}
}

// ============================================================================
// RelationalPlugin Tests
// ============================================================================

func TestRelationalPlugin_Name(t *testing.T) {
	plugin := NewRelationalPlugin()

	if plugin.Name() != "Relational" {
		t.Errorf("Name() = %s, want Relational", plugin.Name())
	}
}

func TestRelationalPlugin_CanHandle(t *testing.T) {
	plugin := NewRelationalPlugin()

	t.Run("handles Constraint", func(t *testing.T) {
		constraint := NewTypeConstraint(Fresh("x"), NumberType)

		if !plugin.CanHandle(constraint) {
			t.Error("Should handle Constraint interface")
		}
	})

	t.Run("rejects PropagationConstraint", func(t *testing.T) {
		model := NewModel()
		v1 := model.NewVariable(NewBitSetDomain(10))
		v2 := model.NewVariable(NewBitSetDomain(10))
		constraint, _ := NewArithmetic(v1, v2, 1)

		if plugin.CanHandle(constraint) {
			t.Error("Should not handle PropagationConstraint")
		}
	})

	t.Run("rejects non-constraint", func(t *testing.T) {
		if plugin.CanHandle(42) {
			t.Error("Should not handle integer")
		}
	})
}

func TestRelationalPlugin_Propagate_SatisfiedConstraint(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// Add type constraint: x must be a number
	x := Fresh("x")
	constraint := NewTypeConstraint(x, NumberType)
	store = store.AddConstraint(constraint)

	// Bind x to a number
	store, _ = store.AddBinding(x.id, NewAtom(42))

	// Propagate
	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Should succeed
	if newStore == nil {
		t.Error("Propagate should return non-nil store")
	}
}

func TestRelationalPlugin_Propagate_ViolatedConstraint(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// Add type constraint: x must be a number
	x := Fresh("x")
	constraint := NewTypeConstraint(x, NumberType)
	store = store.AddConstraint(constraint)

	// Bind x to a string (violates constraint)
	store, _ = store.AddBinding(x.id, NewAtom("not a number"))

	// Propagate
	_, err := plugin.Propagate(store)

	if err == nil {
		t.Error("Propagate should detect constraint violation")
	}
}

func TestRelationalPlugin_Propagate_PendingConstraint(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// Add type constraint: x must be a number
	x := Fresh("x")
	constraint := NewTypeConstraint(x, NumberType)
	store = store.AddConstraint(constraint)

	// Don't bind x - constraint is pending

	// Propagate
	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Should succeed (pending is not an error)
	if newStore == nil {
		t.Error("Propagate should return non-nil store")
	}
}

func TestRelationalPlugin_PromoteSingletons(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// Add singleton FD domain
	domain := NewBitSetDomainFromValues(10, []int{7})
	store, _ = store.SetDomain(1, domain)

	// Propagate (should promote singleton to binding)
	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Check that binding was created
	binding := newStore.GetBinding(1)
	if binding == nil {
		t.Fatal("Singleton should be promoted to binding")
	}

	atom, ok := binding.(*Atom)
	if !ok {
		t.Fatal("Binding should be an Atom")
	}

	if atom.Value() != 7 {
		t.Errorf("binding value = %v, want 7", atom.Value())
	}
}

func TestRelationalPlugin_PromoteSingletons_AlreadyBound(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// Add relational binding
	store, _ = store.AddBinding(1, NewAtom(7))

	// Add singleton FD domain with SAME value
	domain := NewBitSetDomainFromValues(10, []int{7})
	store, _ = store.SetDomain(1, domain)

	// Propagate (should succeed since binding and domain agree)
	newStore, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Binding should still exist
	binding := newStore.GetBinding(1)
	if binding == nil {
		t.Error("Binding should still exist")
	}

	// Domain should still be singleton
	resultDomain := newStore.GetDomain(1)
	if !resultDomain.IsSingleton() || resultDomain.SingletonValue() != 7 {
		t.Error("Domain should still be {7}")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestHybrid_Bidirectional_RelationalToFD tests that relational bindings
// propagate to FD domains (x=5 → domain becomes {5}).
func TestHybrid_Bidirectional_RelationalToFD(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Start with FD domain {1,2,3,4,5}
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5}))

	// Add relational binding x=3
	store, _ = store.AddBinding(int64(x.ID()), NewAtom(3))

	// Propagate - should prune FD domain to {3}
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// Check FD domain was pruned
	domain := result.GetDomain(x.ID())
	if !domain.IsSingleton() {
		t.Errorf("Expected singleton domain, got size %d", domain.Count())
	}
	if domain.SingletonValue() != 3 {
		t.Errorf("Expected domain {3}, got value %d", domain.SingletonValue())
	}
}

// TestHybrid_Bidirectional_FDToRelational tests that FD singleton domains
// promote to relational bindings (domain {7} → x=7).
func TestHybrid_Bidirectional_FDToRelational(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")

	// Add constraint: x + 2 = y
	arith, _ := NewArithmetic(x, y, 2)
	model.AddConstraint(arith)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Set x to singleton, y to wide domain
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{5}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Propagate
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// Check x singleton promoted to binding
	xBinding := result.GetBinding(int64(x.ID()))
	if xBinding == nil {
		t.Fatal("x singleton should be promoted to binding")
	}
	if atom, ok := xBinding.(*Atom); ok {
		if atom.Value() != 5 {
			t.Errorf("x binding value = %v, want 5", atom.Value())
		}
	} else {
		t.Error("x binding should be an Atom")
	}

	// Check y singleton promoted to binding
	yBinding := result.GetBinding(int64(y.ID()))
	if yBinding == nil {
		t.Fatal("y singleton should be promoted to binding")
	}
	if atom, ok := yBinding.(*Atom); ok {
		if atom.Value() != 7 {
			t.Errorf("y binding value = %v, want 7", atom.Value())
		}
	} else {
		t.Error("y binding should be an Atom")
	}
}

// TestHybrid_Bidirectional_ConflictDetection tests that conflicts between
// relational bindings and FD domains are detected.
func TestHybrid_Bidirectional_ConflictDetection(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(relPlugin)

	// Create conflict: binding says x=5, domain is {1,2,3}
	store := NewUnifiedStore()
	store, _ = store.AddBinding(int64(x.ID()), NewAtom(5))
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))

	// Propagate should detect conflict
	_, err := solver.Propagate(store)

	if err == nil {
		t.Error("Should detect conflict: binding=5 but domain={1,2,3}")
	}
}

// TestHybrid_Bidirectional_RoundTrip tests full round-trip propagation:
// FD narrows → singleton promotes to binding → binding prunes other FD domains.
func TestHybrid_Bidirectional_RoundTrip(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")
	z := model.NewVariableWithName(NewBitSetDomain(10), "z")

	// x + 1 = y
	arith1, _ := NewArithmetic(x, y, 1)
	model.AddConstraint(arith1)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// x is singleton, y and z have domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{3}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))
	store, _ = store.SetDomain(z.ID(), NewBitSetDomainFromValues(10, []int{3, 4, 5}))

	// Propagate
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// x singleton → x=3 binding
	xBinding := result.GetBinding(int64(x.ID()))
	if xBinding == nil {
		t.Fatal("x should be promoted to binding")
	}

	// FD propagation: x={3} + 1 = y → y={4}
	yDomain := result.GetDomain(y.ID())
	if !yDomain.IsSingleton() || yDomain.SingletonValue() != 4 {
		t.Errorf("y domain should be {4}, got %v", yDomain)
	}

	// y singleton → y=4 binding
	yBinding := result.GetBinding(int64(y.ID()))
	if yBinding == nil {
		t.Fatal("y should be promoted to binding")
	}
}

// TestHybrid_Bidirectional_TypeMismatch tests that non-integer bindings
// with FD domains are detected as conflicts.
func TestHybrid_Bidirectional_TypeMismatch(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(relPlugin)

	// Create type mismatch: string binding with FD domain
	store := NewUnifiedStore()
	store, _ = store.AddBinding(int64(x.ID()), NewAtom("hello"))
	store, _ = store.SetDomain(x.ID(), NewBitSetDomain(10))

	// Propagate should detect type mismatch
	_, err := solver.Propagate(store)

	if err == nil {
		t.Error("Should detect type mismatch: string binding with FD domain")
	}
}

// TestHybrid_Bidirectional_NonAtomicBinding tests that non-atomic bindings
// (pairs, etc.) with FD domains are detected as conflicts.
func TestHybrid_Bidirectional_NonAtomicBinding(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(relPlugin)

	// Create conflict: pair binding with FD domain
	store := NewUnifiedStore()
	pair := NewPair(NewAtom(1), NewAtom(2))
	store, _ = store.AddBinding(int64(x.ID()), pair)
	store, _ = store.SetDomain(x.ID(), NewBitSetDomain(10))

	// Propagate should detect conflict
	_, err := solver.Propagate(store)

	if err == nil {
		t.Error("Should detect conflict: non-atomic binding with FD domain")
	}
}

// TestHybrid_Real_TypeConstraintPlusArithmetic demonstrates a true hybrid problem:
// combining miniKanren type constraints with FD arithmetic constraints.
func TestHybrid_Real_TypeConstraintPlusArithmetic(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")

	// FD constraint: x + 2 = y
	arith, _ := NewArithmetic(x, y, 2)
	model.AddConstraint(arith)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Start with domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Add miniKanren type constraint: y must be a number
	// (In real usage, y would already be a number due to FD, but this tests the mechanism)
	yVar := Fresh("y_rel")
	typeConstraint := NewTypeConstraint(yVar, NumberType)
	store = store.AddConstraint(typeConstraint)

	// Propagate: FD should narrow y to {3,4,5}, then promote singletons
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Hybrid propagation failed: %v", err)
	}

	// FD propagation should work
	yDomain := result.GetDomain(y.ID())
	if yDomain.Count() != 3 {
		t.Errorf("y domain count = %d, want 3", yDomain.Count())
	}

	if !yDomain.Has(3) || !yDomain.Has(4) || !yDomain.Has(5) {
		t.Error("y domain should be {3, 4, 5}")
	}
}

// TestHybrid_Real_RelationalBindingNarrowsFD tests a key hybrid capability:
// miniKanren unification creating a binding that narrows FD domain.
func TestHybrid_Real_RelationalBindingNarrowsFD(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")
	z := model.NewVariableWithName(NewBitSetDomain(10), "z")

	// FD constraints: AllDifferent(x, y, z)
	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Initial domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	store, _ = store.SetDomain(z.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3}))

	// miniKanren unifies x=2 (simulating relational computation)
	store, _ = store.AddBinding(int64(x.ID()), NewAtom(2))

	// Propagate: relational binding → FD domain {2}, then AllDifferent prunes y,z
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// x should be singleton {2}
	xDomain := result.GetDomain(x.ID())
	if !xDomain.IsSingleton() || xDomain.SingletonValue() != 2 {
		t.Errorf("x domain should be {2}, got %v", xDomain)
	}

	// y and z should not contain 2 (AllDifferent propagation)
	yDomain := result.GetDomain(y.ID())
	zDomain := result.GetDomain(z.ID())

	if yDomain.Has(2) {
		t.Error("y domain should not contain 2 after AllDifferent propagation")
	}
	if zDomain.Has(2) {
		t.Error("z domain should not contain 2 after AllDifferent propagation")
	}
}

// TestHybrid_Real_FDNarrowsEnablesRelational tests the reverse direction:
// FD propagation creating a singleton that enables a relational constraint.
func TestHybrid_Real_FDNarrowsEnablesRelational(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")

	// FD: x + 3 = y
	arith, _ := NewArithmetic(x, y, 3)
	model.AddConstraint(arith)

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// x is singleton, y has wide domain
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{4}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Add miniKanren constraint: y must be equal to 7 (but y is unbound)
	// This constraint will be pending until y gets a binding
	yLogicVar := Fresh("y_logic")
	// We can't directly test Eq here as it needs substitution infrastructure,
	// but we can test that singleton promotion happens

	// Propagate: x={4} + 3 → y={7}, then y singleton promotes to binding
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// y should have singleton domain {7}
	yDomain := result.GetDomain(y.ID())
	if !yDomain.IsSingleton() || yDomain.SingletonValue() != 7 {
		t.Errorf("y domain should be {7}, got %v", yDomain)
	}

	// y singleton should be promoted to relational binding
	yBinding := result.GetBinding(int64(y.ID()))
	if yBinding == nil {
		t.Fatal("y singleton should be promoted to binding")
	}

	atom, ok := yBinding.(*Atom)
	if !ok {
		t.Fatal("y binding should be an Atom")
	}

	if atom.Value() != 7 {
		t.Errorf("y binding value = %v, want 7", atom.Value())
	}

	// Now relational constraints on y can fire (they would see y=7)
	_ = yLogicVar // Acknowledge we set this up for future relational constraints
}

func TestHybrid_Integration_RelationalAndFD(t *testing.T) {
	// Create a model with FD variables
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")

	// Add FD constraint: x + 1 = y
	arithConstraint, _ := NewArithmetic(x, y, 1)
	model.AddConstraint(arithConstraint)

	// Create hybrid solver with both plugins
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Create store with initial domains
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{3, 4, 5}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Run propagation
	newStore, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Hybrid propagation failed: %v", err)
	}

	// Check FD propagation worked
	yDomain := newStore.GetDomain(y.ID())
	if yDomain.Count() != 3 {
		t.Errorf("y domain count = %d, want 3", yDomain.Count())
	}

	if !yDomain.Has(4) || !yDomain.Has(5) || !yDomain.Has(6) {
		t.Error("y domain should be {4, 5, 6}")
	}
}

func TestHybrid_Integration_SingletonPromotion(t *testing.T) {
	// Create model with one variable
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	// Create hybrid solver
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Start with singleton domain
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{7}))

	// Add type constraint: x must be a number
	xVar := Fresh("x_rel")
	typeConstraint := NewTypeConstraint(xVar, NumberType)
	store = store.AddConstraint(typeConstraint)

	// Note: For full integration, we'd need to link FD variable IDs to relational variable IDs
	// This is a simplified test showing the plugin cooperation mechanism
	newStore, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// Singleton should be promoted
	binding := newStore.GetBinding(int64(x.ID()))
	if binding == nil {
		t.Fatal("Singleton should be promoted to binding")
	}

	atom := binding.(*Atom)
	if atom.Value() != 7 {
		t.Errorf("promoted value = %v, want 7", atom.Value())
	}
}

func TestHybrid_Integration_ConflictDetection(t *testing.T) {
	// Create model with conflicting constraints
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(10))

	// Add AllDifferent constraint
	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	// Create solver
	fdPlugin := NewFDPlugin(model)
	solver := NewHybridSolver(fdPlugin)

	// Create conflicting state: all three variables must take values from {1, 2}
	store := NewUnifiedStore()
	domain := NewBitSetDomainFromValues(10, []int{1, 2})
	store, _ = store.SetDomain(x.ID(), domain)
	store, _ = store.SetDomain(y.ID(), domain)
	store, _ = store.SetDomain(z.ID(), domain)

	// Propagation should detect conflict (3 variables, only 2 values)
	_, err := solver.Propagate(store)

	if err == nil {
		t.Error("Should detect conflict from AllDifferent with insufficient values")
	}
}

// ============================================================================
// Fixed-Point Convergence Tests
// ============================================================================

// TestHybrid_FixedPoint_MultiRound tests multi-round propagation convergence.
// FD narrows → singleton promotes → relational checks → fixed point.
func TestHybrid_FixedPoint_MultiRound(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(20), "x")
	y := model.NewVariableWithName(NewBitSetDomain(20), "y")
	z := model.NewVariableWithName(NewBitSetDomain(20), "z")

	// x + 5 = y
	arith1, _ := NewArithmetic(x, y, 5)
	model.AddConstraint(arith1)

	// y + 3 = z
	arith2, _ := NewArithmetic(y, z, 3)
	model.AddConstraint(arith2)

	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// x is singleton, others are wide
	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(20, []int{10}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(20))
	store, _ = store.SetDomain(z.ID(), NewBitSetDomain(20))

	// Propagate should converge:
	// Round 1: x={10} + 5 → y={15}, y={15} + 3 → z={18}
	// Round 2: x=10 promoted, y=15 promoted, z=18 promoted
	// Round 3: No changes (fixed point)
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// All should have singleton domains
	xDom := result.GetDomain(x.ID())
	yDom := result.GetDomain(y.ID())
	zDom := result.GetDomain(z.ID())

	if !xDom.IsSingleton() || xDom.SingletonValue() != 10 {
		t.Errorf("x domain = %v, want {10}", xDom)
	}
	if !yDom.IsSingleton() || yDom.SingletonValue() != 15 {
		t.Errorf("y domain = %v, want {15}", yDom)
	}
	if !zDom.IsSingleton() || zDom.SingletonValue() != 18 {
		t.Errorf("z domain = %v, want {18}", zDom)
	}

	// All should have bindings
	if result.GetBinding(int64(x.ID())) == nil {
		t.Error("x should be promoted to binding")
	}
	if result.GetBinding(int64(y.ID())) == nil {
		t.Error("y should be promoted to binding")
	}
	if result.GetBinding(int64(z.ID())) == nil {
		t.Error("z should be promoted to binding")
	}
}

// TestHybrid_FixedPoint_ImmediateConvergence tests that already-converged
// stores don't trigger unnecessary iterations.
func TestHybrid_FixedPoint_ImmediateConvergence(t *testing.T) {
	model := NewModel()
	fdPlugin := NewFDPlugin(model)
	relPlugin := NewRelationalPlugin()
	solver := NewHybridSolver(fdPlugin, relPlugin)

	// Empty store - should converge immediately
	store := NewUnifiedStore()

	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	if result != store {
		t.Error("Empty store should return same store (no changes)")
	}
}

// TestHybrid_FixedPoint_PropagationDisabled tests that propagation can be disabled.
func TestHybrid_FixedPoint_PropagationDisabled(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")
	y := model.NewVariableWithName(NewBitSetDomain(10), "y")

	arith, _ := NewArithmetic(x, y, 1)
	model.AddConstraint(arith)

	fdPlugin := NewFDPlugin(model)
	solver := NewHybridSolver(fdPlugin)
	config := &HybridSolverConfig{
		EnablePropagation: false,
	}
	solver.SetConfig(config)

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{5}))
	store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

	// Propagation disabled - should return unchanged
	result, err := solver.Propagate(store)

	if err != nil {
		t.Fatalf("Propagation failed: %v", err)
	}

	// y should still have full domain (no propagation)
	yDomain := result.GetDomain(y.ID())
	if yDomain.Count() != 10 {
		t.Errorf("With propagation disabled, y domain should be unchanged, got count %d", yDomain.Count())
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

// TestUnifiedStore_Depth tests the depth tracking in parent chains.
func TestUnifiedStore_Depth(t *testing.T) {
	store := NewUnifiedStore()
	if store.Depth() != 0 {
		t.Errorf("Root store depth = %d, want 0", store.Depth())
	}

	clone1 := store.Clone()
	if clone1.Depth() != 1 {
		t.Errorf("First clone depth = %d, want 1", clone1.Depth())
	}

	clone2 := clone1.Clone()
	if clone2.Depth() != 2 {
		t.Errorf("Second clone depth = %d, want 2", clone2.Depth())
	}
}

// TestUnifiedStore_ChangedVariables tests tracking of modified variables.
func TestUnifiedStore_ChangedVariables(t *testing.T) {
	store := NewUnifiedStore()

	// Add binding for variable 5
	store, _ = store.AddBinding(5, NewAtom(42))

	// ChangedVariables should include variable 5
	changed := store.ChangedVariables()
	if !changed[5] {
		t.Error("ChangedVariables should include variable 5")
	}
}

// TestHybridSolver_PropagateWithConstraints tests the convenience method.
func TestHybridSolver_PropagateWithConstraints(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(10), "x")

	fdPlugin := NewFDPlugin(model)
	solver := NewHybridSolver(fdPlugin)

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{3, 4, 5}))

	// Add a type constraint via PropagateWithConstraints
	typeConstraint := NewTypeConstraint(Fresh("x"), NumberType)

	result, err := solver.PropagateWithConstraints(store, typeConstraint)

	if err != nil {
		t.Fatalf("PropagateWithConstraints failed: %v", err)
	}

	// Constraint should be added to store
	constraints := result.GetConstraints()
	if len(constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(constraints))
	}
}

// TestFDPlugin_Propagate_ErrorPath tests error handling in FD plugin.
func TestFDPlugin_Propagate_ErrorPath(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(5), "x")
	y := model.NewVariableWithName(NewBitSetDomain(5), "y")
	z := model.NewVariableWithName(NewBitSetDomain(5), "z")

	// Create impossible constraint: AllDifferent with only 2 values
	allDiff, _ := NewAllDifferent([]*FDVariable{x, y, z})
	model.AddConstraint(allDiff)

	plugin := NewFDPlugin(model)

	store := NewUnifiedStore()
	// Set conflicting domains: 3 variables, only 2 possible values
	domain := NewBitSetDomainFromValues(5, []int{1, 2})
	store, _ = store.SetDomain(x.ID(), domain)
	store, _ = store.SetDomain(y.ID(), domain)
	store, _ = store.SetDomain(z.ID(), domain)

	// Propagate should detect conflict
	_, err := plugin.Propagate(store)

	if err == nil {
		t.Error("Should detect AllDifferent conflict")
	}
}

// TestRelationalPlugin_Propagate_NoConstraints tests empty constraint case.
func TestRelationalPlugin_Propagate_NoConstraints(t *testing.T) {
	plugin := NewRelationalPlugin()
	store := NewUnifiedStore()

	// No constraints, no domains, no bindings
	result, err := plugin.Propagate(store)

	if err != nil {
		t.Fatalf("Propagate with no constraints failed: %v", err)
	}

	if result != store {
		t.Error("Should return same store when no work needed")
	}
}

func TestUnifiedStore_DeepChain(t *testing.T) {
	// Create deep parent chain
	store := NewUnifiedStore()
	for i := 0; i < 100; i++ {
		store, _ = store.AddBinding(int64(i), NewAtom(i))
		store = store.Clone()
	}

	// Should still be able to access all bindings
	bindings := store.getAllBindings()
	if len(bindings) != 100 {
		t.Errorf("len(bindings) = %d, want 100", len(bindings))
	}
}

func TestHybridSolver_MaxIterations(t *testing.T) {
	// Create solver with low max iterations
	config := &HybridSolverConfig{
		MaxPropagationIterations: 2,
		EnablePropagation:        true,
	}
	solver := NewHybridSolverWithConfig(config)

	// Create a plugin that always makes changes (infinite loop)
	alwaysChanges := &mockPluginAlwaysChanges{}
	solver.RegisterPlugin(alwaysChanges)

	store := NewUnifiedStore()
	_, err := solver.Propagate(store)

	if err == nil {
		t.Error("Should fail with max iterations exceeded")
	}
}

// Mock plugin for testing
type mockPluginAlwaysChanges struct{}

func (m *mockPluginAlwaysChanges) Name() string {
	return "Mock"
}

func (m *mockPluginAlwaysChanges) CanHandle(constraint interface{}) bool {
	return false
}

func (m *mockPluginAlwaysChanges) Propagate(store *UnifiedStore) (*UnifiedStore, error) {
	// Always return a new store (simulating infinite changes)
	return store.Clone(), nil
}
