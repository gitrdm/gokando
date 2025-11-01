package minikanren

import (
	"context"
	"testing"
)

// Phase 2 Performance Benchmarks
// These benchmarks measure the performance characteristics of Phase 2 constraint propagation
// and compare against Phase 1 baseline performance.

// BenchmarkPhase1_Baseline measures Phase 1 architecture without propagation constraints.
func BenchmarkPhase1_Baseline(b *testing.B) {
	ctx := context.Background()

	b.Run("NQueens-4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			model := NewModel()
			n := 4
			queens := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				queens[i] = model.NewVariable(NewBitSetDomain(n))
			}
			solver := NewSolver(model)
			_, _ = solver.Solve(ctx, 1)
		}
	})

	b.Run("NQueens-8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			model := NewModel()
			n := 8
			queens := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				queens[i] = model.NewVariable(NewBitSetDomain(n))
			}
			solver := NewSolver(model)
			_, _ = solver.Solve(ctx, 1)
		}
	})

	b.Run("StateCreation-10vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 10)
		for i := 0; i < 10; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				state, _ = solver.SetDomain(state, j, NewBitSetDomainFromValues(10, []int{i % 10}))
			}
		}
	})

	b.Run("DomainOperations-100values", func(b *testing.B) {
		model := NewModel()
		v := model.NewVariable(NewBitSetDomain(100))
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dom := solver.GetDomain(state, v.ID())
			for j := 0; j < 50; j++ {
				dom = dom.Remove(j)
			}
			state, _ = solver.SetDomain(state, v.ID(), dom)
		}
	})
}

// BenchmarkPhase2_Propagation measures Phase 2 constraint propagation overhead.
func BenchmarkPhase2_Propagation(b *testing.B) {
	b.Run("AllDifferent-4vars-4values", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 4)
		for i := 0; i < 4; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(4))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("AllDifferent-8vars-8values", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 8)
		for i := 0; i < 8; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(8))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("AllDifferent-12vars-12values", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 12)
		for i := 0; i < 12; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(12))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("ArithmeticChain-10vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 10)
		for i := 0; i < 10; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(20))
		}
		for i := 0; i < 9; i++ {
			c, _ := NewArithmetic(vars[i], vars[i+1], 1)
			model.AddConstraint(c)
		}
		solver := NewSolver(model)
		state, _ := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(20, []int{10}))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("InequalityChain-10vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 10)
		for i := 0; i < 10; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(100))
		}
		for i := 0; i < 9; i++ {
			c, _ := NewInequality(vars[i], vars[i+1], LessThan)
			model.AddConstraint(c)
		}
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("Mixed-AllDiff+Arith+Ineq", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 6)
		for i := 0; i < 6; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		// AllDifferent on first 4
		c1, _ := NewAllDifferent(vars[:4])
		model.AddConstraint(c1)
		// Arithmetic: vars[4] = vars[0] + 1
		c2, _ := NewArithmetic(vars[0], vars[4], 1)
		model.AddConstraint(c2)
		// Inequality: vars[4] < vars[5]
		c3, _ := NewInequality(vars[4], vars[5], LessThan)
		model.AddConstraint(c3)

		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})
}

// BenchmarkPhase2_NQueens measures N-Queens with AllDifferent constraint.
func BenchmarkPhase2_NQueens(b *testing.B) {
	ctx := context.Background()

	b.Run("NQueens-4-WithAllDiff", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			model := NewModel()
			n := 4
			queens := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				queens[i] = model.NewVariable(NewBitSetDomain(n))
			}
			// Add AllDifferent constraint
			c, _ := NewAllDifferent(queens)
			model.AddConstraint(c)
			solver := NewSolver(model)
			_, _ = solver.Solve(ctx, 1)
		}
	})

	b.Run("NQueens-8-WithAllDiff", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			model := NewModel()
			n := 8
			queens := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				queens[i] = model.NewVariable(NewBitSetDomain(n))
			}
			c, _ := NewAllDifferent(queens)
			model.AddConstraint(c)
			solver := NewSolver(model)
			_, _ = solver.Solve(ctx, 1)
		}
	})
}

// BenchmarkPhase2_MemoryAllocation measures allocation overhead.
func BenchmarkPhase2_MemoryAllocation(b *testing.B) {
	b.Run("Propagate-AllDiff-8vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 8)
		for i := 0; i < 8; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(8))
		}
		constraint, _ := NewAllDifferent(vars)
		model.AddConstraint(constraint)
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("Propagate-ArithChain-10vars", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 10)
		for i := 0; i < 10; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(20))
		}
		for i := 0; i < 9; i++ {
			c, _ := NewArithmetic(vars[i], vars[i+1], 1)
			model.AddConstraint(c)
		}
		solver := NewSolver(model)
		state, _ := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(20, []int{10}))

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = solver.propagate(state)
		}
	})

	b.Run("StateCreation-WithPropagation", func(b *testing.B) {
		model := NewModel()
		vars := make([]*FDVariable, 10)
		for i := 0; i < 10; i++ {
			vars[i] = model.NewVariable(NewBitSetDomain(10))
		}
		for i := 0; i < 9; i++ {
			c, _ := NewArithmetic(vars[i], vars[i+1], 1)
			model.AddConstraint(c)
		}
		solver := NewSolver(model)
		state := (*SolverState)(nil)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 10; j++ {
				state, _ = solver.SetDomain(state, j, NewBitSetDomainFromValues(10, []int{i % 10}))
				state, _ = solver.propagate(state)
			}
		}
	})
}

// BenchmarkPhase2_ScalabilityStress tests scaling characteristics.
func BenchmarkPhase2_ScalabilityStress(b *testing.B) {
	sizes := []int{10, 20, 50}

	for _, n := range sizes {
		b.Run("AllDifferent-vars="+string(rune('0'+n/10)), func(b *testing.B) {
			model := NewModel()
			vars := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				vars[i] = model.NewVariable(NewBitSetDomain(n))
			}
			constraint, _ := NewAllDifferent(vars)
			model.AddConstraint(constraint)
			solver := NewSolver(model)
			state := (*SolverState)(nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = solver.propagate(state)
			}
		})
	}

	chainLengths := []int{10, 20, 50}
	for _, n := range chainLengths {
		b.Run("ArithChain-length="+string(rune('0'+n/10)), func(b *testing.B) {
			model := NewModel()
			vars := make([]*FDVariable, n)
			for i := 0; i < n; i++ {
				vars[i] = model.NewVariable(NewBitSetDomain(100))
			}
			for i := 0; i < n-1; i++ {
				c, _ := NewArithmetic(vars[i], vars[i+1], 1)
				model.AddConstraint(c)
			}
			solver := NewSolver(model)
			state, _ := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(100, []int{50}))

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = solver.propagate(state)
			}
		})
	}
}
