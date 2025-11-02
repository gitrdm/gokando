package minikanren

import (
	"context"
	"testing"
	"time"
)

// Test basic pruning: a fixed high-demand task blocks overlapping starts
// for a second task when adding it would exceed capacity.
func TestCumulative_PruneStarts(t *testing.T) {
	model := NewModel()

	// Task A: fixed at start=2, duration=2, demand=2
	a := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	// Task B: start in [1..4], duration=2, demand=1
	b := model.NewVariableWithName(NewBitSetDomain(4), "B")

	cap := 2
	durations := []int{2, 2}
	demands := []int{2, 1}
	cum, err := NewCumulative([]*FDVariable{a, b}, durations, demands, cap)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)

	solver := NewSolver(model)
	// Run one explicit propagate via Solve (which caches base state)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	// Inspect post-propagation domain for B (should be pruned to {4})
	domB := solver.GetDomain(nil, b.ID())
	want := NewBitSetDomainFromValues(domB.MaxValue(), []int{4})
	if !domB.Equal(want) {
		t.Fatalf("unexpected B domain: got %s, want %s", domB.String(), want.String())
	}
}

// Ensure Propagate returns a changed state for the same scenario.
func TestCumulative_DirectPropagate(t *testing.T) {
	model := NewModel()
	a := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	b := model.NewVariableWithName(NewBitSetDomain(4), "B")
	cum, err := NewCumulative([]*FDVariable{a, b}, []int{2, 2}, []int{2, 1}, 2)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)
	solver := NewSolver(model)
	// Call propagate directly
	newState, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate returned error: %v", err)
	}
	domB := solver.GetDomain(newState, b.ID())
	want := NewBitSetDomainFromValues(domB.MaxValue(), []int{4})
	if !domB.Equal(want) {
		t.Fatalf("unexpected B domain after direct propagate: got %s, want %s", domB.String(), want.String())
	}
}

// Test immediate inconsistency when compulsory parts alone exceed capacity.
func TestCumulative_Inconsistency(t *testing.T) {
	model := NewModel()

	// Two tasks both fixed overlapping; combined demand 4 > capacity 3.
	x := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "X")
	y := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "Y")

	cum, err := NewCumulative([]*FDVariable{x, y}, []int{2, 2}, []int{2, 2}, 3)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// No solutions should be found; solver returns zero solutions without error.
	sols, err := solver.Solve(ctx, 1)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) != 0 {
		t.Fatalf("expected no solutions due to capacity overload, got %d", len(sols))
	}
}

// Test constructor validation paths.
func TestCumulative_ConstructorValidation(t *testing.T) {
	model := NewModel()
	v := model.NewVariable(NewBitSetDomain(5))

	// Empty starts
	if _, err := NewCumulative(nil, nil, nil, 1); err == nil {
		t.Fatalf("expected error for empty starts")
	}
	// Length mismatch
	if _, err := NewCumulative([]*FDVariable{v}, []int{1}, []int{}, 1); err == nil {
		t.Fatalf("expected error for length mismatch")
	}
	// Capacity <= 0
	if _, err := NewCumulative([]*FDVariable{v}, []int{1}, []int{0}, 0); err == nil {
		t.Fatalf("expected error for capacity <= 0")
	}
	// Duration <= 0
	if _, err := NewCumulative([]*FDVariable{v}, []int{0}, []int{1}, 1); err == nil {
		t.Fatalf("expected error for non-positive duration")
	}
	// Negative demand
	if _, err := NewCumulative([]*FDVariable{v}, []int{1}, []int{-1}, 1); err == nil {
		t.Fatalf("expected error for negative demand")
	}
}
