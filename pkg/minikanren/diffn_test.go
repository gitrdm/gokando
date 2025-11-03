package minikanren

import (
	"context"
	"testing"
)

func TestNewDiffn_Validation(t *testing.T) {
	model := NewModel()
	x := []*FDVariable{model.NewVariable(NewBitSetDomain(10))}
	y := []*FDVariable{model.NewVariable(NewBitSetDomain(10))}
	if _, err := NewDiffn(nil, x, y, []int{1}, []int{1}); err == nil {
		t.Fatalf("expected error on nil model")
	}
	if _, err := NewDiffn(model, []*FDVariable{}, []*FDVariable{}, []int{}, []int{}); err == nil {
		t.Fatalf("expected error on empty inputs")
	}
	if _, err := NewDiffn(model, x, y, []int{0}, []int{1}); err == nil {
		t.Fatalf("expected error on non-positive size")
	}
}

// Two 2x2 squares at y=1 cannot overlap vertically; disjunction forces
// horizontal separation: X2 >= X1+2 when Y1=Y2=1.
func TestDiffn_BasicPruning(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))
	y1 := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4}))
	y2 := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))

	_, err := NewDiffn(model, []*FDVariable{x1, x2}, []*FDVariable{y1, y2}, []int{2, 2}, []int{2, 2})
	if err != nil {
		t.Fatalf("NewDiffn failed: %v", err)
	}

	solver := NewSolver(model)
	if _, err := solver.Solve(context.Background(), 0); err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	dx2 := solver.GetDomain(nil, x2.ID())
	want := NewBitSetDomainFromValues(10, []int{3, 4})
	if !dx2.Equal(want) {
		t.Fatalf("x2 domain mismatch: got %v want %v", dx2, want)
	}
}
