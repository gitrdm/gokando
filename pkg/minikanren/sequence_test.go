package minikanren

import (
	"context"
	"testing"
)

func TestNewInSetReified_Validation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	b := model.NewVariable(NewBitSetDomain(2))
	if _, err := NewInSetReified(nil, []int{1}, b); err == nil {
		t.Fatalf("expected error on nil v")
	}
	if _, err := NewInSetReified(x, []int{}, b); err == nil {
		t.Fatalf("expected error on empty set")
	}
	if _, err := NewInSetReified(x, []int{-1}, b); err == nil {
		t.Fatalf("expected error on negative set value")
	}
}

func TestNewSequence_Validation(t *testing.T) {
	model := NewModel()
	xs := []*FDVariable{model.NewVariable(NewBitSetDomain(5))}
	if _, err := NewSequence(nil, xs, []int{1}, 1, 0, 1); err == nil {
		t.Fatalf("expected error on nil model")
	}
	if _, err := NewSequence(model, []*FDVariable{}, []int{1}, 1, 0, 1); err == nil {
		t.Fatalf("expected error on empty vars")
	}
	if _, err := NewSequence(model, xs, []int{1}, 0, 0, 0); err == nil {
		t.Fatalf("expected error on bad k")
	}
	if _, err := NewSequence(model, xs, []int{1}, 1, 2, 1); err == nil {
		t.Fatalf("expected error on min>max")
	}
	if _, err := NewSequence(model, xs, []int{1}, 1, 0, 2); err == nil {
		t.Fatalf("expected error on max>k")
	}
}

// With k=3, min=2 and some positions already forced to not-in-S, remaining
// positions in a window must be in S. We model S={1} and force X2=2.
func TestSequence_BasicPruning(t *testing.T) {
	model := NewModel()
	// Vars domains {1,2}; S={1}
	x1 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(2, []int{2})) // not in S
	x3 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	x4 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	x5 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))

	_, err := NewSequence(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, 3, 2, 3)
	if err != nil {
		t.Fatalf("NewSequence failed: %v", err)
	}

	solver := NewSolver(model)
	if _, err := solver.Solve(context.Background(), 0); err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	// Window [x1,x2,x3] must have at least two 1s; since x2!=1, both x1 and x3 must be 1
	d1 := solver.GetDomain(nil, x1.ID())
	d3 := solver.GetDomain(nil, x3.ID())
	want := NewBitSetDomainFromValues(2, []int{1})
	if !d1.Equal(want) || !d3.Equal(want) {
		t.Fatalf("expected x1 and x3 forced to {1}, got x1=%v x3=%v", d1, d3)
	}
}
