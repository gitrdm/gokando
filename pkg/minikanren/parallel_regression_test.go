package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// TestParallelSearch_EnumerateAllCounts validates that enumerate-all problems
// return the exact expected number of solutions across multiple worker counts.
func TestParallelSearch_EnumerateAllCounts(t *testing.T) {
	if testing.Short() && !shouldRunHeavy() {
		t.Skip("skip enumerate-all regression counts in short mode")
	}

	cases := []struct {
		n      int
		expect int
	}{
		{4, 24},  // 4! permutations
		{5, 120}, // 5!
		{6, 720}, // 6! (still fast enough)
	}

	workerCounts := []int{1, 2, 4, 8}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("n=%d", tc.n), func(t *testing.T) {
			// Build AllDifferent(n) over domain 1..n
			model := NewModel()
			vars := model.NewVariables(tc.n, NewBitSetDomain(tc.n))
			alldiff, err := NewAllDifferent(vars)
			if err != nil {
				t.Fatalf("NewAllDifferent failed: %v", err)
			}
			model.AddConstraint(alldiff)

			for _, workers := range workerCounts {
				t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
					solver := NewSolver(model)
					ctx := context.Background()
					solutions, err := solver.SolveParallel(ctx, workers, 0)
					if err != nil {
						t.Fatalf("SolveParallel error: %v", err)
					}
					if len(solutions) != tc.expect {
						t.Fatalf("expected %d solutions for n=%d, got %d (workers=%d)", tc.expect, tc.n, len(solutions), workers)
					}
				})
			}
		})
	}
}
