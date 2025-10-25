package minikanren

import (
	"fmt"
	"runtime"
	"testing"
)

// TestGoroutineExplosion tests how many goroutines are created
func TestGoroutineExplosion(t *testing.T) {
	// Baseline goroutines
	baseline := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baseline)

	// Simple test - 10 Eq operations in Disj
	vars := make([]*Var, 10)
	for j := range vars {
		vars[j] = Fresh(fmt.Sprintf("var_%d", j))
	}

	beforeTest := runtime.NumGoroutine()
	t.Logf("Before test goroutines: %d", beforeTest)

	// This should create: 10 Eq goals + 1 Disj coordinator = 11 goroutines minimum
	results := Run(1, func(q *Var) Goal {
		goals := make([]Goal, len(vars))
		for k, v := range vars {
			goals[k] = Eq(v, NewAtom(k))
		}
		return Disj(goals...)
	})

	afterTest := runtime.NumGoroutine()
	t.Logf("After test goroutines: %d", afterTest)
	t.Logf("Goroutines created: %d", afterTest-beforeTest)
	t.Logf("Results: %d", len(results))

	// Wait a bit for goroutines to finish
	runtime.GC()

	finalCount := runtime.NumGoroutine()
	t.Logf("Final goroutines: %d", finalCount)
	t.Logf("Goroutines still running: %d", finalCount-baseline)
}
