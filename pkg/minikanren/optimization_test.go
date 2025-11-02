package minikanren

import (
	"context"
	"errors"
	"testing"
)

// Minimize a single variable's value without additional constraints.
func TestSolveOptimal_MinimizeIdentity(t *testing.T) {
	model := NewModel()
	// Domain [1..10]
	x := model.NewVariable(NewBitSetDomain(10))

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), x, true)
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatalf("expected a solution, got nil")
	}
	if obj != 1 {
		t.Fatalf("expected objective 1, got %d", obj)
	}
}

// Minimize a LinearSum total: x + 2*y = T, minimize T.
func TestSolveOptimal_LinearSumMinimize(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	tvar := model.NewVariable(NewBitSetDomain(20))

	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	if err != nil {
		t.Fatalf("NewLinearSum error: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), tvar, true)
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatalf("expected a solution")
	}
	// Minimum is x=1, y=1 → T=3
	if obj != 3 {
		t.Fatalf("expected objective 3, got %d", obj)
	}
}

// Integrate with Cumulative via a simple makespan: two tasks on a single machine (capacity=1).
// Define end times e1,e2 and a makespan M ≥ e1,e2; minimize M.
func TestSolveOptimal_MinimizeMakespanTwoTasks(t *testing.T) {
	model := NewModel()
	// starts in [1..5]
	s1 := model.NewVariable(NewBitSetDomain(5))
	s2 := model.NewVariable(NewBitSetDomain(5))
	// durations
	durs := []int{2, 1}
	// Demands=1 and capacity=1 ⇒ NoOverlap
	cum, err := NewCumulative([]*FDVariable{s1, s2}, durs, []int{1, 1}, 1)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)

	// End times: e = s + dur - 1 (half-open mapped to inclusive)
	e1 := model.NewVariable(NewBitSetDomain(8))
	e2 := model.NewVariable(NewBitSetDomain(8))
	// s1 + (dur1-1) = e1; s2 + (dur2-1) = e2
	c1, err := NewArithmetic(s1, e1, durs[0]-1)
	if err != nil {
		t.Fatalf("NewArithmetic c1: %v", err)
	}
	c2, err := NewArithmetic(s2, e2, durs[1]-1)
	if err != nil {
		t.Fatalf("NewArithmetic c2: %v", err)
	}
	model.AddConstraint(c1)
	model.AddConstraint(c2)

	// Makespan M ≥ e1 and M ≥ e2
	m := model.NewVariable(NewBitSetDomain(8))
	ge1, err := NewInequality(m, e1, GreaterEqual)
	if err != nil {
		t.Fatalf("ineq e1<=m: %v", err)
	}
	ge2, err := NewInequality(m, e2, GreaterEqual)
	if err != nil {
		t.Fatalf("ineq e2<=m: %v", err)
	}
	model.AddConstraint(ge1)
	model.AddConstraint(ge2)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), m, true)
	if err != nil {
		t.Fatalf("SolveOptimal error: %v", err)
	}
	if sol == nil {
		t.Fatalf("expected a solution")
	}
	// With capacity=1, durations 2 and 1, the minimal makespan is at least 3.
	if obj < 3 {
		t.Fatalf("expected makespan >= 3, got %d", obj)
	}
}

// Parallel minimize identity objective: should find 1 with multiple workers.
func TestSolveOptimal_Parallel_MinimizeIdentity(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimalWithOptions(context.Background(), x, true, WithParallelWorkers(4))
	if err != nil && err != context.DeadlineExceeded && err != ErrSearchLimitReached {
		t.Fatalf("unexpected error: %v", err)
	}
	if sol == nil {
		t.Fatalf("expected a solution")
	}
	if obj != 1 {
		t.Fatalf("expected objective 1, got %d", obj)
	}
}

// Node limit triggers anytime return of incumbent. Ensure we return a valid incumbent and the limit error.
func TestSolveOptimal_NodeLimit_ReturnsIncumbent(t *testing.T) {
	model := NewModel()
	// Small linear sum to force at least a couple nodes
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 4}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 4}))
	tvar := model.NewVariable(NewBitSetDomain(30))
	ls, err := NewLinearSum([]*FDVariable{x, y}, []int{1, 1}, tvar)
	if err != nil {
		t.Fatalf("NewLinearSum: %v", err)
	}
	model.AddConstraint(ls)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimalWithOptions(context.Background(), tvar, true, WithNodeLimit(1))
	if err == nil {
		t.Fatalf("expected ErrSearchLimitReached, got nil")
	}
	if !errors.Is(err, ErrSearchLimitReached) {
		t.Fatalf("expected ErrSearchLimitReached, got %v", err)
	}
	// We should have an incumbent (the first leaf often)
	if sol == nil {
		t.Fatalf("expected incumbent solution, got nil")
	}
	// Objective should be within feasible bounds
	if obj < 4 || obj > 8 {
		t.Fatalf("unexpected incumbent objective: %d", obj)
	}
}
