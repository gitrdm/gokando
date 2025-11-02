package minikanren

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestParallelSearch_Basic verifies parallel search finds correct solutions.
func TestParallelSearch_Basic(t *testing.T) {
	// Create a simple AllDifferent problem
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	
	// Add AllDifferent constraint
	constraint, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("Failed to create AllDifferent: %v", err)
	}
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	
	// Solve with parallel search
	ctx := context.Background()
	solutions, err := solver.SolveParallel(ctx, 4, 10)
	if err != nil {
		t.Fatalf("Parallel solve failed: %v", err)
	}
	
	// Verify we found solutions (4! = 24 permutations)
	if len(solutions) == 0 {
		t.Fatal("No solutions found")
	}
	
	// Verify all solutions are valid
	for i, sol := range solutions {
		if len(sol) != 4 {
			t.Errorf("Solution %d has wrong length: got %d, want 4", i, len(sol))
			continue
		}
		
		// Check all different
		seen := make(map[int]bool)
		for _, val := range sol {
			if val < 1 || val > 4 {
				t.Errorf("Solution %d has invalid value: %d", i, val)
			}
			if seen[val] {
				t.Errorf("Solution %d has duplicate value: %d", i, val)
			}
			seen[val] = true
		}
	}
	
	t.Logf("Found %d valid solutions", len(solutions))
}

// TestParallelSearch_CompareWithSequential verifies parallel search finds same solutions.
func TestParallelSearch_CompareWithSequential(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))
	
	// Add AllDifferent - should have 6 solutions (3! permutations)
	constraint, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("Failed to create AllDifferent: %v", err)
	}
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	ctx := context.Background()
	
	// Sequential solve
	seqSolutions, err := solver.Solve(ctx, 10)
	if err != nil {
		t.Fatalf("Sequential solve failed: %v", err)
	}
	
	// Parallel solve
	parSolutions, err := solver.SolveParallel(ctx, 4, 10)
	if err != nil {
		t.Fatalf("Parallel solve failed: %v", err)
	}
	
	// Should find same number of solutions
	if len(seqSolutions) != len(parSolutions) {
		t.Errorf("Solution count mismatch: sequential=%d, parallel=%d", len(seqSolutions), len(parSolutions))
	}
	
	// Convert to sets for comparison (order may differ)
	seqSet := make(map[string]bool)
	for _, sol := range seqSolutions {
		key := fmt.Sprintf("%v", sol)
		seqSet[key] = true
	}
	
	parSet := make(map[string]bool)
	for _, sol := range parSolutions {
		key := fmt.Sprintf("%v", sol)
		parSet[key] = true
	}
	
	// Every parallel solution should be in sequential set
	for key := range parSet {
		if !seqSet[key] {
			t.Errorf("Parallel found solution not in sequential: %s", key)
		}
	}
	
	// Every sequential solution should be in parallel set
	for key := range seqSet {
		if !parSet[key] {
			t.Errorf("Sequential found solution not in parallel: %s", key)
		}
	}
	
	t.Logf("Both methods found %d solutions", len(seqSolutions))
}

// TestParallelSearch_Cancellation verifies context cancellation works correctly.
func TestParallelSearch_Cancellation(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	
	// Add AllDifferent
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	
	// Create cancellable context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	// Solve with parallel search
	solutions, err := solver.SolveParallel(ctx, 4, 1000)
	
	// Should either succeed with some solutions or get cancelled
	if err == context.DeadlineExceeded {
		t.Logf("Correctly cancelled after timeout, found %d solutions", len(solutions))
	} else if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	} else {
		t.Logf("Completed before timeout, found %d solutions", len(solutions))
	}
}

// TestParallelSearch_WorkerScaling tests different numbers of workers.
func TestParallelSearch_WorkerScaling(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(5, NewBitSetDomain(5))
	
	// Add AllDifferent
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	ctx := context.Background()
	
	// Test with different worker counts
	workerCounts := []int{1, 2, 4, 8}
	
	for _, numWorkers := range workerCounts {
		t.Run(fmt.Sprintf("Workers=%d", numWorkers), func(t *testing.T) {
			solver := NewSolver(model)
			
			solutions, err := solver.SolveParallel(ctx, numWorkers, 10)
			if err != nil {
				t.Fatalf("Failed with %d workers: %v", numWorkers, err)
			}
			
			if len(solutions) == 0 {
				t.Errorf("No solutions found with %d workers", numWorkers)
			}
			
			t.Logf("%d workers found %d solutions", numWorkers, len(solutions))
		})
	}
}

// TestParallelSearch_EmptyModel tests parallel search with empty model.
func TestParallelSearch_EmptyModel(t *testing.T) {
	model := NewModel()
	solver := NewSolver(model)
	
	ctx := context.Background()
	solutions, err := solver.SolveParallel(ctx, 4, 10)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// With zero variables, the empty assignment counts as one solution
	if len(solutions) != 1 {
		t.Errorf("Expected 1 empty solution for empty model, got %d", len(solutions))
	}
}

// TestParallelSearch_SingleVariable tests parallel search with just one variable.
func TestParallelSearch_SingleVariable(t *testing.T) {
	model := NewModel()
	model.NewVariables(1, NewBitSetDomain(3))
	
	// No constraints, should find 3 solutions
	solver := NewSolver(model)
	ctx := context.Background()
	
	solutions, err := solver.SolveParallel(ctx, 4, 10)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	
	// Should find all 3 values
	if len(solutions) != 3 {
		t.Errorf("Expected 3 solutions, got %d", len(solutions))
	}
	
	// Verify solutions are correct
	seen := make(map[int]bool)
	for _, sol := range solutions {
		if len(sol) != 1 {
			t.Errorf("Solution has wrong length: %d", len(sol))
			continue
		}
		if sol[0] < 1 || sol[0] > 3 {
			t.Errorf("Invalid solution value: %d", sol[0])
		}
		seen[sol[0]] = true
	}
	
	if len(seen) != 3 {
		t.Errorf("Expected 3 distinct solutions, got %d", len(seen))
	}
}

// TestParallelSearch_NoSolution tests parallel search when no solution exists.
func TestParallelSearch_NoSolution(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(2))
	
	// 3 variables with domain {1,2} but must all be different -> impossible
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	ctx := context.Background()
	
	solutions, err := solver.SolveParallel(ctx, 4, 10)
	if err == nil {
		t.Fatalf("Expected inconsistency error, got nil (solutions=%d)", len(solutions))
	}
}

// TestParallelSearch_RaceDetector runs parallel search with race detector enabled.
func TestParallelSearch_RaceDetector(t *testing.T) {
	// This test is primarily for running with -race flag
	model := NewModel()
	vars := model.NewVariables(4, NewBitSetDomain(4))
	
	// Add AllDifferent
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	ctx := context.Background()
	
	// Run multiple times to increase chance of catching races
	for i := 0; i < 10; i++ {
		_, err := solver.SolveParallel(ctx, 4, 5)
		if err != nil {
			t.Fatalf("Solve iteration %d failed: %v", i, err)
		}
	}
}

// TestParallelSearch_LimitSolutions verifies maxSolutions parameter works.
func TestParallelSearch_LimitSolutions(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(3))
	
	// AllDifferent on 3 vars with domain {1,2,3} has 6 solutions
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	ctx := context.Background()
	
	// Request only 3 solutions
	solutions, err := solver.SolveParallel(ctx, 4, 3)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	
	if len(solutions) > 3 {
		t.Errorf("Expected at most 3 solutions, got %d", len(solutions))
	}
	
	if len(solutions) == 0 {
		t.Error("Expected at least some solutions")
	}
	
	t.Logf("Found %d solutions (limit was 3)", len(solutions))
}

// TestParallelSearch_StressTest runs a larger problem to stress test the implementation.
func TestParallelSearch_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	// 8 variables with AllDifferent
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	
	// Add AllDifferent constraint
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	solver := NewSolver(model)
	ctx := context.Background()
	
	start := time.Now()
	solutions, err := solver.SolveParallel(ctx, runtime.NumCPU(), 10)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	
	if len(solutions) == 0 {
		t.Fatal("No solutions found")
	}
	
	t.Logf("Found %d solutions in %v with %d workers", len(solutions), elapsed, runtime.NumCPU())
}

// TestParallelSearchConfig tests configuration options.
func TestParallelSearchConfig(t *testing.T) {
	cfg := DefaultParallelSearchConfig()
	
	if cfg.NumWorkers <= 0 {
		t.Error("Default NumWorkers should be positive")
	}
	
	if cfg.WorkQueueSize < 1 {
		t.Error("WorkQueueSize should be at least 1")
	}
}

// BenchmarkParallelSearch_vs_Sequential compares parallel vs sequential performance.
func BenchmarkParallelSearch_vs_Sequential(b *testing.B) {
	// Create AllDifferent problem with 8 variables
	model := NewModel()
	vars := model.NewVariables(8, NewBitSetDomain(8))
	
	constraint, _ := NewAllDifferent(vars)
	model.AddConstraint(constraint)
	
	ctx := context.Background()
	
	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.Solve(ctx, 1)
			if err != nil || len(solutions) == 0 {
				b.Fatal("Solve failed")
			}
		}
	})
	
	b.Run("Parallel-2workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 2, 1)
			if err != nil || len(solutions) == 0 {
				b.Fatal("Solve failed")
			}
		}
	})
	
	b.Run("Parallel-4workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 4, 1)
			if err != nil || len(solutions) == 0 {
				b.Fatal("Solve failed")
			}
		}
	})
	
	b.Run("Parallel-8workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			solver := NewSolver(model)
			solutions, err := solver.SolveParallel(ctx, 8, 1)
			if err != nil || len(solutions) == 0 {
				b.Fatal("Solve failed")
			}
		}
	})
}

// TestParallelSearch_DoesNotBlockOnLimit ensures SolveParallel returns promptly
// when maxSolutions is small by canceling and draining the solution channel,
// preventing workers from blocking on send (regression test).
func TestParallelSearch_DoesNotBlockOnLimit(t *testing.T) {
	model := NewModel()
	// Use a problem with many solutions to encourage concurrent solution sends
	vars := model.NewVariables(6, NewBitSetDomain(6))
	alldiff, _ := NewAllDifferent(vars)
	model.AddConstraint(alldiff)

	solver := NewSolver(model)

	// Run with many workers to amplify concurrency conditions
	numWorkers := runtime.NumCPU() * 2
	if numWorkers < 2 {
		numWorkers = 2
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Ask for just 1 solution; previously this could deadlock if the collector
		// stopped reading while workers continued sending.
		ctx := context.Background()
		sols, err := solver.SolveParallel(ctx, numWorkers, 1)
		if err != nil {
			t.Errorf("SolveParallel returned error: %v", err)
			return
		}
		if len(sols) == 0 {
			t.Errorf("expected at least 1 solution, got 0")
		}
	}()

	select {
	case <-done:
		// returned promptly
	case <-time.After(3 * time.Second):
		t.Fatal("SolveParallel blocked when hitting solution limit (regression)")
	}
}

// TestParallelSearch_NQueens validates parallel solving on the classic N-Queens
// problem with proper diagonal modeling using arithmetic constraints. This also
// serves as a meaningful stress test of propagation + parallel search.
func TestParallelSearch_NQueens(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping N-Queens stress test in short mode")
	}

	n := 8
	model := NewModel()

	// Columns for each row: values 1..n
	cols := model.NewVariables(n, NewBitSetDomain(n))

	// Diagonals need larger domains to accommodate offsets. We keep values positive
	// for diag2 by shifting with +n.
	diag1 := model.NewVariables(n, NewBitSetDomain(2*n))
	diag2 := model.NewVariables(n, NewBitSetDomain(2*n))

	// Link diagonals to columns: diag1[i] = cols[i] + i; diag2[i] = cols[i] - i + n
	for i := 0; i < n; i++ {
		c1, err := NewArithmetic(cols[i], diag1[i], i)
		if err != nil {
			t.Fatalf("failed to create arithmetic constraint diag1[%d]: %v", i, err)
		}
		model.AddConstraint(c1)

		c2, err := NewArithmetic(cols[i], diag2[i], -i+n)
		if err != nil {
			t.Fatalf("failed to create arithmetic constraint diag2[%d]: %v", i, err)
		}
		model.AddConstraint(c2)
	}

	// All different across columns and both diagonals
	if c, err := NewAllDifferent(cols); err != nil {
		t.Fatalf("failed to create AllDifferent(cols): %v", err)
	} else {
		model.AddConstraint(c)
	}
	if c, err := NewAllDifferent(diag1); err != nil {
		t.Fatalf("failed to create AllDifferent(diag1): %v", err)
	} else {
		model.AddConstraint(c)
	}
	if c, err := NewAllDifferent(diag2); err != nil {
		t.Fatalf("failed to create AllDifferent(diag2): %v", err)
	} else {
		model.AddConstraint(c)
	}

	solver := NewSolver(model)
	ctx := context.Background()

	// Find all solutions in parallel. The 8-queens problem has exactly 92 solutions.
	solutions, err := solver.SolveParallel(ctx, runtime.NumCPU(), 0)
	if err != nil {
		t.Fatalf("SolveParallel failed: %v", err)
	}

	if len(solutions) != 92 {
		t.Fatalf("expected 92 solutions for %d-queens, got %d", n, len(solutions))
	}
}
