package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestNewSolver(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(5))

	solver := NewSolver(model)
	if solver == nil {
		t.Fatal("NewSolver() returned nil")
	}
	if solver.Model() != model {
		t.Error("Solver.Model() should return the same model instance")
	}
}

func TestNewSolverWithConfig(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(5))

	config := &SolverConfig{
		VariableHeuristic: HeuristicDom,
		ValueHeuristic:    ValueOrderDesc,
	}

	solver := NewSolverWithConfig(model, config)
	if solver == nil {
		t.Fatal("NewSolverWithConfig() returned nil")
	}

	// Test with nil config
	solver2 := NewSolverWithConfig(model, nil)
	if solver2 == nil {
		t.Fatal("NewSolverWithConfig(nil) returned nil")
	}
}

func TestSolver_GetDomain(t *testing.T) {
	model := NewModel()
	vars := model.NewVariables(3, NewBitSetDomain(10))

	solver := NewSolver(model)

	// Test with nil state (should return initial domains)
	domain := solver.GetDomain(nil, 0)
	if domain == nil {
		t.Fatal("GetDomain() returned nil")
	}
	if domain.Count() != 10 {
		t.Errorf("GetDomain() count = %d, want 10", domain.Count())
	}

	// Test with modified state
	newDomain := NewBitSetDomainFromValues(10, []int{5})
	state := solver.SetDomain(nil, 0, newDomain)
	retrieved := solver.GetDomain(state, 0)
	if retrieved.Count() != 1 {
		t.Errorf("GetDomain() after SetDomain count = %d, want 1", retrieved.Count())
	}
	if !retrieved.Has(5) {
		t.Error("GetDomain() should return modified domain")
	}

	// Test that other variables are unchanged
	domain1 := solver.GetDomain(state, 1)
	if domain1.Count() != 10 {
		t.Errorf("unmodified variable domain count = %d, want 10", domain1.Count())
	}

	_ = vars // Suppress unused warning
}

func TestSolver_SetDomain(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(10))

	solver := NewSolver(model)

	// Create state chain
	d1 := NewBitSetDomainFromValues(10, []int{5})
	state1 := solver.SetDomain(nil, 0, d1)

	if state1 == nil {
		t.Fatal("SetDomain() returned nil")
	}

	d2 := NewBitSetDomainFromValues(10, []int{7})
	state2 := solver.SetDomain(state1, 1, d2)

	// Verify state chain
	if solver.GetDomain(state2, 0).Count() != 1 {
		t.Error("state chain should preserve earlier modifications")
	}
	if solver.GetDomain(state2, 1).Count() != 1 {
		t.Error("state chain should include new modifications")
	}
}

func TestSolver_ReleaseState(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(10))

	solver := NewSolver(model)

	// Create and release states
	state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(10, []int{5}))
	solver.ReleaseState(state)

	// Should not crash
	solver.ReleaseState(nil)
}

func TestSolver_IsComplete(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(5))

	solver := NewSolver(model)

	// Empty state is not complete
	if solver.isComplete(nil) {
		t.Error("empty state should not be complete")
	}

	// Partially assigned state is not complete
	state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(5, []int{1}))
	if solver.isComplete(state) {
		t.Error("partially assigned state should not be complete")
	}

	// Fully assigned state is complete
	state = solver.SetDomain(state, 1, NewBitSetDomainFromValues(5, []int{2}))
	state = solver.SetDomain(state, 2, NewBitSetDomainFromValues(5, []int{3}))
	if !solver.isComplete(state) {
		t.Error("fully assigned state should be complete")
	}
}

func TestSolver_ExtractSolution(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(5))

	solver := NewSolver(model)

	// Create complete state
	state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(5, []int{1}))
	state = solver.SetDomain(state, 1, NewBitSetDomainFromValues(5, []int{2}))
	state = solver.SetDomain(state, 2, NewBitSetDomainFromValues(5, []int{3}))

	solution := solver.extractSolution(state)
	expected := []int{1, 2, 3}

	if len(solution) != len(expected) {
		t.Fatalf("solution length = %d, want %d", len(solution), len(expected))
	}

	for i, v := range expected {
		if solution[i] != v {
			t.Errorf("solution[%d] = %d, want %d", i, solution[i], v)
		}
	}
}

func TestSolver_SelectVariable(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(10))

	solver := NewSolver(model)

	// Select from empty state
	varID, values := solver.selectVariable(nil)
	if varID < 0 || varID > 2 {
		t.Errorf("selectVariable() varID = %d, want 0-2", varID)
	}
	if len(values) != 10 {
		t.Errorf("selectVariable() values count = %d, want 10", len(values))
	}

	// Select when one variable is assigned
	state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(10, []int{5}))
	varID, values = solver.selectVariable(state)
	if varID == 0 {
		t.Error("selectVariable() should not select assigned variable")
	}
	if len(values) == 0 {
		t.Error("selectVariable() should return values")
	}

	// Select when all variables are assigned
	state = solver.SetDomain(state, 1, NewBitSetDomainFromValues(10, []int{3}))
	state = solver.SetDomain(state, 2, NewBitSetDomainFromValues(10, []int{7}))
	varID, values = solver.selectVariable(state)
	if varID != -1 {
		t.Errorf("selectVariable() on complete state should return -1, got %d", varID)
	}
	if values != nil {
		t.Error("selectVariable() on complete state should return nil values")
	}
}

func TestSolver_ComputeVariableScore(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(10))

	// Test different heuristics
	heuristics := []VariableOrderingHeuristic{
		HeuristicDom,
		HeuristicDomDeg,
		HeuristicDeg,
		HeuristicLex,
	}

	for _, h := range heuristics {
		config := &SolverConfig{VariableHeuristic: h}
		solver := NewSolverWithConfig(model, config)

		domain := NewBitSetDomain(5)
		score := solver.computeVariableScore(0, domain)

		// Score should be a finite number
		if score != score { // NaN check
			t.Errorf("computeVariableScore with %v returned NaN", h)
		}
	}
}

func TestSolver_Solve_EmptyModel(t *testing.T) {
	model := NewModel()
	solver := NewSolver(model)

	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 1)

	if err != nil {
		t.Errorf("Solve() on empty model error = %v, want nil", err)
	}
	if len(solutions) != 1 {
		t.Errorf("Solve() on empty model returned %d solutions, want 1", len(solutions))
	}
	if len(solutions) > 0 && len(solutions[0]) != 0 {
		t.Errorf("Solve() on empty model solution length = %d, want 0", len(solutions[0]))
	}
}

func TestSolver_Solve_SimpleModel(t *testing.T) {
	model := NewModel()
	model.NewVariables(2, NewBitSetDomain(2))

	solver := NewSolver(model)
	ctx := context.Background()

	solutions, err := solver.Solve(ctx, 0) // Get all solutions

	if err != nil {
		t.Fatalf("Solve() error = %v", err)
	}

	// Should have 2*2 = 4 solutions
	if len(solutions) != 4 {
		t.Errorf("Solve() returned %d solutions, want 4", len(solutions))
	}

	// Each solution should have 2 values
	for i, sol := range solutions {
		if len(sol) != 2 {
			t.Errorf("solution[%d] length = %d, want 2", i, len(sol))
		}
	}
}

func TestSolver_Solve_WithLimit(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(3))

	solver := NewSolver(model)
	ctx := context.Background()

	// Request only 5 solutions
	solutions, err := solver.Solve(ctx, 5)

	if err != nil {
		t.Fatalf("Solve() error = %v", err)
	}

	if len(solutions) != 5 {
		t.Errorf("Solve() returned %d solutions, want 5", len(solutions))
	}
}

func TestSolver_Solve_WithTimeout(t *testing.T) {
	model := NewModel()
	model.NewVariables(10, NewBitSetDomain(10)) // Large search space

	solver := NewSolver(model)

	// Set a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	solutions, err := solver.Solve(ctx, 0)

	// Should either timeout or complete
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Solve() unexpected error = %v", err)
	}

	// Log how many solutions were found before timeout
	t.Logf("Found %d solutions before timeout", len(solutions))
}

func TestSolver_Solve_InvalidModel(t *testing.T) {
	model := NewModel()
	model.NewVariable(NewBitSetDomainFromValues(5, []int{})) // Empty domain

	solver := NewSolver(model)
	ctx := context.Background()

	_, err := solver.Solve(ctx, 1)
	if err == nil {
		t.Error("Solve() on invalid model should return error")
	}
}

func TestSolver_Solve_Cancellation(t *testing.T) {
	model := NewModel()
	model.NewVariables(10, NewBitSetDomain(10))

	solver := NewSolver(model)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := solver.Solve(ctx, 0)
	if err != context.Canceled {
		t.Errorf("Solve() with cancelled context error = %v, want %v", err, context.Canceled)
	}
}

func TestSolver_MultipleSolvers_ShareModel(t *testing.T) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(3))

	// Create multiple solvers sharing the same model
	solver1 := NewSolver(model)
	solver2 := NewSolver(model)

	if solver1.Model() != solver2.Model() {
		t.Error("solvers should share the same model instance")
	}

	// Both should be able to solve independently
	ctx := context.Background()

	solutions1, err1 := solver1.Solve(ctx, 1)
	solutions2, err2 := solver2.Solve(ctx, 1)

	if err1 != nil {
		t.Errorf("solver1.Solve() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("solver2.Solve() error = %v", err2)
	}

	if len(solutions1) == 0 || len(solutions2) == 0 {
		t.Error("both solvers should find solutions")
	}
}

// Benchmark solver operations
func BenchmarkSolver_SetDomain(b *testing.B) {
	model := NewModel()
	model.NewVariables(100, NewBitSetDomain(100))
	solver := NewSolver(model)
	domain := NewBitSetDomainFromValues(100, []int{50})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := solver.SetDomain(nil, 0, domain)
		solver.ReleaseState(state)
	}
}

func BenchmarkSolver_GetDomain(b *testing.B) {
	model := NewModel()
	model.NewVariables(100, NewBitSetDomain(100))
	solver := NewSolver(model)

	// Create a state chain
	var state *SolverState
	for i := 0; i < 10; i++ {
		domain := NewBitSetDomainFromValues(100, []int{i + 1})
		state = solver.SetDomain(state, i, domain)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		solver.GetDomain(state, 5)
	}
}

func BenchmarkSolver_Solve_Small(b *testing.B) {
	model := NewModel()
	model.NewVariables(3, NewBitSetDomain(3))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		solver := NewSolver(model)
		solver.Solve(ctx, 1)
	}
}

// Edge case tests for >90% coverage

func TestSolver_EdgeCases(t *testing.T) {
	t.Run("GetDomain with invalid varID", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(3, NewBitSetDomain(5))
		solver := NewSolver(model)

		// Out of bounds varID
		domain := solver.GetDomain(nil, 999)
		if domain != nil {
			t.Error("GetDomain with invalid varID should return nil")
		}

		// Negative varID
		domain = solver.GetDomain(nil, -1)
		if domain != nil {
			t.Error("GetDomain with negative varID should return nil")
		}
	})

	t.Run("SetMonitor", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(3, NewBitSetDomain(5))
		solver := NewSolver(model)

		monitor := NewSolverMonitor()
		solver.SetMonitor(monitor)

		// Monitor should be used during solving
		ctx := context.Background()
		_, err := solver.Solve(ctx, 1)
		if err != nil {
			t.Errorf("Solve with monitor error = %v", err)
		}

		// Verify monitor collected stats
		stats := monitor.GetStats()
		if stats.NodesExplored == 0 {
			t.Error("Monitor should have recorded nodes explored")
		}
	})

	t.Run("Propagation with monitor", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(2, NewBitSetDomain(3))
		solver := NewSolver(model)

		monitor := NewSolverMonitor()
		solver.SetMonitor(monitor)

		// Create a state and propagate
		state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(3, []int{1}))
		newState, err := solver.propagate(state)
		if err != nil {
			t.Errorf("propagate error = %v", err)
		}
		if newState == nil {
			t.Error("propagate should return a state")
		}

		// Monitor should have recorded propagation
		stats := monitor.GetStats()
		if stats.PropagationCount == 0 {
			t.Error("Monitor should have recorded propagation")
		}
	})

	t.Run("Nil monitor is safe", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(2, NewBitSetDomain(3))
		solver := NewSolver(model)

		// Don't set a monitor (nil monitor)
		ctx := context.Background()
		solutions, err := solver.Solve(ctx, 1)
		if err != nil {
			t.Errorf("Solve with nil monitor error = %v", err)
		}
		if len(solutions) == 0 {
			t.Error("Solve should find solutions even without monitor")
		}
	})

	t.Run("Propagation detects empty domain", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(2, NewBitSetDomain(3))
		solver := NewSolver(model)

		// Create state with empty domain
		emptyDomain := NewBitSetDomainFromValues(3, []int{})
		state := solver.SetDomain(nil, 0, emptyDomain)

		_, err := solver.propagate(state)
		if err == nil {
			t.Error("propagate should detect empty domain and return error")
		}
	})

	t.Run("ExtractSolution with unbound variable", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(2, NewBitSetDomain(5))
		solver := NewSolver(model)

		// Create incomplete state (only one variable bound)
		state := solver.SetDomain(nil, 0, NewBitSetDomainFromValues(5, []int{3}))

		// extractSolution should handle unbound variables
		solution := solver.extractSolution(state)
		if len(solution) != 2 {
			t.Errorf("extractSolution length = %d, want 2", len(solution))
		}
	})

	t.Run("Search with all heuristics", func(t *testing.T) {
		heuristics := []struct {
			name string
			h    VariableOrderingHeuristic
		}{
			{"Dom", HeuristicDom},
			{"DomDeg", HeuristicDomDeg},
			{"Deg", HeuristicDeg},
			{"Lex", HeuristicLex},
		}

		for _, tt := range heuristics {
			t.Run(tt.name, func(t *testing.T) {
				model := NewModel()
				model.NewVariables(2, NewBitSetDomain(2))
				model.SetConfig(&SolverConfig{VariableHeuristic: tt.h})

				solver := NewSolver(model)
				ctx := context.Background()
				solutions, err := solver.Solve(ctx, 1)

				if err != nil {
					t.Errorf("Solve with %s heuristic error = %v", tt.name, err)
				}
				if len(solutions) == 0 {
					t.Errorf("Solve with %s heuristic found no solutions", tt.name)
				}
			})
		}
	})

	t.Run("NewSolverWithConfig nil config", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(3, NewBitSetDomain(5))

		// Pass nil config, should use model's config
		solver := NewSolverWithConfig(model, nil)
		if solver == nil {
			t.Fatal("NewSolverWithConfig with nil should not return nil")
		}
	})

	t.Run("Search depth and backtracking", func(t *testing.T) {
		model := NewModel()
		model.NewVariables(5, NewBitSetDomain(3))
		solver := NewSolver(model)

		monitor := NewSolverMonitor()
		solver.SetMonitor(monitor)

		ctx := context.Background()
		solutions, err := solver.Solve(ctx, 10)

		if err != nil {
			t.Errorf("Solve error = %v", err)
		}
		if len(solutions) != 10 {
			t.Errorf("Solve found %d solutions, want 10", len(solutions))
		}

		stats := monitor.GetStats()
		if stats.NodesExplored == 0 {
			t.Error("Should have explored nodes")
		}
		if stats.Backtracks == 0 {
			t.Error("Should have backtracked")
		}
	})

	t.Run("ComputeVariableScore Deg heuristic", func(t *testing.T) {
		// Test Deg heuristic without constraints (degree will be 0)
		model := NewModel()
		model.NewVariable(NewBitSetDomain(5))
		model.NewVariable(NewBitSetDomain(5))

		config := &SolverConfig{VariableHeuristic: HeuristicDeg}
		solver := NewSolverWithConfig(model, config)

		domain := NewBitSetDomain(5)
		score := solver.computeVariableScore(0, domain)

		// Score should be 0 when no constraints exist
		if score != 0 {
			t.Errorf("computeVariableScore = %f, want 0 (no constraints)", score)
		}
	})
}

// Helper function to check domain nil safety
func TestDomain_NilSafety(t *testing.T) {
	d := NewBitSetDomain(10)

	// Equal with nil should not panic
	if d.Equal(nil) {
		t.Error("Equal(nil) should return false, not true")
	}
}
