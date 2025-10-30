// Package minikanren provides integration tests for the constraint manager
// and pluggable solver system. These tests validate:
//
//   - Automatic solver registration and routing
//   - Constraint type mapping and capability matching
//   - Performance metrics collection and solver selection
//   - Fallback solver mechanisms
//   - Thread safety and concurrent operations
//   - Error handling and edge cases
package minikanren

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestSolver provides a test wrapper around concrete solvers for controlled testing
type TestSolver struct {
	Solver
	solveOverride     func(context.Context, Constraint, ConstraintStore) (ConstraintStore, error)
	canHandleOverride func(Constraint) bool
	callCount         int
	mu                sync.Mutex
}

// NewTestSolver creates a test solver wrapping a concrete solver
func NewTestSolver(baseSolver Solver) *TestSolver {
	return &TestSolver{
		Solver: baseSolver,
	}
}

// SetSolveOverride sets a custom solve function for testing
func (ts *TestSolver) SetSolveOverride(fn func(context.Context, Constraint, ConstraintStore) (ConstraintStore, error)) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.solveOverride = fn
}

// SetCanHandleOverride sets a custom CanHandle function for testing
func (ts *TestSolver) SetCanHandleOverride(fn func(Constraint) bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.canHandleOverride = fn
}

// GetCallCount returns the number of times Solve was called
func (ts *TestSolver) GetCallCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.callCount
}

// CanHandle implements Solver.CanHandle with optional override
func (ts *TestSolver) CanHandle(constraint Constraint) bool {
	ts.mu.Lock()
	fn := ts.canHandleOverride
	ts.mu.Unlock()

	if fn != nil {
		return fn(constraint)
	}
	return ts.Solver.CanHandle(constraint)
}

// Solve implements Solver.Solve with optional override
func (ts *TestSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	ts.mu.Lock()
	ts.callCount++
	fn := ts.solveOverride
	ts.mu.Unlock()

	if fn != nil {
		return fn(ctx, constraint, store)
	}
	return ts.Solver.Solve(ctx, constraint, store)
}

func TestConstraintManager_NewConstraintManager(t *testing.T) {
	cm := NewConstraintManager()
	if cm == nil {
		t.Fatal("NewConstraintManager returned nil")
	}
	if cm.solvers == nil {
		t.Error("solvers map not initialized")
	}
	if cm.constraintTypes == nil {
		t.Error("constraintTypes map not initialized")
	}
	if cm.metrics == nil {
		t.Error("metrics not initialized")
	}
	if cm.shutdown {
		t.Error("manager should not be shutdown initially")
	}
}

func TestConstraintManager_RegisterSolver(t *testing.T) {
	cm := NewConstraintManager()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"TestConstraint"}, 0, 10, time.Second))

	err := cm.RegisterSolver(solver)
	if err != nil {
		t.Fatalf("RegisterSolver failed: %v", err)
	}

	if _, exists := cm.solvers["test-solver"]; !exists {
		t.Error("solver not registered")
	}

	if _, exists := cm.metrics.solverStats["test-solver"]; !exists {
		t.Error("solver metrics not initialized")
	}

	// Test duplicate registration
	err = cm.RegisterSolver(solver)
	if err == nil {
		t.Error("expected error for duplicate solver registration")
	}
}

func TestConstraintManager_UnregisterSolver(t *testing.T) {
	cm := NewConstraintManager()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"TestConstraint"}, 0, 10, time.Second))

	// Register first
	err := cm.RegisterSolver(solver)
	if err != nil {
		t.Fatalf("RegisterSolver failed: %v", err)
	}

	// Unregister
	err = cm.UnregisterSolver("test-solver")
	if err != nil {
		t.Fatalf("UnregisterSolver failed: %v", err)
	}

	if _, exists := cm.solvers["test-solver"]; exists {
		t.Error("solver not unregistered")
	}

	if _, exists := cm.metrics.solverStats["test-solver"]; exists {
		t.Error("solver metrics not cleaned up")
	}

	// Test unregistering non-existent solver
	err = cm.UnregisterSolver("non-existent")
	if err == nil {
		t.Error("expected error for unregistering non-existent solver")
	}
}

func TestConstraintManager_RegisterConstraintType(t *testing.T) {
	cm := NewConstraintManager()

	// Register solvers first
	solver1 := NewTestSolver(NewBacktrackingSolver("solver1", "Solver 1", []string{"DisequalityConstraint"}, 0, 10, time.Second))
	solver2 := NewTestSolver(NewPropagationSolver("solver2", "Solver 2", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	cm.RegisterSolver(solver1)
	cm.RegisterSolver(solver2)

	// Register constraint type
	err := cm.RegisterConstraintType("DisequalityConstraint", []string{"solver1", "solver2"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed: %v", err)
	}

	solvers, exists := cm.constraintTypes["DisequalityConstraint"]
	if !exists {
		t.Error("constraint type not registered")
	}
	if len(solvers) != 2 {
		t.Errorf("expected 2 solvers, got %d", len(solvers))
	}

	// Test registering with non-existent solver
	err = cm.RegisterConstraintType("AnotherConstraint", []string{"non-existent"})
	if err == nil {
		t.Error("expected error for non-existent solver")
	}
}

func TestConstraintManager_SetFallbackSolvers(t *testing.T) {
	cm := NewConstraintManager()

	solver1 := NewTestSolver(NewBacktrackingSolver("fallback1", "Fallback 1", []string{}, 0, 10, time.Second))
	solver2 := NewTestSolver(NewPropagationSolver("fallback2", "Fallback 2", []string{}, 0, 10, time.Second))

	cm.RegisterSolver(solver1)
	cm.RegisterSolver(solver2)

	err := cm.SetFallbackSolvers([]Solver{solver1, solver2})
	if err != nil {
		t.Fatalf("SetFallbackSolvers failed: %v", err)
	}

	if len(cm.fallbackSolvers) != 2 {
		t.Errorf("expected 2 fallback solvers, got %d", len(cm.fallbackSolvers))
	}

	// Test setting fallback with non-existent solver
	fakeSolver := NewTestSolver(NewBacktrackingSolver("fake", "Fake", []string{}, 0, 10, time.Second))
	err = cm.SetFallbackSolvers([]Solver{fakeSolver})
	if err == nil {
		t.Error("expected error for unregistered fallback solver")
	}
}

func TestConstraintManager_SolveConstraint(t *testing.T) {
	cm := NewConstraintManager()
	store := NewLocalConstraintStore(nil)
	constraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))

	// Register solver
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"DisequalityConstraint"}, 0, 10, time.Second))
	cm.RegisterSolver(solver)

	// Register constraint type
	cm.RegisterConstraintType("DisequalityConstraint", []string{"test-solver"})

	// Solve constraint
	ctx := context.Background()
	result, err := cm.SolveConstraint(ctx, constraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result store")
	}

	if solver.GetCallCount() != 1 {
		t.Errorf("expected solver to be called once, got %d", solver.GetCallCount())
	}
}

func TestConstraintManager_SolverSelection(t *testing.T) {
	cm := NewConstraintManager()

	// Create solvers with different priorities
	highPriority := NewTestSolver(NewBacktrackingSolver("high", "High Priority", []string{"DisequalityConstraint"}, 10, 10, time.Second))
	lowPriority := NewTestSolver(NewPropagationSolver("low", "Low Priority", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	cm.RegisterSolver(highPriority)
	cm.RegisterSolver(lowPriority)

	// Register constraint type with both solvers
	cm.RegisterConstraintType("DisequalityConstraint", []string{"high", "low"})

	constraint := NewDisequalityConstraint(Fresh("a"), Fresh("b"))
	store := NewLocalConstraintStore(nil)

	ctx := context.Background()
	_, err := cm.SolveConstraint(ctx, constraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}

	// High priority solver should be selected
	if highPriority.GetCallCount() != 1 {
		t.Errorf("expected high priority solver to be called, got %d calls", highPriority.GetCallCount())
	}
	if lowPriority.GetCallCount() != 0 {
		t.Errorf("expected low priority solver not to be called, got %d calls", lowPriority.GetCallCount())
	}
}

func TestConstraintManager_FallbackSolvers(t *testing.T) {
	cm := NewConstraintManager()

	// Create solvers - primary will fail, fallback will succeed
	primary := NewTestSolver(NewBacktrackingSolver("primary", "Primary", []string{"DisequalityConstraint"}, 0, 10, time.Second))
	primary.SetSolveOverride(func(ctx context.Context, c Constraint, store ConstraintStore) (ConstraintStore, error) {
		return nil, errors.New("primary solver failed")
	})

	fallback := NewTestSolver(NewPropagationSolver("fallback", "Fallback", []string{}, 0, 10, time.Second))
	fallback.SetSolveOverride(func(ctx context.Context, c Constraint, store ConstraintStore) (ConstraintStore, error) {
		return store, nil // Success
	})

	cm.RegisterSolver(primary)
	cm.RegisterSolver(fallback)

	// Register constraint type
	cm.RegisterConstraintType("DisequalityConstraint", []string{"primary"})

	// Set fallback
	cm.SetFallbackSolvers([]Solver{fallback})

	constraint := NewDisequalityConstraint(Fresh("p"), Fresh("q"))
	store := NewLocalConstraintStore(nil)

	ctx := context.Background()
	result, err := cm.SolveConstraint(ctx, constraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint should succeed with fallback: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result store")
	}

	if primary.GetCallCount() != 1 {
		t.Errorf("expected primary solver to be called, got %d", primary.GetCallCount())
	}
	if fallback.GetCallCount() != 1 {
		t.Errorf("expected fallback solver to be called, got %d", fallback.GetCallCount())
	}
}

func TestConstraintManager_Metrics(t *testing.T) {
	cm := NewConstraintManager()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	cm.RegisterSolver(solver)
	cm.RegisterConstraintType("DisequalityConstraint", []string{"test-solver"})

	constraint := NewDisequalityConstraint(Fresh("m"), Fresh("n"))
	store := NewLocalConstraintStore(nil)

	ctx := context.Background()
	_, err := cm.SolveConstraint(ctx, constraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}

	metrics := cm.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics returned nil")
	}

	if metrics.totalConstraintsProcessed != 1 {
		t.Errorf("expected 1 total constraint processed, got %d", metrics.totalConstraintsProcessed)
	}

	solverStats, exists := metrics.solverStats["test-solver"]
	if !exists {
		t.Error("solver stats not found")
	}
	if solverStats.constraintsProcessed != 1 {
		t.Errorf("expected 1 constraint processed for solver, got %d", solverStats.constraintsProcessed)
	}
	if solverStats.successCount != 1 {
		t.Errorf("expected 1 success for solver, got %d", solverStats.successCount)
	}
}

func TestConstraintManager_Shutdown(t *testing.T) {
	cm := NewConstraintManager()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	cm.RegisterSolver(solver)

	// Operations should work before shutdown
	err := cm.RegisterConstraintType("DisequalityConstraint", []string{"test-solver"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed before shutdown: %v", err)
	}

	cm.Shutdown()

	if !cm.IsShutdown() {
		t.Error("manager should be shutdown")
	}

	// Operations should fail after shutdown
	err = cm.RegisterSolver(NewTestSolver(NewPropagationSolver("new-solver", "New Solver", []string{}, 0, 10, time.Second)))
	if err == nil {
		t.Error("expected error registering solver after shutdown")
	}

	err = cm.RegisterConstraintType("NewType", []string{"test-solver"})
	if err == nil {
		t.Error("expected error registering constraint type after shutdown")
	}
}

func TestSolverRegistry_RegisterSolver(t *testing.T) {
	registry := NewSolverRegistry()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	err := registry.RegisterSolver(solver)
	if err != nil {
		t.Fatalf("RegisterSolver failed: %v", err)
	}

	if _, exists := registry.solvers["test-solver"]; !exists {
		t.Error("solver not registered")
	}

	constraintType := "DisequalityConstraint"
	solvers := registry.GetSolversForType(constraintType)
	if len(solvers) != 1 {
		t.Errorf("expected 1 solver for type %s, got %d", constraintType, len(solvers))
	}

	// Test duplicate registration
	err = registry.RegisterSolver(solver)
	if err == nil {
		t.Error("expected error for duplicate solver registration")
	}
}

func TestSolverRegistry_UnregisterSolver(t *testing.T) {
	registry := NewSolverRegistry()
	solver := NewTestSolver(NewBacktrackingSolver("test-solver", "Test Solver", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	// Register first
	registry.RegisterSolver(solver)

	// Unregister
	err := registry.UnregisterSolver("test-solver")
	if err != nil {
		t.Fatalf("UnregisterSolver failed: %v", err)
	}

	if _, exists := registry.solvers["test-solver"]; exists {
		t.Error("solver not unregistered")
	}

	solvers := registry.GetSolversForType("DisequalityConstraint")
	if len(solvers) != 0 {
		t.Errorf("expected no solvers for type after unregister, got %d", len(solvers))
	}

	// Test unregistering non-existent solver
	err = registry.UnregisterSolver("non-existent")
	if err == nil {
		t.Error("expected error for unregistering non-existent solver")
	}
}

func TestSolverRegistry_FindBestSolver(t *testing.T) {
	registry := NewSolverRegistry()

	// Create solvers with different priorities
	highPriority := NewTestSolver(NewBacktrackingSolver("high", "High Priority", []string{"DisequalityConstraint"}, 10, 10, time.Second))
	lowPriority := NewTestSolver(NewPropagationSolver("low", "Low Priority", []string{"DisequalityConstraint"}, 0, 10, time.Second))

	registry.RegisterSolver(highPriority)
	registry.RegisterSolver(lowPriority)

	constraint := NewDisequalityConstraint(Fresh("h"), Fresh("l"))

	bestSolver, err := registry.FindBestSolver(constraint)
	if err != nil {
		t.Fatalf("FindBestSolver failed: %v", err)
	}

	if bestSolver.ID() != "high" {
		t.Errorf("expected high priority solver, got %s", bestSolver.ID())
	}

	// Test with constraint that no solver can handle
	highPriority.SetCanHandleOverride(func(Constraint) bool { return false })
	lowPriority.SetCanHandleOverride(func(Constraint) bool { return false })

	_, err = registry.FindBestSolver(constraint)
	if err == nil {
		t.Error("expected error when no solver can handle constraint")
	}
}

func TestBaseSolver(t *testing.T) {
	solver := NewBaseSolver("test-base", "Test Base Solver", []string{"DisequalityConstraint"}, 5)

	if solver.ID() != "test-base" {
		t.Errorf("expected ID 'test-base', got '%s'", solver.ID())
	}
	if solver.Name() != "Test Base Solver" {
		t.Errorf("expected name 'Test Base Solver', got '%s'", solver.Name())
	}
	caps := solver.Capabilities()
	if len(caps) != 1 || caps[0] != "DisequalityConstraint" {
		t.Errorf("expected capabilities ['DisequalityConstraint'], got %v", caps)
	}
	if solver.Priority() != 5 {
		t.Errorf("expected priority 5, got %d", solver.Priority())
	}

	constraint := NewDisequalityConstraint(Fresh("b"), Fresh("c"))
	if !solver.CanHandle(constraint) {
		t.Error("expected solver to handle DisequalityConstraint")
	}

	// Test Solve (should return error)
	store := NewLocalConstraintStore(nil)
	ctx := context.Background()
	_, err := solver.Solve(ctx, constraint, store)
	if err == nil {
		t.Error("expected error from base solver Solve method")
	}
}

func TestConstraintManagerConcurrentAccess(t *testing.T) {
	cm := NewConstraintManager()
	store := NewLocalConstraintStore(nil)

	// Register multiple solvers
	for i := 0; i < 5; i++ {
		solver := NewTestSolver(NewBacktrackingSolver(fmt.Sprintf("solver-%d", i), fmt.Sprintf("Solver %d", i), []string{"DisequalityConstraint"}, i, 10, time.Second))
		cm.RegisterSolver(solver)
		cm.RegisterConstraintType("DisequalityConstraint", []string{fmt.Sprintf("solver-%d", i)})
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent constraint solving
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				constraint := NewDisequalityConstraint(Fresh(fmt.Sprintf("x%d", id)), Fresh(fmt.Sprintf("y%d", j)))
				ctx := context.Background()
				_, err := cm.SolveConstraint(ctx, constraint, store)
				if err != nil {
					t.Errorf("concurrent SolveConstraint failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Check that metrics were collected correctly
	metrics := cm.GetMetrics()
	if metrics.totalConstraintsProcessed != int64(numGoroutines*numOperations) {
		t.Errorf("expected %d total constraints processed, got %d",
			numGoroutines*numOperations, metrics.totalConstraintsProcessed)
	}
}
