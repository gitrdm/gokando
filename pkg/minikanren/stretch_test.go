package minikanren

import (
	"context"
	"testing"
)

func TestNewStretch_Validation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	vars := []*FDVariable{x}

	if _, err := NewStretch(nil, vars, []int{1}, []int{1}, []int{1}); err == nil {
		t.Fatalf("expected error on nil model")
	}
	if _, err := NewStretch(model, []*FDVariable{}, []int{1}, []int{1}, []int{1}); err == nil {
		t.Fatalf("expected error on empty vars")
	}
	if _, err := NewStretch(model, vars, []int{0}, []int{1}, []int{1}); err == nil {
		t.Fatalf("expected error on non-positive value")
	}
	if _, err := NewStretch(model, vars, []int{1, 2}, []int{1}, []int{1}); err == nil {
		t.Fatalf("expected error on mismatched lengths")
	}
	if _, err := NewStretch(model, vars, []int{1}, []int{2}, []int{1}); err == nil {
		t.Fatalf("expected error on min>max")
	}
	if _, err := NewStretch(model, vars, []int{1}, []int{6}, []int{6}); err == nil {
		t.Fatalf("expected error on lengths exceeding n")
	}
}

// With n=5, domains {1,2}, value 1 constrained to runs of exactly 2.
// Forcing x3=2 splits into two segments requiring pairs: x1=x2=1 and x4=x5=1.
func TestStretch_RunLengthPruning(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1})) // force 1
	x3 := model.NewVariable(NewBitSetDomainFromValues(2, []int{2})) // force separator
	x4 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1})) // force 1
	x5 := model.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))

	_, err := NewStretch(model, []*FDVariable{x1, x2, x3, x4, x5}, []int{1}, []int{2}, []int{2})
	if err != nil {
		t.Fatalf("NewStretch failed: %v", err)
	}

	solver := NewSolver(model)
	if _, err := solver.Solve(context.Background(), 0); err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	want1 := NewBitSetDomainFromValues(2, []int{1})
	// x2 and x4 are already 1; Stretch should force x1 and x5 to 1 to complete runs of length 2.
	for idx, v := range []*FDVariable{x1, x5} {
		d := solver.GetDomain(nil, v.ID())
		if !d.Equal(want1) {
			t.Fatalf("expected x[%d] forced to {1}, got %v", idx+1, d)
		}
	}
}
