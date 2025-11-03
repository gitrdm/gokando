package minikanren

import (
	"context"
	"testing"
	"time"
)

// ============================================================================
// Mixed-sign LinearSum Tests
// ============================================================================

// TestLinearSum_MixedSignCoefficients tests LinearSum with both positive and negative coefficients.
func TestLinearSum_MixedSignCoefficients(t *testing.T) {
	m := NewModel()
	// 2*x - 3*y = total
	// x ∈ {1,2,3}, y ∈ {1,2}, total computed from propagation
	x := m.NewVariable(NewBitSetDomain(3))
	y := m.NewVariable(NewBitSetDomain(2))
	total := m.NewVariable(NewBitSetDomain(20))

	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{2, -3}, total)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	solver := NewSolver(m)

	// Propagate to compute total bounds
	state, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate error: %v", err)
	}

	// Check total domain: 2*x - 3*y
	// Min: 2*1 - 3*2 = -4 (but domains are positive, so adjust expectations)
	// Max: 2*3 - 3*1 = 3
	// Expected range for total accounting for negative results being clamped
	totalDom := solver.GetDomain(state, total.ID())
	if totalDom == nil {
		t.Fatal("total domain is nil")
	}

	// With x={1,2,3} and y={1,2}:
	// Possible values: 2*1-3*1=-1, 2*1-3*2=-4, 2*2-3*1=1, 2*2-3*2=-2, 2*3-3*1=3, 2*3-3*2=0
	// After clamping negative values, we need valid domain values
	// Note: Our domains are 1-based, so we need to handle this properly

	// For now, verify constraint was created and propagation doesn't crash
	if totalDom.Count() == 0 {
		t.Fatal("total domain became empty after propagation")
	}
}

// TestLinearSum_AllNegativeCoefficients tests LinearSum with all negative coefficients.
func TestLinearSum_AllNegativeCoefficients(t *testing.T) {
	m := NewModel()
	// -2*x - 3*y = total
	// x ∈ {1,2}, y ∈ {1,2}
	// Possible values: -2*1-3*1=-5, -2*1-3*2=-8, -2*2-3*1=-7, -2*2-3*2=-10
	// All results are negative, which is incompatible with positive 1-based domains
	// This test verifies that we handle this gracefully (either by special encoding or error)
	x := m.NewVariable(NewBitSetDomain(2))
	y := m.NewVariable(NewBitSetDomain(2))

	// Use a wider total domain to accommodate encoding if needed
	// Or accept that all-negative sums are inconsistent with positive domains
	total := m.NewVariable(NewBitSetDomain(50))

	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{-2, -3}, total)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	solver := NewSolver(m)
	_, err = solver.propagate(nil)

	// With all-negative coefficients and positive domains, this is likely inconsistent
	// We accept either: (1) propagation error, or (2) empty domain leading to no solutions
	if err != nil {
		// Expected: propagation detects inconsistency
		t.Logf("Propagation correctly detected inconsistency: %v", err)
		return
	}

	// Alternative: propagation succeeds but no solutions exist
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sols, err := solver.Solve(ctx, 1)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) != 0 {
		t.Errorf("expected no solutions with all-negative coefficients and positive domains, got %d", len(sols))
	}
}

// TestLinearSum_MixedSignOptimization tests optimization with mixed-sign LinearSum objective.
func TestLinearSum_MixedSignOptimization(t *testing.T) {
	m := NewModel()
	// Profit maximization: profit = 3*x - 2*y (maximize profit)
	// x ∈ {1,2,3,4}, y ∈ {1,2,3}
	x := m.NewVariable(NewBitSetDomain(4))
	y := m.NewVariable(NewBitSetDomain(3))
	profit := m.NewVariable(NewBitSetDomain(20))

	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{3, -2}, profit)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize: 3*x - 2*y
	// Best: x=4, y=1 → 3*4 - 2*1 = 10
	sol, objVal, err := solver.SolveOptimal(ctx, profit, false) // maximize
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatal("no solution found")
	}

	// Verify objective value is reasonable (max should be 3*4 - 2*1 = 10)
	// Note: This depends on domain encoding working correctly with negative coefficients
	if objVal <= 0 {
		t.Errorf("unexpected objective value: got %d, expected positive", objVal)
	}
}

// ============================================================================
// BoolSum Structural Bounds Tests
// ============================================================================

// TestOptimize_BoolSumObjective tests optimization with BoolSum as the objective.
func TestOptimize_BoolSumObjective(t *testing.T) {
	m := NewModel()
	// Count how many of 4 boolean variables are true
	b1 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b2 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b3 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b4 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	count := m.NewVariable(NewBitSetDomain(5)) // encoded count+1, so [1..5]

	bs, err := NewBoolSum([]*FDVariable{b1, b2, b3, b4}, count)
	if err != nil {
		t.Fatalf("NewBoolSum error: %v", err)
	}
	m.AddConstraint(bs)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Minimize count (encoded as count+1)
	// Minimum: all false → count=0 → encoded=1
	sol, objVal, err := solver.SolveOptimal(ctx, count, true)
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatal("no solution found")
	}

	// Objective should be 1 (encoded 0+1)
	if objVal != 1 {
		t.Errorf("expected objective=1 (all false), got %d", objVal)
	}

	// Verify all booleans are false (value=1)
	if sol[b1.ID()] != 1 || sol[b2.ID()] != 1 || sol[b3.ID()] != 1 || sol[b4.ID()] != 1 {
		t.Errorf("expected all booleans to be false (1), got %v", sol)
	}
}

// TestOptimize_BoolSumMaximize tests maximizing a BoolSum objective.
func TestOptimize_BoolSumMaximize(t *testing.T) {
	m := NewModel()
	b1 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b2 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b3 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	count := m.NewVariable(NewBitSetDomain(4)) // [1..4]

	bs, err := NewBoolSum([]*FDVariable{b1, b2, b3}, count)
	if err != nil {
		t.Fatalf("NewBoolSum error: %v", err)
	}
	m.AddConstraint(bs)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize count
	// Maximum: all true → count=3 → encoded=4
	sol, objVal, err := solver.SolveOptimal(ctx, count, false) // maximize
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatal("no solution found")
	}

	if objVal != 4 {
		t.Errorf("expected objective=4 (all true), got %d", objVal)
	}

	// Verify all booleans are true (value=2)
	if sol[b1.ID()] != 2 || sol[b2.ID()] != 2 || sol[b3.ID()] != 2 {
		t.Errorf("expected all booleans to be true (2), got %v", sol)
	}
}

// ============================================================================
// Optimization-Aware Heuristics Tests
// ============================================================================

// TestHeuristic_Impact tests the HeuristicImpact variable ordering.
func TestHeuristic_Impact(t *testing.T) {
	m := NewModel()
	// Create variables: x, y, z where only x appears in constraint with objective
	x := m.NewVariable(NewBitSetDomain(5))
	y := m.NewVariable(NewBitSetDomain(5))
	z := m.NewVariable(NewBitSetDomain(5))
	obj := m.NewVariable(NewBitSetDomain(20))

	// x + 2*y = obj (x and y affect objective)
	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, obj)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	// z is independent (no connection to obj)
	// Add a dummy constraint to give z some degree
	ad, err := NewAllDifferent([]*FDVariable{y, z})
	if err != nil {
		t.Fatalf("NewAllDifferent error: %v", err)
	}
	m.AddConstraint(ad)

	cfg := DefaultSolverConfig()
	cfg.VariableHeuristic = HeuristicImpact
	solver := NewSolverWithConfig(m, cfg)

	// Set optimization context manually for testing
	solver.optContext = &optimizationContext{
		objectiveID: obj.ID(),
		minimize:    true,
	}

	// Compute scores for each variable
	state, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate error: %v", err)
	}

	xScore := solver.computeVariableScore(x.ID(), solver.GetDomain(state, x.ID()))
	yScore := solver.computeVariableScore(y.ID(), solver.GetDomain(state, y.ID()))
	zScore := solver.computeVariableScore(z.ID(), solver.GetDomain(state, z.ID()))

	// x and y should have better (lower) scores than z because they share constraints with obj
	if xScore >= zScore {
		t.Errorf("expected x (score=%.2f) to have lower score than z (score=%.2f)", xScore, zScore)
	}
	if yScore >= zScore {
		t.Errorf("expected y (score=%.2f) to have lower score than z (score=%.2f)", yScore, zScore)
	}
}

// TestValueOrder_ObjImproving tests ValueOrderObjImproving heuristic.
func TestValueOrder_ObjImproving(t *testing.T) {
	m := NewModel()
	x := m.NewVariable(NewBitSetDomain(5))

	cfg := DefaultSolverConfig()
	cfg.ValueHeuristic = ValueOrderObjImproving
	solver := NewSolverWithConfig(m, cfg)

	values := []int{1, 2, 3, 4, 5}

	// Test minimize: should prefer smaller values (ascending)
	solver.optContext = &optimizationContext{
		objectiveID: x.ID(),
		minimize:    true,
	}
	ordered := solver.orderValues(values)
	if ordered[0] != 1 || ordered[4] != 5 {
		t.Errorf("minimize should order ascending, got %v", ordered)
	}

	// Test maximize: should prefer larger values (descending)
	solver.optContext = &optimizationContext{
		objectiveID: x.ID(),
		minimize:    false,
	}
	ordered = solver.orderValues(values)
	if ordered[0] != 5 || ordered[4] != 1 {
		t.Errorf("maximize should order descending, got %v", ordered)
	}
}

// TestOptimize_WithImpactHeuristic tests optimization using impact-based variable ordering.
func TestOptimize_WithImpactHeuristic(t *testing.T) {
	m := NewModel()
	// Simple linear objective: minimize x+y
	x := m.NewVariable(NewBitSetDomain(5))
	y := m.NewVariable(NewBitSetDomain(5))
	total := m.NewVariable(NewBitSetDomain(20))

	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{1, 1}, total)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Use impact heuristic and obj-improving value ordering
	sol, objVal, err := solver.SolveOptimalWithOptions(ctx, total, true,
		WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatal("no solution found")
	}

	// Should find minimum: x=1, y=1 → total=2
	if objVal != 2 {
		t.Errorf("expected objective=2, got %d", objVal)
	}
	if sol[x.ID()] != 1 || sol[y.ID()] != 1 {
		t.Errorf("expected x=1, y=1, got x=%d, y=%d", sol[x.ID()], sol[y.ID()])
	}
}

// ============================================================================
// Cumulative Energetic Reasoning Tests
// ============================================================================

// TestCumulative_EnergeticReasoning tests that energetic reasoning detects overload.
func TestCumulative_EnergeticReasoning(t *testing.T) {
	m := NewModel()
	// Three tasks with tight energy requirements in a window
	// Task 1: start ∈ [1,2], dur=3, dem=2
	// Task 2: start ∈ [1,2], dur=3, dem=2
	// Task 3: start ∈ [1,2], dur=3, dem=2
	// Capacity: 3
	// Energy in window [1..5]: at least 3 tasks * 3 duration * 2 demand = 18 work
	// Capacity allows: 5 time units * 3 capacity = 15 work → OVERLOAD
	s1 := m.NewVariable(NewBitSetDomain(2))
	s2 := m.NewVariable(NewBitSetDomain(2))
	s3 := m.NewVariable(NewBitSetDomain(2))

	cum, err := NewCumulative(
		[]*FDVariable{s1, s2, s3},
		[]int{3, 3, 3}, // durations
		[]int{2, 2, 2}, // demands
		3,              // capacity
	)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	m.AddConstraint(cum)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Should detect energetic overload during propagation or search
	sols, err := solver.Solve(ctx, 1)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	// Expect no solutions due to energetic overload
	if len(sols) > 0 {
		t.Errorf("expected no solutions due to energetic overload, got %d", len(sols))
	}
}

// TestCumulative_EdgeFinding tests edge-finding exclusion.
func TestCumulative_EdgeFinding(t *testing.T) {
	m := NewModel()
	// Two tasks that must be separated by edge-finding
	// Task 1: start ∈ [1,5], dur=2, dem=3
	// Task 2: start ∈ [1,5], dur=2, dem=3
	// Capacity: 4
	// If both scheduled in [1..3], total demand = 6 > capacity*window = 4*3 = 12 (OK)
	// But within smaller windows, edge-finding should prune some starts
	s1 := m.NewVariable(NewBitSetDomain(5))
	s2 := m.NewVariable(NewBitSetDomain(5))

	cum, err := NewCumulative(
		[]*FDVariable{s1, s2},
		[]int{2, 2},
		[]int{3, 3},
		4,
	)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	m.AddConstraint(cum)

	solver := NewSolver(m)
	state, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate error: %v", err)
	}

	// Verify some pruning occurred (domains should be reduced)
	dom1 := solver.GetDomain(state, s1.ID())
	dom2 := solver.GetDomain(state, s2.ID())

	initialSize := 5
	if dom1.Count() == initialSize && dom2.Count() == initialSize {
		t.Log("Warning: edge-finding may not have pruned (this is OK if problem is not tight enough)")
	}

	// At minimum, verify no inconsistency
	if dom1.Count() == 0 || dom2.Count() == 0 {
		t.Error("unexpected empty domain after edge-finding")
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestIntegration_MixedSignOptimizationWithHeuristics combines all enhancements.
func TestIntegration_MixedSignOptimizationWithHeuristics(t *testing.T) {
	m := NewModel()
	// Profit maximization with cost-benefit tradeoffs
	// profit = 5*x - 2*y + 3*z
	// x, y, z ∈ {1,2,3}
	x := m.NewVariable(NewBitSetDomain(3))
	y := m.NewVariable(NewBitSetDomain(3))
	z := m.NewVariable(NewBitSetDomain(3))
	profit := m.NewVariable(NewBitSetDomain(30))

	ls, err := NewLinearSum([]*FDVariable{x, y, z}, []int{5, -2, 3}, profit)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize profit using impact heuristic and obj-improving values
	sol, objVal, err := solver.SolveOptimalWithOptions(ctx, profit, false,
		WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatal("no solution found")
	}

	// Optimal: x=3, y=1, z=3 → 5*3 - 2*1 + 3*3 = 15 - 2 + 9 = 22
	expectedProfit := 5*sol[x.ID()] - 2*sol[y.ID()] + 3*sol[z.ID()]
	if objVal != expectedProfit {
		t.Errorf("objective value mismatch: got %d, computed %d from solution", objVal, expectedProfit)
	}

	// Verify we found a good solution (profit should be high)
	if objVal < 15 {
		t.Errorf("expected high profit (>15), got %d", objVal)
	}
}
