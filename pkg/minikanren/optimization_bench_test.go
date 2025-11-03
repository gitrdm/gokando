package minikanren

import (
	"context"
	"testing"
)

// BenchmarkConstraintBusStrategies compares different constraint bus allocation strategies
func BenchmarkConstraintBusStrategies(b *testing.B) {

	b.Run("Original_NewBusPerRun", func(b *testing.B) {
		// Simulate the old approach
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// Old approach: new bus every time
			initialStore := NewLocalConstraintStore(NewGlobalConstraintBus())
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
		}
	})

	b.Run("Optimized_SharedBus", func(b *testing.B) {
		// New approach with shared bus
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// New approach: shared bus
			initialStore := NewLocalConstraintStore(GetDefaultGlobalBus())
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
		}
	})

	b.Run("Optimized_PooledBus", func(b *testing.B) {
		// Pooled approach for isolation
		for i := 0; i < b.N; i++ {
			q := Fresh("q")
			goal := func(ctx context.Context, store ConstraintStore) *Stream {
				return Eq(q, NewAtom(i))(ctx, store)
			}

			// Pooled approach
			bus := GetPooledGlobalBus()
			initialStore := NewLocalConstraintStore(bus)
			stream := goal(context.Background(), initialStore)
			stream.Take(1)
			ReturnPooledGlobalBus(bus)
		}
	})

	b.Run("StandardRun_After_Optimization", func(b *testing.B) {
		// Test the optimized Run function
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("IsolatedRun", func(b *testing.B) {
		// Test the isolated Run function
		for i := 0; i < b.N; i++ {
			RunWithIsolation(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	b.Run("SharedBus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("PooledBus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			RunWithIsolation(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})
}

// BenchmarkSolve_SmallLinearSum benchmarks basic Solve on a small LinearSum instance.
// This provides a baseline: Solve explores states without optimization pruning.
func BenchmarkSolve_SmallLinearSum(b *testing.B) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	total := model.NewVariable(NewBitSetDomain(30))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, total)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = solver.Solve(context.Background(), 1)
	}
}

// BenchmarkSolveOptimal_SmallLinearSum benchmarks SolveOptimal minimizing a LinearSum total.
// Optimization pruning via incumbent cutoffs should reduce search space versus plain Solve.
func BenchmarkSolveOptimal_SmallLinearSum(b *testing.B) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	total := model.NewVariable(NewBitSetDomain(30))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, total)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = solver.SolveOptimal(context.Background(), total, true)
	}
}

// BenchmarkSolve_AllDifferent4 benchmarks Solve on AllDifferent with 4 variables.
// Provides a baseline for comparison with optimization on similar constraints.
func BenchmarkSolve_AllDifferent4(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	ad, _ := NewAllDifferent(vars)
	model.AddConstraint(ad)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = solver.Solve(context.Background(), 1)
	}
}

// BenchmarkSolveOptimal_MinOfArray benchmarks SolveOptimal maximizing a MinOfArray result.
// Demonstrates optimization with structural lower bounds on Min objectives.
func BenchmarkSolveOptimal_MinOfArray(b *testing.B) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 4, 5}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 4, 5, 6, 7}))
	r := model.NewVariable(NewBitSetDomain(10))
	c, _ := NewMin([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = solver.SolveOptimal(context.Background(), r, false) // maximize
	}
}

// BenchmarkSolveOptimal_Parallel4Workers benchmarks parallel branch-and-bound with 4 workers.
// Shows parallel overhead and speedup potential for optimization problems.
func BenchmarkSolveOptimal_Parallel4Workers(b *testing.B) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	total := model.NewVariable(NewBitSetDomain(30))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, total)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = solver.SolveOptimalWithOptions(context.Background(), total, true, WithParallelWorkers(4))
	}
}

// BenchmarkSolveOptimal_Makespan benchmarks makespan minimization with Cumulative and M >= e_i constraints.
// Demonstrates a real-world scheduling optimization scenario with structural lower bounds.
func BenchmarkSolveOptimal_Makespan(b *testing.B) {
	model := NewModel()
	// Two tasks: durations [2,1]
	durations := []int{2, 1}
	s1 := model.NewVariable(NewBitSetDomain(5))
	s2 := model.NewVariable(NewBitSetDomain(5))
	cum, _ := NewCumulative([]*FDVariable{s1, s2}, durations, []int{1, 1}, 1)
	model.AddConstraint(cum)

	// End times: e_i = s_i + dur_i - 1
	e1 := model.NewVariable(NewBitSetDomain(8))
	e2 := model.NewVariable(NewBitSetDomain(8))
	c1, _ := NewArithmetic(s1, e1, durations[0]-1)
	c2, _ := NewArithmetic(s2, e2, durations[1]-1)
	model.AddConstraint(c1)
	model.AddConstraint(c2)

	// Makespan M >= e_i
	m := model.NewVariable(NewBitSetDomain(8))
	ge1, _ := NewInequality(m, e1, GreaterEqual)
	ge2, _ := NewInequality(m, e2, GreaterEqual)
	model.AddConstraint(ge1)
	model.AddConstraint(ge2)

	solver := NewSolver(model)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = solver.SolveOptimal(context.Background(), m, true)
	}
}
