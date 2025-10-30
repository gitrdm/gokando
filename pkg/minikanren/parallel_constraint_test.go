package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestParallelConstraintPropagator(t *testing.T) {
	store := NewFDStoreWithDomain(9)

	// Create some variables
	vars := make([]*FDVar, 3)
	for i := range vars {
		vars[i] = store.NewVar()
	}

	// Add AllDifferent constraint
	store.AddAllDifferent(vars)

	// Create parallel propagator
	pcp := NewParallelConstraintPropagator(store, 2)
	defer pcp.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Submit some propagation tasks
	if err := pcp.SubmitTask(vars[0].ID, taskSingletonPropagation); err != nil {
		t.Errorf("Failed to submit propagation task: %v", err)
	}

	if err := pcp.SubmitTask(vars[1].ID, taskOffsetPropagation); err != nil {
		t.Errorf("Failed to submit offset propagation task: %v", err)
	}

	// Wait for completion
	if err := pcp.WaitForCompletion(ctx); err != nil {
		t.Errorf("Propagation failed: %v", err)
	}
}

func TestParallelConstraintPropagatorWithDependencies(t *testing.T) {
	store := NewFDStoreWithDomain(9)

	// Create variables for a chain constraint: A + 1 = B, B + 1 = C
	A := store.NewVar()
	B := store.NewVar()
	C := store.NewVar()

	// Add offset constraints
	store.AddOffsetLink(A, 1, B)
	store.AddOffsetLink(B, 1, C)

	// Create parallel propagator
	pcp := NewParallelConstraintPropagator(store, 3)
	defer pcp.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Assign A = 1, which should propagate through the chain
	if err := store.Assign(A, 1); err != nil {
		t.Fatalf("Failed to assign A=1: %v", err)
	}

	// Submit propagation tasks for the dependency chain
	if err := pcp.SubmitTask(A.ID, taskOffsetPropagation); err != nil {
		t.Errorf("Failed to submit A propagation: %v", err)
	}

	// Wait for completion
	if err := pcp.WaitForCompletion(ctx); err != nil {
		t.Errorf("Propagation failed: %v", err)
	}

	// Check that propagation worked
	bDomain := store.GetDomain(B)
	if !bDomain.Has(2) {
		t.Errorf("Expected B to contain 2, domain: %+v", bDomain)
	}

	cDomain := store.GetDomain(C)
	if !cDomain.Has(3) {
		t.Errorf("Expected C to contain 3, domain: %+v", cDomain)
	}
}

func TestParallelExecutorWithStats(t *testing.T) {
	config := &ParallelConfig{
		MaxWorkers:           4,
		MinWorkers:           1,
		EnableDynamicScaling: true,
		EnableWorkStealing:   true,
	}

	executor := NewParallelExecutor(config)
	defer executor.Shutdown()

	// Check that we can get stats
	stats := executor.GetExecutionStats()
	if stats == nil {
		t.Error("Expected non-nil execution stats")
	}

	// Check that we can get deadlock detector
	dd := executor.GetDeadlockDetector()
	if dd == nil {
		t.Error("Expected non-nil deadlock detector")
	}

	// Check that we can get alert channel
	alerts := executor.GetDeadlockAlerts()
	if alerts == nil {
		t.Error("Expected non-nil alert channel")
	}
}

func BenchmarkParallelConstraintPropagation(b *testing.B) {
	store := NewFDStoreWithDomain(9)

	// Create a larger set of variables for benchmarking
	vars := make([]*FDVar, 20)
	for i := range vars {
		vars[i] = store.NewVar()
	}

	// Add AllDifferent constraint
	store.AddAllDifferent(vars)

	pcp := NewParallelConstraintPropagator(store, 4)
	defer pcp.Shutdown()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Submit propagation tasks
		for _, v := range vars {
			pcp.SubmitTask(v.ID, taskSingletonPropagation)
		}

		// Wait for completion
		if err := pcp.WaitForCompletion(ctx); err != nil {
			b.Fatalf("Propagation failed: %v", err)
		}
	}
}
