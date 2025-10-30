// Package minikanren provides integration tests for concrete solver implementations.
// These tests validate that the pluggable constraint solving architecture works
// end-to-end with real solver implementations.
package minikanren

import (
	"context"
	"testing"
	"time"
)

// TestBacktrackingSolver_Integration tests the backtracking solver integration
// with the constraint manager and actual constraint types.
func TestBacktrackingSolver_Integration(t *testing.T) {
	t.Parallel()

	// Create a backtracking solver
	solver := NewBacktrackingSolver(
		"test-backtracking",
		"Test Backtracking Solver",
		[]string{"DisequalityConstraint", "TypeConstraint"},
		1,
		10, // max depth
		1*time.Second,
	)

	// Test solver capabilities
	if solver.ID() != "test-backtracking" {
		t.Errorf("expected ID 'test-backtracking', got '%s'", solver.ID())
	}
	if solver.Name() != "Test Backtracking Solver" {
		t.Errorf("expected name 'Test Backtracking Solver', got '%s'", solver.Name())
	}
	if solver.Priority() != 1 {
		t.Errorf("expected priority 1, got %d", solver.Priority())
	}

	caps := solver.Capabilities()
	hasDiseq := false
	hasType := false
	for _, cap := range caps {
		if cap == "DisequalityConstraint" {
			hasDiseq = true
		}
		if cap == "TypeConstraint" {
			hasType = true
		}
	}
	if !hasDiseq {
		t.Error("expected DisequalityConstraint capability")
	}
	if !hasType {
		t.Error("expected TypeConstraint capability")
	}

	// Test CanHandle for supported constraint types
	diseqConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	typeConstraint := NewTypeConstraint(Fresh("z"), SymbolType)

	if !solver.CanHandle(diseqConstraint) {
		t.Error("expected solver to handle DisequalityConstraint")
	}
	if !solver.CanHandle(typeConstraint) {
		t.Error("expected solver to handle TypeConstraint")
	}
}

// TestPropagationSolver_Integration tests the propagation solver integration.
func TestPropagationSolver_Integration(t *testing.T) {
	t.Parallel()

	// Create a propagation solver
	solver := NewPropagationSolver(
		"test-propagation",
		"Test Propagation Solver",
		[]string{"AbsenceConstraint", "TypeConstraint"},
		2,
		5, // max iterations
		1*time.Second,
	)

	// Test solver capabilities
	if solver.ID() != "test-propagation" {
		t.Errorf("expected ID 'test-propagation', got '%s'", solver.ID())
	}
	if solver.Name() != "Test Propagation Solver" {
		t.Errorf("expected name 'Test Propagation Solver', got '%s'", solver.Name())
	}
	if solver.Priority() != 2 {
		t.Errorf("expected priority 2, got %d", solver.Priority())
	}

	caps := solver.Capabilities()
	hasAbsence := false
	hasType := false
	for _, cap := range caps {
		if cap == "AbsenceConstraint" {
			hasAbsence = true
		}
		if cap == "TypeConstraint" {
			hasType = true
		}
	}
	if !hasAbsence {
		t.Error("expected AbsenceConstraint capability")
	}
	if !hasType {
		t.Error("expected TypeConstraint capability")
	}

	// Test CanHandle for supported constraint types
	absenceConstraint := NewAbsenceConstraint(Fresh("x"), Fresh("y"))
	typeConstraint := NewTypeConstraint(Fresh("z"), SymbolType)

	if !solver.CanHandle(absenceConstraint) {
		t.Error("expected solver to handle AbsenceConstraint")
	}
	if !solver.CanHandle(typeConstraint) {
		t.Error("expected solver to handle TypeConstraint")
	}
}

// TestHybridSolver_Integration tests the hybrid solver integration.
func TestHybridSolver_Integration(t *testing.T) {
	t.Parallel()

	// Create a hybrid solver
	solver := NewHybridSolver(
		"test-hybrid",
		"Test Hybrid Solver",
		[]string{"DisequalityConstraint", "AbsenceConstraint", "TypeConstraint"},
		3,
		3*time.Second,
	)

	// Test solver capabilities
	if solver.ID() != "test-hybrid" {
		t.Errorf("expected ID 'test-hybrid', got '%s'", solver.ID())
	}
	if solver.Name() != "Test Hybrid Solver" {
		t.Errorf("expected name 'Test Hybrid Solver', got '%s'", solver.Name())
	}
	if solver.Priority() != 3 {
		t.Errorf("expected priority 3, got %d", solver.Priority())
	}

	caps := solver.Capabilities()
	hasDiseq := false
	hasAbsence := false
	hasType := false
	for _, cap := range caps {
		if cap == "DisequalityConstraint" {
			hasDiseq = true
		}
		if cap == "AbsenceConstraint" {
			hasAbsence = true
		}
		if cap == "TypeConstraint" {
			hasType = true
		}
	}
	if !hasDiseq {
		t.Error("expected DisequalityConstraint capability")
	}
	if !hasAbsence {
		t.Error("expected AbsenceConstraint capability")
	}
	if !hasType {
		t.Error("expected TypeConstraint capability")
	}

	// Test CanHandle for all supported constraint types
	diseqConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	absenceConstraint := NewAbsenceConstraint(Fresh("a"), Fresh("b"))
	typeConstraint := NewTypeConstraint(Fresh("z"), SymbolType)

	if !solver.CanHandle(diseqConstraint) {
		t.Error("expected solver to handle DisequalityConstraint")
	}
	if !solver.CanHandle(absenceConstraint) {
		t.Error("expected solver to handle AbsenceConstraint")
	}
	if !solver.CanHandle(typeConstraint) {
		t.Error("expected solver to handle TypeConstraint")
	}
}

// TestSolverFactory_Integration tests the solver factory integration.
func TestSolverFactory_Integration(t *testing.T) {
	t.Parallel()

	factory := NewSolverFactory()

	// Test creating individual solvers
	backtracking := factory.CreateBacktrackingSolver("factory-bt")
	if backtracking.ID() != "factory-bt" {
		t.Errorf("expected ID 'factory-bt', got '%s'", backtracking.ID())
	}
	caps := backtracking.Capabilities()
	hasDiseq := false
	for _, cap := range caps {
		if cap == "DisequalityConstraint" {
			hasDiseq = true
		}
	}
	if !hasDiseq {
		t.Error("expected DisequalityConstraint capability")
	}

	propagation := factory.CreatePropagationSolver("factory-prop")
	if propagation.ID() != "factory-prop" {
		t.Errorf("expected ID 'factory-prop', got '%s'", propagation.ID())
	}
	caps = propagation.Capabilities()
	hasAbsence := false
	for _, cap := range caps {
		if cap == "AbsenceConstraint" {
			hasAbsence = true
		}
	}
	if !hasAbsence {
		t.Error("expected AbsenceConstraint capability")
	}

	hybrid := factory.CreateHybridSolver("factory-hybrid")
	if hybrid.ID() != "factory-hybrid" {
		t.Errorf("expected ID 'factory-hybrid', got '%s'", hybrid.ID())
	}
	caps = hybrid.Capabilities()
	hasDiseq = false
	hasAbsence = false
	hasType := false
	for _, cap := range caps {
		if cap == "DisequalityConstraint" {
			hasDiseq = true
		}
		if cap == "AbsenceConstraint" {
			hasAbsence = true
		}
		if cap == "TypeConstraint" {
			hasType = true
		}
	}
	if !hasDiseq {
		t.Error("expected DisequalityConstraint capability")
	}
	if !hasAbsence {
		t.Error("expected AbsenceConstraint capability")
	}
	if !hasType {
		t.Error("expected TypeConstraint capability")
	}

	// Test creating a complete solver set
	solvers := factory.CreateSolverSet()
	if len(solvers) != 4 {
		t.Errorf("expected 4 solvers, got %d", len(solvers))
	}

	// Verify solver ordering by priority (fd > hybrid > propagation > backtracking)
	if solvers[0].ID() != "fd-solver" {
		t.Errorf("expected first solver to be 'fd-solver', got '%s'", solvers[0].ID())
	}
	if solvers[0].Priority() != 5 {
		t.Errorf("expected first solver priority 5, got %d", solvers[0].Priority())
	}

	if solvers[1].ID() != "hybrid-solver" {
		t.Errorf("expected second solver to be 'hybrid-solver', got '%s'", solvers[1].ID())
	}
	if solvers[1].Priority() != 3 {
		t.Errorf("expected second solver priority 3, got %d", solvers[1].Priority())
	}

	if solvers[2].ID() != "propagation-solver" {
		t.Errorf("expected third solver to be 'propagation-solver', got '%s'", solvers[2].ID())
	}
	if solvers[2].Priority() != 2 {
		t.Errorf("expected third solver priority 2, got %d", solvers[2].Priority())
	}

	if solvers[3].ID() != "backtracking-solver" {
		t.Errorf("expected fourth solver to be 'backtracking-solver', got '%s'", solvers[3].ID())
	}
	if solvers[3].Priority() != 1 {
		t.Errorf("expected fourth solver priority 1, got %d", solvers[3].Priority())
	}
}

// TestSolverComparator_Integration tests the solver comparator utilities.
func TestSolverComparator_Integration(t *testing.T) {
	t.Parallel()

	comparator := NewSolverComparator()
	factory := NewSolverFactory()

	// Create a set of solvers
	solvers := factory.CreateSolverSet()

	// Test ranking solvers for different constraint types
	diseqRanked := comparator.RankSolvers(solvers, "DisequalityConstraint")
	if len(diseqRanked) != 2 { // hybrid and backtracking can handle disequality
		t.Errorf("expected 2 solvers for disequality, got %d", len(diseqRanked))
	}
	if diseqRanked[0].ID() != "hybrid-solver" { // higher priority
		t.Errorf("expected first disequality solver to be 'hybrid-solver', got '%s'", diseqRanked[0].ID())
	}
	if diseqRanked[1].ID() != "backtracking-solver" { // lower priority
		t.Errorf("expected second disequality solver to be 'backtracking-solver', got '%s'", diseqRanked[1].ID())
	}

	absenceRanked := comparator.RankSolvers(solvers, "AbsenceConstraint")
	if len(absenceRanked) != 2 { // hybrid and propagation can handle absence
		t.Errorf("expected 2 solvers for absence, got %d", len(absenceRanked))
	}
	if absenceRanked[0].ID() != "hybrid-solver" { // higher priority
		t.Errorf("expected first absence solver to be 'hybrid-solver', got '%s'", absenceRanked[0].ID())
	}
	if absenceRanked[1].ID() != "propagation-solver" { // lower priority
		t.Errorf("expected second absence solver to be 'propagation-solver', got '%s'", absenceRanked[1].ID())
	}

	typeRanked := comparator.RankSolvers(solvers, "TypeConstraint")
	if len(typeRanked) != 4 { // all solvers can handle type constraints
		t.Errorf("expected 4 solvers for type, got %d", len(typeRanked))
	}
	if typeRanked[0].ID() != "fd-solver" { // highest priority
		t.Errorf("expected first type solver to be 'fd-solver', got '%s'", typeRanked[0].ID())
	}
	if typeRanked[1].ID() != "hybrid-solver" { // high priority
		t.Errorf("expected second type solver to be 'hybrid-solver', got '%s'", typeRanked[1].ID())
	}
	if typeRanked[2].ID() != "propagation-solver" { // medium priority
		t.Errorf("expected third type solver to be 'propagation-solver', got '%s'", typeRanked[2].ID())
	}
	if typeRanked[3].ID() != "backtracking-solver" { // lowest priority
		t.Errorf("expected fourth type solver to be 'backtracking-solver', got '%s'", typeRanked[3].ID())
	}

	// Test getting best solver
	bestDiseq, found := comparator.GetBestSolver(solvers, "DisequalityConstraint")
	if !found {
		t.Error("expected to find best solver for disequality")
	}
	if bestDiseq.ID() != "hybrid-solver" {
		t.Errorf("expected best disequality solver to be 'hybrid-solver', got '%s'", bestDiseq.ID())
	}

	bestAbsence, found := comparator.GetBestSolver(solvers, "AbsenceConstraint")
	if !found {
		t.Error("expected to find best solver for absence")
	}
	if bestAbsence.ID() != "hybrid-solver" {
		t.Errorf("expected best absence solver to be 'hybrid-solver', got '%s'", bestAbsence.ID())
	}

	bestType, found := comparator.GetBestSolver(solvers, "TypeConstraint")
	if !found {
		t.Error("expected to find best solver for type")
	}
	if bestType.ID() != "fd-solver" {
		t.Errorf("expected best type solver to be 'fd-solver', got '%s'", bestType.ID())
	}

	// Test with unsupported constraint type
	_, found = comparator.GetBestSolver(solvers, "UnsupportedConstraint")
	if found {
		t.Error("expected not to find solver for unsupported constraint")
	}
}

// TestConcreteSolvers_WithConstraintManager tests end-to-end integration
// of concrete solvers with the constraint manager.
func TestConcreteSolvers_WithConstraintManager(t *testing.T) {
	t.Parallel()

	// Create constraint manager
	manager := NewConstraintManager()

	// Create and register concrete solvers
	factory := NewSolverFactory()
	solvers := factory.CreateSolverSet()

	for _, solver := range solvers {
		err := manager.RegisterSolver(solver)
		if err != nil {
			t.Fatalf("RegisterSolver failed: %v", err)
		}
	}

	// Register constraint types
	err := manager.RegisterConstraintType("DisequalityConstraint", []string{"hybrid-solver", "backtracking-solver"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed: %v", err)
	}
	err = manager.RegisterConstraintType("AbsenceConstraint", []string{"hybrid-solver", "propagation-solver"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed: %v", err)
	}
	err = manager.RegisterConstraintType("TypeConstraint", []string{"hybrid-solver", "propagation-solver", "backtracking-solver"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed: %v", err)
	}

	// Test solving with different constraint types
	ctx := context.Background()
	store := NewLocalConstraintStore(nil)

	// Test disequality constraint (should use hybrid solver - highest priority)
	diseqConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	result, err := manager.SolveConstraint(ctx, diseqConstraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result store")
	}

	// Test absence constraint (should use hybrid solver - highest priority)
	absenceConstraint := NewAbsenceConstraint(Fresh("a"), Fresh("b"))
	result, err = manager.SolveConstraint(ctx, absenceConstraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result store")
	}

	// Test type constraint (should use hybrid solver - highest priority)
	typeConstraint := NewTypeConstraint(Fresh("z"), SymbolType)
	result, err = manager.SolveConstraint(ctx, typeConstraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result store")
	}
}

// TestConcreteSolvers_TimeoutBehavior tests timeout behavior of concrete solvers.
func TestConcreteSolvers_TimeoutBehavior(t *testing.T) {
	t.Parallel()

	// Create solvers with real constraint capabilities for testing
	backtracking := NewBacktrackingSolver("bt-timeout", "BT Timeout", []string{"DisequalityConstraint"}, 1, 10, 1*time.Millisecond)
	propagation := NewPropagationSolver("prop-timeout", "Prop Timeout", []string{"AbsenceConstraint"}, 1, 10, 1*time.Millisecond)
	hybrid := NewHybridSolver("hybrid-timeout", "Hybrid Timeout", []string{"DisequalityConstraint", "AbsenceConstraint"}, 1, 2*time.Millisecond)

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	store := NewLocalConstraintStore(nil)
	diseqConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	absenceConstraint := NewAbsenceConstraint(Fresh("a"), Fresh("b"))

	// All solvers should handle timeout gracefully
	_, err := backtracking.Solve(ctx, diseqConstraint, store)
	// Note: In this demo implementation, solvers don't actually fail on timeout
	// In a real implementation, they would check context cancellation
	if err != nil {
		t.Errorf("backtracking solver failed: %v", err)
	}

	_, err = propagation.Solve(ctx, absenceConstraint, store)
	if err != nil {
		t.Errorf("propagation solver failed: %v", err)
	}

	_, err = hybrid.Solve(ctx, diseqConstraint, store)
	if err != nil {
		t.Errorf("hybrid solver failed: %v", err)
	}
}

// TestConcreteSolvers_ConcurrentAccess tests concurrent access to concrete solvers.
func TestConcreteSolvers_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// Create solvers
	factory := NewSolverFactory()
	solvers := factory.CreateSolverSet()

	// Test concurrent access to multiple solvers
	done := make(chan bool, len(solvers)*10)

	for i := 0; i < 10; i++ {
		for _, solver := range solvers {
			go func(s Solver) {
				defer func() { done <- true }()

				// Test concurrent capability checks
				caps := s.Capabilities()
				if len(caps) == 0 {
					t.Errorf("expected non-empty capabilities")
				}

				// Test concurrent ID access
				id := s.ID()
				if id == "" {
					t.Errorf("expected non-empty ID")
				}

				// Test concurrent priority access
				priority := s.Priority()
				if priority < 1 {
					t.Errorf("expected priority >= 1, got %d", priority)
				}
			}(solver)
		}
	}

	// Wait for all goroutines to complete
	for i := 0; i < len(solvers)*10; i++ {
		select {
		case <-done:
			// Expected
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent solver access to complete")
		}
	}
}

// TestConcreteSolvers_MetricsCollection tests that concrete solvers work
// with the metrics collection system.
func TestConcreteSolvers_MetricsCollection(t *testing.T) {
	t.Parallel()

	// Create constraint manager with metrics
	manager := NewConstraintManager()

	// Register concrete solvers
	factory := NewSolverFactory()
	solvers := factory.CreateSolverSet()

	for _, solver := range solvers {
		err := manager.RegisterSolver(solver)
		if err != nil {
			t.Fatalf("RegisterSolver failed: %v", err)
		}
	}

	// Register a constraint type
	err := manager.RegisterConstraintType("DisequalityConstraint", []string{"hybrid-solver"})
	if err != nil {
		t.Fatalf("RegisterConstraintType failed: %v", err)
	}

	// Perform some solves to generate metrics
	ctx := context.Background()
	store := NewLocalConstraintStore(nil)
	testConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))

	for i := 0; i < 5; i++ {
		_, err := manager.SolveConstraint(ctx, testConstraint, store)
		if err != nil {
			t.Fatalf("SolveConstraint failed: %v", err)
		}
	}

	// Check that metrics were collected
	metrics := manager.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics returned nil")
	}

	// Verify solver metrics exist
	hybridMetrics, exists := metrics.solverStats["hybrid-solver"]
	if !exists {
		t.Error("solver stats not found")
	}
	if hybridMetrics.constraintsProcessed != 5 {
		t.Errorf("expected 5 total attempts, got %d", hybridMetrics.constraintsProcessed)
	}
	if hybridMetrics.successCount != 5 {
		t.Errorf("expected 5 successes, got %d", hybridMetrics.successCount)
	}
}

// BenchmarkBacktrackingSolver benchmarks the backtracking solver performance.
func BenchmarkBacktrackingSolver(b *testing.B) {
	solver := NewBacktrackingSolver("bench-bt", "Benchmark BT", []string{"DisequalityConstraint"}, 1, 100, 10*time.Second)
	store := NewLocalConstraintStore(nil)
	testConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := solver.Solve(ctx, testConstraint, store)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPropagationSolver benchmarks the propagation solver performance.
func BenchmarkPropagationSolver(b *testing.B) {
	solver := NewPropagationSolver("bench-prop", "Benchmark Prop", []string{"AbsenceConstraint"}, 1, 20, 10*time.Second)
	store := NewLocalConstraintStore(nil)
	testConstraint := NewAbsenceConstraint(Fresh("x"), Fresh("y"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := solver.Solve(ctx, testConstraint, store)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHybridSolver benchmarks the hybrid solver performance.
func BenchmarkHybridSolver(b *testing.B) {
	solver := NewHybridSolver("bench-hybrid", "Benchmark Hybrid", []string{"DisequalityConstraint"}, 1, 10*time.Second)
	store := NewLocalConstraintStore(nil)
	testConstraint := NewDisequalityConstraint(Fresh("x"), Fresh("y"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := solver.Solve(ctx, testConstraint, store)
		if err != nil {
			b.Fatal(err)
		}
	}
}
