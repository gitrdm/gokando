package minikanren

import (
	"testing"
)

// Phase 3 Performance Benchmarks
// These benchmarks measure the performance of the hybrid solver framework
// and compare against Phase 1 (baseline) and Phase 2 (FD propagation) performance.

// BenchmarkPhase3_HybridPropagation measures hybrid solver propagation overhead.
func BenchmarkPhase3_HybridPropagation(b *testing.B) {
	b.Run("FDOnly-AllDifferent-4vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(4))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)

		fdPlugin := NewFDPlugin(model)
		solver := NewHybridSolver(fdPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(4))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("FDOnly-AllDifferent-8vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 8)
		for i := 0; i < 8; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(8))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)

		fdPlugin := NewFDPlugin(model)
		solver := NewHybridSolver(fdPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 8; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(8))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("RelationalOnly-TypeConstraints-4vars", func(b *testing.B) {
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(relPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			typeConstraint := NewTypeConstraint(Fresh("x"), NumberType)
			store = store.AddConstraint(typeConstraint)
			store, _ = store.AddBinding(int64(i), NewAtom(42))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("Hybrid-FD+Relational-4vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(10))
			typeConstraint := NewTypeConstraint(Fresh("x"), NumberType)
			store = store.AddConstraint(typeConstraint)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("Hybrid-Arithmetic-SingletonPromotion", func(b *testing.B) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(10))
		y := model.NewVariable(NewBitSetDomain(10))
		arith, _ := NewArithmetic(x, y, 2)
		model.AddConstraint(arith)

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{5}))
		store, _ = store.SetDomain(y.ID(), NewBitSetDomain(10))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})
}

// BenchmarkPhase3_BidirectionalPropagation measures the cost of bidirectional sync.
func BenchmarkPhase3_BidirectionalPropagation(b *testing.B) {
	b.Run("RelationalToFD-SingleVar", func(b *testing.B) {
		model := NewModel()
		x := model.NewVariable(NewBitSetDomain(10))

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		store, _ = store.SetDomain(x.ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5}))
		store, _ = store.AddBinding(int64(x.ID()), NewAtom(3))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("RelationalToFD-4vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		allDiff, _ := NewAllDifferent(vars)
		model.AddConstraint(allDiff)

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}))
		}
		// Bind first var relationally
		store, _ = store.AddBinding(int64(vars[0].ID()), NewAtom(2))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("FDToRelational-ArithChain", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 5)
		for i := 0; i < 5; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(20))
		}
		for i := 0; i < 4; i++ {
			c, _ := NewArithmetic(vars[i], vars[i+1], 2)
			model.AddConstraint(c)
		}

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		store, _ = store.SetDomain(vars[0].ID(), NewBitSetDomainFromValues(20, []int{5}))
		for i := 1; i < 5; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(20))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})
}

// BenchmarkPhase3_UnifiedStoreOperations measures store overhead.
func BenchmarkPhase3_UnifiedStoreOperations(b *testing.B) {
	b.Run("Clone-EmptyStore", func(b *testing.B) {
		store := NewUnifiedStore()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Clone()
		}
	})

	b.Run("Clone-With10Bindings", func(b *testing.B) {
		store := NewUnifiedStore()
		for i := 0; i < 10; i++ {
			store, _ = store.AddBinding(int64(i), NewAtom(i*10))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Clone()
		}
	})

	b.Run("AddBinding-10vars", func(b *testing.B) {
		store := NewUnifiedStore()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				store, _ = store.AddBinding(int64(j), NewAtom(j))
			}
		}
	})

	b.Run("SetDomain-10vars", func(b *testing.B) {
		store := NewUnifiedStore()
		domain := NewBitSetDomain(10)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				store, _ = store.SetDomain(j, domain)
			}
		}
	})

	b.Run("GetBinding-DeepChain", func(b *testing.B) {
		store := NewUnifiedStore()
		store, _ = store.AddBinding(0, NewAtom(42))
		// Create 10-deep chain
		for i := 0; i < 10; i++ {
			store = store.Clone()
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.GetBinding(0)
		}
	})

	b.Run("GetAllBindings-10bindings", func(b *testing.B) {
		store := NewUnifiedStore()
		for i := 0; i < 10; i++ {
			store, _ = store.AddBinding(int64(i), NewAtom(i*10))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.getAllBindings()
		}
	})
}

// BenchmarkPhase3_MemoryAllocation measures allocation overhead.
func BenchmarkPhase3_MemoryAllocation(b *testing.B) {
	b.Run("Hybrid-Propagate-4vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(10))
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})

	b.Run("StoreClone-10bindings", func(b *testing.B) {
		store := NewUnifiedStore()
		for i := 0; i < 10; i++ {
			store, _ = store.AddBinding(int64(i), NewAtom(i*10))
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.Clone()
		}
	})

	b.Run("BidirectionalSync-4vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}

		fdPlugin := NewFDPlugin(model)
		relPlugin := NewRelationalPlugin()
		solver := NewHybridSolver(fdPlugin, relPlugin)

		store := NewUnifiedStore()
		for i := 0; i < 4; i++ {
			store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}))
		}
		store, _ = store.AddBinding(int64(vars[0].ID()), NewAtom(2))

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.Propagate(store)
		}
	})
}

// BenchmarkPhase3_Scalability tests scaling with problem size.
func BenchmarkPhase3_Scalability(b *testing.B) {
	sizes := []int{4, 8, 12}

	for _, n := range sizes {
		name := "AllDiff-"
		if n < 10 {
			name += string(rune('0' + n))
		} else {
			name += string(rune('0'+n/10)) + string(rune('0'+n%10))
		}

		b.Run(name+"vars-Hybrid", func(b *testing.B) {
			model := NewModel()
			vars := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				vars[i] = model.NewVariable(NewBitSetDomain(n))
			}
			constraint, _ := NewAllDifferent(vars)
			model.AddConstraint(constraint)

			fdPlugin := NewFDPlugin(model)
			relPlugin := NewRelationalPlugin()
			solver := NewHybridSolver(fdPlugin, relPlugin)

			store := NewUnifiedStore()
			for i := 0; i < n; i++ {
				store, _ = store.SetDomain(vars[i].ID(), NewBitSetDomain(n))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = solver.Propagate(store)
			}
		})
	}
}
