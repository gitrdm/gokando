package minikanren

import "testing"

// TestLinearSum_PrunesTotalBounds verifies total domain is tightened to the
// min/max achievable weighted sum given current variable bounds.
func TestLinearSum_PrunesTotalBounds(t *testing.T) {
	model := NewModel()

	// X in [1..5], Y in [1..5]
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	// Total in [1..100]
	tot := model.NewVariable(NewBitSetDomain(100))

	coeffs := []int{1, 2}
	ls, err := NewLinearSum([]*FDVariable{x, y}, coeffs, tot)
	if err != nil {
		t.Fatalf("NewLinearSum failed: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	newState, err := ls.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	tDom := solver.GetDomain(newState, tot.ID())
	// SumMin = 1*1 + 2*1 = 3; SumMax = 1*5 + 2*5 = 15
	if tDom.Min() != 3 || tDom.Max() != 15 {
		t.Fatalf("unexpected total bounds: got [%d,%d], want [3,15]", tDom.Min(), tDom.Max())
	}
}

// TestLinearSum_PrunesVariableBounds verifies per-variable bounds tighten
// based on the total domain when coefficients are non-zero.
func TestLinearSum_PrunesVariableBounds(t *testing.T) {
	model := NewModel()

	// X in [1..5], Y in [1..5]
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	// Total fixed to 10
	tot := model.NewVariable(NewBitSetDomain(100))

	coeffs := []int{1, 2}
	ls, err := NewLinearSum([]*FDVariable{x, y}, coeffs, tot)
	if err != nil {
		t.Fatalf("NewLinearSum failed: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Fix total to 10
	state, _ = solver.SetDomain(state, tot.ID(), NewBitSetDomainFromValues(100, []int{10}))

	newState, err := ls.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	xDom := solver.GetDomain(newState, x.ID())
	yDom := solver.GetDomain(newState, y.ID())

	// For coeffs [1,2] and t=10:
	// - X can remain [1..5]
	// - Y must be in [3..4]
	if xDom.Min() != 1 || xDom.Max() != 5 {
		t.Errorf("unexpected X bounds: got [%d,%d], want [1,5]", xDom.Min(), xDom.Max())
	}
	if yDom.Min() != 3 || yDom.Max() != 4 {
		t.Errorf("unexpected Y bounds: got [%d,%d], want [3,4]", yDom.Min(), yDom.Max())
	}
}

// TestLinearSum_ZeroCoefficient ensures zero-weighted variables are ignored
// by the propagator (no pruning on that variable from this constraint).
func TestLinearSum_ZeroCoefficient(t *testing.T) {
	model := NewModel()

	z := model.NewVariable(NewBitSetDomain(9)) // should remain [1..9]
	x := model.NewVariable(NewBitSetDomain(5))
	tot := model.NewVariable(NewBitSetDomain(100))

	coeffs := []int{0, 2}
	ls, err := NewLinearSum([]*FDVariable{z, x}, coeffs, tot)
	if err != nil {
		t.Fatalf("NewLinearSum failed: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Narrow total to force some pruning on x
	state, _ = solver.SetDomain(state, tot.ID(), NewBitSetDomainFromValues(100, []int{6}))

	newState, err := ls.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	zDom := solver.GetDomain(newState, z.ID())
	if zDom.Min() != 1 || zDom.Max() != 9 {
		t.Errorf("zero-coeff var pruned unexpectedly: got [%d,%d], want [1,9]", zDom.Min(), zDom.Max())
	}
}

// TestLinearSum_Inconsistency ensures incompatible bounds lead to failure.
func TestLinearSum_Inconsistency(t *testing.T) {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(100, []int{10}))     // fixed at 10
	tot := model.NewVariable(NewBitSetDomainFromValues(100, []int{1, 2})) // too small

	coeffs := []int{3}
	ls, err := NewLinearSum([]*FDVariable{x}, coeffs, tot)
	if err != nil {
		t.Fatalf("NewLinearSum failed: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	_, err = ls.Propagate(solver, state)
	if err == nil {
		t.Fatalf("expected inconsistency error, got nil")
	}
}
