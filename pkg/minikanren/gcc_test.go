package minikanren

import (
	"context"
	"testing"
	"time"
)

// Basic pruning: if value 1 must occur exactly once and a variable is already
// fixed to 1, then remove 1 from all other variables.
func TestGCC_PruneSaturatedValue(t *testing.T) {
	model := NewModel()
	// Three vars with domain {1,2}
	a := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "a") // fixed to 1
	b := model.NewVariableWithName(NewBitSetDomain(2), "b")
	c := model.NewVariableWithName(NewBitSetDomain(2), "c")

	min := make([]int, 3)
	max := make([]int, 3)
	// Value 1 exactly once; value 2 unbounded up to 3
	min[1], max[1] = 1, 1
	min[2], max[2] = 0, 3

	gcc, err := NewGlobalCardinality([]*FDVariable{a, b, c}, min, max)
	if err != nil {
		t.Fatalf("NewGlobalCardinality error: %v", err)
	}
	model.AddConstraint(gcc)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	for _, v := range []*FDVariable{b, c} {
		dom := solver.GetDomain(nil, v.ID())
		want := NewBitSetDomainFromValues(dom.MaxValue(), []int{2})
		if !dom.Equal(want) {
			t.Fatalf("unexpected domain for %s: got %s want %s", v.Name(), dom.String(), want.String())
		}
	}
}

// Inconsistency: min requirement exceeds possible occurrences.
func TestGCC_Inconsistency(t *testing.T) {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{1}), "x")
	y := model.NewVariableWithName(NewBitSetDomainFromValues(2, []int{2}), "y")

	min := make([]int, 3)
	max := make([]int, 3)
	// Require two occurrences of value 1, but only one variable can take 1.
	min[1], max[1] = 2, 2
	min[2], max[2] = 0, 2

	gcc, err := NewGlobalCardinality([]*FDVariable{x, y}, min, max)
	if err != nil {
		t.Fatalf("NewGlobalCardinality error: %v", err)
	}
	model.AddConstraint(gcc)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sols, err := solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) != 0 {
		t.Fatalf("expected no solutions due to GCC infeasibility, got %d", len(sols))
	}
}

// Constructor validation: bounds length and feasibility checks.
func TestGCC_ConstructorValidation(t *testing.T) {
	model := NewModel()
	v := model.NewVariable(NewBitSetDomain(3))

	if _, err := NewGlobalCardinality(nil, nil, nil); err == nil {
		t.Fatalf("expected error for empty vars")
	}
	// Too short bounds
	if _, err := NewGlobalCardinality([]*FDVariable{v}, make([]int, 2), make([]int, 2)); err == nil {
		t.Fatalf("expected error for short bounds")
	}
	// min > max
	min := make([]int, 4)
	max := make([]int, 4)
	min[1], max[1] = 2, 1
	if _, err := NewGlobalCardinality([]*FDVariable{v}, min, max); err == nil {
		t.Fatalf("expected error for min>max")
	}
	// sum(min) > n
	min = make([]int, 4)
	max = make([]int, 4)
	min[1], max[1] = 2, 2
	min[2], max[2] = 2, 2
	if _, err := NewGlobalCardinality([]*FDVariable{v}, min, max); err == nil {
		t.Fatalf("expected error for sum(min) > n")
	}
}
