package minikanren

import (
	"context"
	"testing"
)

func TestFDSolver_Integration(t *testing.T) {
	// Create FD solver
	fdSolver := NewFDSolver("test-fd", "Test FD Solver", 9, 1, DefaultSolverConfig())

	// Test basic properties
	if fdSolver.ID() != "test-fd" {
		t.Errorf("expected ID 'test-fd', got '%s'", fdSolver.ID())
	}
	if fdSolver.Name() != "Test FD Solver" {
		t.Errorf("expected name 'Test FD Solver', got '%s'", fdSolver.Name())
	}

	caps := fdSolver.Capabilities()
	expectedCaps := []string{
		"FDAllDifferentConstraint",
		"FDOffsetConstraint",
		"FDInequalityConstraint",
		"FDCustomConstraint",
		"TypeConstraint",
	}
	if len(caps) != len(expectedCaps) {
		t.Errorf("expected %d capabilities, got %d", len(expectedCaps), len(caps))
	}

	// Test CanHandle for FD constraints
	fdConstraint := NewFDAllDifferentConstraint([]*Var{Fresh("x"), Fresh("y")})
	if !fdSolver.CanHandle(fdConstraint) {
		t.Error("FD solver should handle FDAllDifferentConstraint")
	}

	// Test CanHandle for non-FD constraints
	neqConstraint := NewDisequalityConstraint(Fresh("a"), Fresh("b"))
	if fdSolver.CanHandle(neqConstraint) {
		t.Error("FD solver should not handle DisequalityConstraint")
	}
}

func TestFDConstraintWrappers(t *testing.T) {
	// Test FDAllDifferentConstraint
	vars := []*Var{Fresh("x"), Fresh("y"), Fresh("z")}
	allDiff := NewFDAllDifferentConstraint(vars)

	if allDiff.ID() == "" {
		t.Error("FDAllDifferentConstraint should have non-empty ID")
	}
	if allDiff.IsLocal() {
		t.Error("FD constraints should not be local")
	}
	if len(allDiff.Variables()) != 3 {
		t.Errorf("expected 3 variables, got %d", len(allDiff.Variables()))
	}

	// Test Check returns pending (FD constraints are handled by solver)
	result := allDiff.Check(map[int64]Term{}) // Empty bindings for test
	if result != ConstraintPending {
		t.Errorf("expected ConstraintPending, got %v", result)
	}

	// Test Clone
	cloned := allDiff.Clone()
	if cloned.ID() != allDiff.ID() {
		t.Error("cloned constraint should have same ID")
	}

	// Test FDOffsetConstraint
	var1, var2 := Fresh("a"), Fresh("b")
	offset := NewFDOffsetConstraint(var1, var2, 5)

	if offset.ID() == "" {
		t.Error("FDOffsetConstraint should have non-empty ID")
	}
	if len(offset.Variables()) != 2 {
		t.Errorf("expected 2 variables, got %d", len(offset.Variables()))
	}

	// Test FDInequalityConstraint
	ineq := NewFDInequalityConstraint(var1, var2, IneqNotEqual)

	if ineq.ID() == "" {
		t.Error("FDInequalityConstraint should have non-empty ID")
	}
	if ineq.inequalityType != IneqNotEqual {
		t.Error("inequality type not set correctly")
	}
}

func TestConstraintManagerWithFDSolver(t *testing.T) {
	// Create constraint manager with FD solver
	cm := NewConstraintManagerWithFDSolver(9)
	if cm == nil {
		t.Fatal("NewConstraintManagerWithFDSolver returned nil")
	}

	// Verify FD solver is registered
	fdSolver, exists := cm.solvers["fd-solver"]
	if !exists {
		t.Fatal("FD solver not found in constraint manager")
	}
	if fdSolver == nil {
		t.Fatal("FD solver is nil")
	}

	// Verify FD constraint types are registered
	fdTypes := []string{
		"FDAllDifferentConstraint",
		"FDOffsetConstraint",
		"FDInequalityConstraint",
		"FDCustomConstraint",
	}

	for _, constraintType := range fdTypes {
		solverIDs, exists := cm.constraintTypes[constraintType]
		if !exists {
			t.Errorf("constraint type %s not registered", constraintType)
			continue
		}
		if len(solverIDs) == 0 {
			t.Errorf("no solvers registered for constraint type %s", constraintType)
		}
		// Verify FD solver is in the list
		found := false
		for _, id := range solverIDs {
			if id == "fd-solver" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FD solver not registered for constraint type %s", constraintType)
		}
	}
}

func TestFDSolver_SolveFDConstraint(t *testing.T) {
	cm := NewConstraintManagerWithFDSolver(9)

	// Create a simple FD all-different constraint
	vars := []*Var{Fresh("x"), Fresh("y")}
	constraint := NewFDAllDifferentConstraint(vars)
	store := NewLocalConstraintStore(nil)

	// Add the constraint to the store
	err := store.AddConstraint(constraint)
	if err != nil {
		t.Fatalf("AddConstraint failed: %v", err)
	}

	// Try to solve (this is a simplified test - real FD solving would be more complex)
	ctx := context.Background()
	result, err := cm.SolveConstraint(ctx, constraint, store)
	if err != nil {
		t.Fatalf("SolveConstraint failed: %v", err)
	}

	// For this basic test, we just verify the call succeeds
	// In a real scenario, the FD solver would perform actual constraint solving
	if result == nil {
		t.Error("expected non-nil result store")
	}
	// Note: The actual solving logic is simplified for this roadmap implementation
	// A complete implementation would verify that FD constraints are properly solved
}
