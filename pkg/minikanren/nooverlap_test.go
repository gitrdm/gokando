package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestNoOverlap_BasicPropagation(t *testing.T) {
	model := NewModel()
	A := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{2}), "A")
	B := model.NewVariableWithName(NewBitSetDomain(4), "B")

	noov, err := NewNoOverlap([]*FDVariable{A, B}, []int{2, 2})
	if err != nil {
		t.Fatalf("NewNoOverlap error: %v", err)
	}
	model.AddConstraint(noov)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = solver.Solve(ctx, 1)

	if got := solver.GetDomain(nil, B.ID()).String(); got != "{4}" {
		t.Fatalf("expected B={4}, got %s", got)
	}
}

func TestNoOverlap_ConstructorValidation(t *testing.T) {
	model := NewModel()
	v := model.NewVariable(NewBitSetDomain(5))

	if _, err := NewNoOverlap(nil, nil); err == nil {
		t.Fatalf("expected error for nil inputs")
	}
	if _, err := NewNoOverlap([]*FDVariable{v}, []int{}); err == nil {
		t.Fatalf("expected error for mismatched lengths")
	}
}
