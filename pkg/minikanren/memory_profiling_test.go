package minikanren

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// TestMemoryProfiling demonstrates Go's memory profiling capabilities
// Run with: go test -run=TestMemoryProfiling -memprofile=mem.prof
func TestMemoryProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory profiling test in short mode")
	}

	// Create a memory profile file
	f, err := os.Create("gokando_memory.prof")
	if err != nil {
		t.Fatalf("Could not create memory profile: %v", err)
	}
	defer f.Close()

	// Force garbage collection before starting
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup

	// Record initial memory stats
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform memory-intensive operations
	// Test actual memory usage without accumulating results
	var totalSolutions int
	for i := 0; i < 1000; i++ {
		// Create many variables and goals
		vars := make([]*Var, 100)
		for j := range vars {
			vars[j] = Fresh(fmt.Sprintf("var_%d_%d", i, j))
		}

		// Run goals that generate solutions
		goalResults := Run(10, func(q *Var) Goal {
			goals := make([]Goal, len(vars))
			for k, v := range vars {
				goals[k] = Eq(v, NewAtom(k+i))
			}
			return Disj(goals...)
		})

		// Count solutions but don't accumulate them (prevent memory accumulation)
		totalSolutions += len(goalResults)

		// Periodically force GC to test for leaks
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Force final garbage collection
	runtime.GC()
	runtime.GC()

	// Record final memory stats
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Write memory profile
	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatalf("Could not write memory profile: %v", err)
	}

	// Report memory usage
	t.Logf("Memory usage:")
	t.Logf("  Alloc: %d bytes", m2.Alloc)
	t.Logf("  TotalAlloc: %d bytes", m2.TotalAlloc)
	t.Logf("  Sys: %d bytes", m2.Sys)
	t.Logf("  NumGC: %d", m2.NumGC)
	t.Logf("  Mallocs: %d", m2.Mallocs)
	t.Logf("  Frees: %d", m2.Frees)
	t.Logf("  Live objects: %d", m2.Mallocs-m2.Frees)

	// Check for potential memory leaks
	// The exact threshold depends on your application's requirements
	// Based on scaling analysis: 1000 iterations × 100 vars × 10 solutions = ~110-120MB expected
	const maxAllocMB = 130 // 130MB threshold allows for reasonable variance in intensive workload
	allocMB := m2.Alloc / 1024 / 1024
	if allocMB > maxAllocMB {
		t.Errorf("Potential memory leak: %d MB allocated (threshold: %d MB)", allocMB, maxAllocMB)
	} else {
		t.Logf("✅ Memory usage within expected range: %d MB (threshold: %d MB)", allocMB, maxAllocMB)
	}

	// Ensure we actually processed the data (prevent optimization)
	if totalSolutions != 10000 {
		t.Errorf("Expected 10000 total solutions, got %d", totalSolutions)
	}

	t.Logf("Memory profiling complete. Analyze with:")
	t.Logf("  go tool pprof gokando_memory.prof")
	t.Logf("  (pprof) top10")
	t.Logf("  (pprof) list <function_name>")
	t.Logf("  (pprof) web")
}

// TestCPUProfiling demonstrates CPU profiling
// Run with: go test -run=TestCPUProfiling -cpuprofile=cpu.prof
func TestCPUProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CPU profiling test in short mode")
	}

	// Create a CPU profile file
	f, err := os.Create("gokando_cpu.prof")
	if err != nil {
		t.Fatalf("Could not create CPU profile: %v", err)
	}
	defer f.Close()

	// Start CPU profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatalf("Could not start CPU profile: %v", err)
	}
	defer pprof.StopCPUProfile()

	// Perform CPU-intensive operations
	start := time.Now()

	// Test complex goal execution
	results := Run(1000, func(q *Var) Goal {
		// Create a complex search space
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")

		return Disj(
			Conj(
				Eq(x, NewAtom(1)),
				Eq(y, NewAtom(2)),
				Eq(z, NewAtom(3)),
				Eq(q, List(x, y, z)),
			),
			Conj(
				Eq(x, NewAtom(10)),
				Eq(y, NewAtom(20)),
				Eq(z, NewAtom(30)),
				Eq(q, List(x, y, z)),
			),
			// Add many more disjunctions to create work
			Membero(q, List(
				NewAtom(42), NewAtom(43), NewAtom(44),
				NewAtom(45), NewAtom(46), NewAtom(47),
			)),
		)
	})

	duration := time.Since(start)

	t.Logf("CPU profiling complete:")
	t.Logf("  Processed %d results in %v", len(results), duration)
	t.Logf("  Rate: %.2f results/second", float64(len(results))/duration.Seconds())
	t.Logf("Analyze with:")
	t.Logf("  go tool pprof gokando_cpu.prof")
	t.Logf("  (pprof) top10")
	t.Logf("  (pprof) list <function_name>")
	t.Logf("  (pprof) web")
}

// TestMemoryLeakDetection tests for potential memory leaks
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak detection in short mode")
	}

	// Function to measure memory usage
	measureMem := func() uint64 {
		runtime.GC()
		runtime.GC() // Call twice for thorough cleanup
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.Alloc
	}

	// Baseline measurement
	baseline := measureMem()
	t.Logf("Baseline memory: %d bytes", baseline)

	// Perform operations multiple times
	for round := 0; round < 10; round++ {
		// Create and discard many objects
		for i := 0; i < 100; i++ {
			// Test variable creation and cleanup
			vars := make([]*Var, 50)
			for j := range vars {
				vars[j] = Fresh(fmt.Sprintf("leak_test_%d_%d", round, j))
			}

			// Test constraint store operations
			ctx := context.Background()
			store := NewLocalConstraintStore(GetDefaultGlobalBus()) // Create goals and execute them
			goal := Disj(
				Eq(vars[0], NewAtom(1)),
				Eq(vars[1], NewAtom(2)),
				Eq(vars[2], NewAtom(3)),
			)

			stream := goal(ctx, store)
			solutions, _ := stream.Take(10)

			// Use the solutions to prevent optimization
			_ = len(solutions)
		}

		// Measure memory after each round
		currentMem := measureMem()
		growth := currentMem - baseline
		growthMB := float64(growth) / 1024 / 1024

		t.Logf("Round %d: %d bytes (+%.2f MB from baseline)", round, currentMem, growthMB)

		// Check for excessive growth (indicates potential leak)
		if growthMB > 50 { // 50MB threshold
			t.Errorf("Potential memory leak detected: %.2f MB growth after %d rounds", growthMB, round+1)
			break
		}
	}

	finalMem := measureMem()
	totalGrowth := float64(finalMem-baseline) / 1024 / 1024
	t.Logf("Final memory growth: %.2f MB", totalGrowth)

	// Final leak check
	if totalGrowth > 20 { // 20MB total growth threshold
		t.Errorf("Excessive memory growth detected: %.2f MB total", totalGrowth)
	} else {
		t.Logf("✅ No significant memory leaks detected")
	}
}

// BenchmarkMemoryAllocations benchmarks memory allocation patterns
func BenchmarkMemoryAllocations(b *testing.B) {
	b.Run("Variable Creation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Fresh("bench")
		}
	})

	b.Run("Goal Execution", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Run(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})

	b.Run("Complex Goal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x := Fresh("x")
			y := Fresh("y")
			Run(1, func(q *Var) Goal {
				return Conj(
					Eq(x, NewAtom(i)),
					Eq(y, NewAtom(i*2)),
					Eq(q, List(x, y)),
				)
			})
		}
	})

	b.Run("Parallel Execution", func(b *testing.B) {
		b.ReportAllocs()
		executor := NewParallelExecutor(nil)
		defer executor.Shutdown()

		for i := 0; i < b.N; i++ {
			ParallelRun(1, func(q *Var) Goal {
				return Eq(q, NewAtom(i))
			})
		}
	})
}
