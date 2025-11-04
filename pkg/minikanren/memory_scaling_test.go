package minikanren

import (
	"fmt"
	"runtime"
	"testing"
)

// TestMemoryScaling tests memory usage at different scales
func TestMemoryScaling(t *testing.T) {
	if testing.Short() && !shouldRunHeavy() {
		t.Skip("Skipping memory scaling test in short mode")
	}

	// Test different iteration counts to understand scaling
	testCases := []struct {
		iterations int
		expectMB   int
	}{
		{10, 5},     // 10 iterations should use ~5MB
		{100, 20},   // 100 iterations should use ~20MB
		{1000, 100}, // 1000 iterations currently uses ~112MB
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d_iterations", tc.iterations), func(t *testing.T) {
			// Record initial memory
			runtime.GC()
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			var totalSolutions int
			for i := 0; i < tc.iterations; i++ {
				// Create variables and goals (smaller than main test)
				vars := make([]*Var, 10) // Reduced from 100
				for j := range vars {
					vars[j] = Fresh(fmt.Sprintf("var_%d_%d", i, j))
				}

				// Run goals that generate solutions
				goalResults := Run(5, func(q *Var) Goal { // Reduced from 10
					goals := make([]Goal, len(vars))
					for k, v := range vars {
						goals[k] = Eq(v, NewAtom(k+i))
					}
					return Disj(goals...)
				})

				totalSolutions += len(goalResults)

				// Periodically force GC
				if i%50 == 0 {
					runtime.GC()
				}
			}

			// Force final garbage collection
			runtime.GC()
			runtime.GC()

			// Record final memory
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			allocMB := m2.Alloc / 1024 / 1024
			t.Logf("%d iterations: %d MB allocated, %d solutions", tc.iterations, allocMB, totalSolutions)

			if allocMB > uint64(tc.expectMB) {
				t.Logf("Memory usage higher than expected: %d MB > %d MB threshold", allocMB, tc.expectMB)
			}
		})
	}
}
