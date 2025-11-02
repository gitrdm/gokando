package minikanren

import (
	"context"
	"runtime"
	"testing"
)

// Phase 4.1 Benchmarks: Parallel Search Performance
//
// These benchmarks measure the performance characteristics of the parallel search
// implementation introduced in Phase 4.1, including:
// - Sequential vs parallel speedup
// - Worker scaling efficiency
// - Overhead of parallel coordination
// - Memory allocation patterns

// BenchmarkSequentialVsParallel_4Queens compares sequential and parallel solving.
func BenchmarkSequentialVsParallel_4Queens(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 0)
			if err != nil || len(solutions) != 24 { // 4! = 24 permutations
				b.Fatalf("Expected 24 solutions, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-1worker", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 1, 0)
			if err != nil || len(solutions) != 24 {
				b.Fatalf("Expected 24 solutions, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-2workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 2, 0)
			if err != nil || len(solutions) != 24 {
				b.Fatalf("Expected 24 solutions, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 0)
			if err != nil || len(solutions) != 24 {
				b.Fatalf("Expected 24 solutions, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-NumCPU", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, runtime.NumCPU(), 0)
			if err != nil || len(solutions) != 24 {
				b.Fatalf("Expected 24 solutions, got %d (err=%v)", len(solutions), err)
			}
		}
	})
}

// BenchmarkSequentialVsParallel_8Queens measures scaling on larger problem.
func BenchmarkSequentialVsParallel_8Queens(b *testing.B) {
	model := NewModel()
	n := 8
	cols := model.NewVariables(n, NewBitSetDomain(n))
	diag1 := model.NewVariables(n, NewBitSetDomain(2*n))
	diag2 := model.NewVariables(n, NewBitSetDomain(2*n))

	// Diagonal constraints
	for i := 0; i < n; i++ {
		c1, _ := NewArithmetic(cols[i], diag1[i], i)
		model.AddConstraint(c1)
		c2, _ := NewArithmetic(cols[i], diag2[i], -i+n)
		model.AddConstraint(c2)
	}

	// AllDifferent constraints
	c, _ := NewAllDifferent(cols)
	model.AddConstraint(c)
	c, _ = NewAllDifferent(diag1)
	model.AddConstraint(c)
	c, _ = NewAllDifferent(diag2)
	model.AddConstraint(c)

	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			// Find first solution only for consistent comparison
			solutions, err := solver.Solve(ctx, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-2workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 2, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d (err=%v)", len(solutions), err)
			}
		}
	})

	b.Run("Parallel-8workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 8, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d (err=%v)", len(solutions), err)
			}
		}
	})
}

// BenchmarkParallelSearchOverhead measures the coordination overhead.
func BenchmarkParallelSearchOverhead(b *testing.B) {
	// Small problem where parallelism should have overhead
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 0)
			if err != nil || len(solutions) != 6 {
				b.Fatalf("Expected 6 solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Parallel-1worker", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 1, 0)
			if err != nil || len(solutions) != 6 {
				b.Fatalf("Expected 6 solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 0)
			if err != nil || len(solutions) != 6 {
				b.Fatalf("Expected 6 solutions, got %d", len(solutions))
			}
		}
	})
}

// BenchmarkParallelSearch_WorkerScaling measures how performance scales with workers.
func BenchmarkParallelSearch_WorkerScaling(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(6, NewBitSetDomain(6))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	workerCounts := []int{1, 2, 4, 8, 16}
	for _, workers := range workerCounts {
		b.Run(string(rune('0'+workers))+"workers", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				solver := NewSolver(model)
				// Find first solution for consistent timing
				solutions, err := solver.SolveParallel(ctx, workers, 1)
				if err != nil || len(solutions) != 1 {
					b.Fatalf("Expected 1 solution, got %d", len(solutions))
				}
			}
		})
	}
}

// BenchmarkParallelSearch_StatePooling measures refcount and pooling overhead.
func BenchmarkParallelSearch_StatePooling(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(5, NewBitSetDomain(5))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	b.Run("Sequential-NoPooling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 5)
			if err != nil || len(solutions) < 1 {
				b.Fatalf("Expected solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Parallel-WithPooling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 5)
			if err != nil || len(solutions) < 1 {
				b.Fatalf("Expected solutions, got %d", len(solutions))
			}
		}
	})
}

// BenchmarkParallelSearch_LimitedSolutions measures early termination efficiency.
func BenchmarkParallelSearch_LimitedSolutions(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	b.Run("FindFirst-Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d", len(solutions))
			}
		}
	})

	b.Run("FindFirst-Parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 1)
			if err != nil || len(solutions) != 1 {
				b.Fatalf("Expected 1 solution, got %d", len(solutions))
			}
		}
	})

	b.Run("Find10-Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 10)
			if err != nil || len(solutions) != 10 {
				b.Fatalf("Expected 10 solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Find10-Parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 10)
			if err != nil || len(solutions) != 10 {
				b.Fatalf("Expected 10 solutions, got %d", len(solutions))
			}
		}
	})
}

// BenchmarkParallelSearch_AllSolutions compares finding all solutions.
func BenchmarkParallelSearch_AllSolutions(b *testing.B) {
	model := NewModel()
	vars := model.NewVariables(5, NewBitSetDomain(5))
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 0)
			if err != nil || len(solutions) != 120 { // 5! = 120
				b.Fatalf("Expected 120 solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Parallel-2workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 2, 0)
			if err != nil || len(solutions) != 120 {
				b.Fatalf("Expected 120 solutions, got %d", len(solutions))
			}
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 0)
			if err != nil || len(solutions) != 120 {
				b.Fatalf("Expected 120 solutions, got %d", len(solutions))
			}
		}
	})
}

// BenchmarkParallelConfig measures configuration overhead.
func BenchmarkParallelConfig(b *testing.B) {
	b.Run("DefaultConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = DefaultParallelSearchConfig()
		}
	})
}
