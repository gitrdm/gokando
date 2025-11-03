package minikanren

import (
	"context"
	"testing"
	"time"
)

// Basic pruning when K's upper bound equals the number of mandatory variables.
func TestAmong_PruneWhenMandatoryEqualsKMax(t *testing.T) {
	model := NewModel()
	// S = {1,2}
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1") // subset of S -> mandatory
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{2, 3}), "x2") // mayIn
	x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{3, 4}), "x3") // disjoint
	// K encodes count+1; we want exactly 1 counted → K={2}
	k := model.NewVariableWithName(NewBitSetDomainFromValues(4, []int{2}), "K")

	among, err := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	if err != nil {
		t.Fatalf("NewAmong error: %v", err)
	}
	model.AddConstraint(among)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := solver.Solve(ctx, 0); err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	// With K=1 (encoded 2) and one mandatory, others must be OUT of S.
	d2 := solver.GetDomain(nil, x2.ID())
	d3 := solver.GetDomain(nil, x3.ID())
	if !d2.Equal(NewBitSetDomainFromValues(5, []int{3})) {
		t.Fatalf("x2 domain = %v, want {3}", d2)
	}
	if !d3.Equal(NewBitSetDomainFromValues(5, []int{3, 4})) {
		t.Fatalf("x3 domain = %v, want {3,4}", d3)
	}
}

// Basic pruning when K's lower bound equals the number of possible variables.
func TestAmong_ForceIntoSWhenPossibleEqualsKMin(t *testing.T) {
	model := NewModel()
	// S = {1}
	x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1") // mayIn
	x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 3}), "x2") // mayIn
	x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{2, 3}), "x3") // disjoint
	// possible=2 (x1,x2). Force count=2 → K={3}
	k := model.NewVariableWithName(NewBitSetDomainFromValues(4, []int{3}), "K")

	among, err := NewAmong([]*FDVariable{x1, x2, x3}, []int{1}, k)
	if err != nil {
		t.Fatalf("NewAmong error: %v", err)
	}
	model.AddConstraint(among)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := solver.Solve(ctx, 0); err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	// x1 and x2 must be forced to 1
	if d := solver.GetDomain(nil, x1.ID()); !d.Equal(NewBitSetDomainFromValues(5, []int{1})) {
		t.Fatalf("x1 domain = %v, want {1}", d)
	}
	if d := solver.GetDomain(nil, x2.ID()); !d.Equal(NewBitSetDomainFromValues(5, []int{1})) {
		t.Fatalf("x2 domain = %v, want {1}", d)
	}
	// x3 unchanged
	if d := solver.GetDomain(nil, x3.ID()); !d.Equal(NewBitSetDomainFromValues(5, []int{2, 3})) {
		t.Fatalf("x3 domain = %v, want {2,3}", d)
	}
}

// Inconsistency when mandatory exceeds K's maximum.
func TestAmong_Inconsistency(t *testing.T) {
	model := NewModel()
	// S = {1,2}; two mandatory, but K allows at most 1 ⇒ incompatibility
	x1 := model.NewVariable(NewBitSetDomainFromValues(5, []int{1})) // subset
	x2 := model.NewVariable(NewBitSetDomainFromValues(5, []int{2})) // subset
	x3 := model.NewVariable(NewBitSetDomainFromValues(5, []int{2, 3}))
	// K max 1 → domain {2} encodes count 1
	k := model.NewVariable(NewBitSetDomainFromValues(3, []int{2}))

	among, err := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	if err != nil {
		t.Fatalf("NewAmong error: %v", err)
	}
	model.AddConstraint(among)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sols, err := solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) != 0 {
		t.Fatalf("expected no solutions due to Among infeasibility, got %d", len(sols))
	}
}

// Constructor validation.
func TestAmong_ConstructorValidation(t *testing.T) {
	model := NewModel()
	v := model.NewVariable(NewBitSetDomain(5))
	k := model.NewVariable(NewBitSetDomain(4))

	if _, err := NewAmong(nil, []int{1}, k); err == nil {
		t.Fatalf("expected error for nil vars")
	}
	if _, err := NewAmong([]*FDVariable{v}, nil, k); err == nil {
		t.Fatalf("expected error for nil values")
	}
	if _, err := NewAmong([]*FDVariable{v}, []int{}, k); err == nil {
		t.Fatalf("expected error for empty values")
	}
	if _, err := NewAmong([]*FDVariable{v}, []int{-1, 0}, k); err == nil {
		t.Fatalf("expected error for non-positive values")
	}
}
