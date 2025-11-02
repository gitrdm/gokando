package minikanren

import (
	"context"
	"testing"
)

func TestDistinctCount_ConstructValidation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1}))
	_, err := NewDistinctCount(nil, []*FDVariable{x}, model.NewVariable(NewBitSetDomain(2)))
	if err == nil {
		t.Fatalf("expected error on nil model")
	}

	_, err = NewDistinctCount(model, []*FDVariable{}, model.NewVariable(NewBitSetDomain(2)))
	if err == nil {
		t.Fatalf("expected error on empty vars")
	}

	_, err = NewDistinctCount(model, []*FDVariable{x}, nil)
	if err == nil {
		t.Fatalf("expected error on nil dPlus1")
	}
}

// If one variable is fixed to value 1 and we enforce AtMostNValues=1,
// other variables cannot take any value other than 1.
func TestAtMostNValues_PruneOnFixedValue(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1})) // fixed to 1
	x2 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2}))
	x3 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2}))
	limit := model.NewVariable(NewBitSetDomain(2)) // [1..2] => distinct ≤ 1

	_, err := NewAtMostNValues(model, []*FDVariable{x1, x2, x3}, limit)
	if err != nil {
		t.Fatalf("NewAtMostNValues error: %v", err)
	}

	solver := NewSolver(model)
	if _, err := solver.Solve(context.Background(), 0); err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	d2 := solver.GetDomain(nil, x2.ID())
	d3 := solver.GetDomain(nil, x3.ID())
	want := NewBitSetDomainFromValues(5, []int{1})
	if !d2.Equal(want) {
		t.Fatalf("x2 domain mismatch: got %v want %v", d2, want)
	}
	if !d3.Equal(want) {
		t.Fatalf("x3 domain mismatch: got %v want %v", d3, want)
	}
}

// When all variables have symmetric domains and NValue=1, no pruning is needed
// since the model admits solutions with all equal values.
func TestAtMostNValues_NoPruneWithSymmetricDomains(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2}))
	limit := model.NewVariable(NewBitSetDomain(2)) // distinct ≤ 1

	_, err := NewAtMostNValues(model, []*FDVariable{x1, x2}, limit)
	if err != nil {
		t.Fatalf("NewAtMostNValues error: %v", err)
	}

	solver := NewSolver(model)
	if _, err := solver.Solve(context.Background(), 0); err != nil {
		t.Fatalf("propagation error: %v", err)
	}

	d1 := solver.GetDomain(nil, x1.ID())
	d2 := solver.GetDomain(nil, x2.ID())
	want := NewBitSetDomainFromValues(5, []int{1, 2})
	if !d1.Equal(want) || !d2.Equal(want) {
		t.Fatalf("unexpected pruning for symmetric domains: x1=%v x2=%v", d1, d2)
	}
}
