package minikanren

import (
	"context"
	"testing"
)

func TestMinMax_ConstructorValidation(t *testing.T) {
	model := NewModel()
	a := model.NewVariable(NewBitSetDomain(9))
	r := model.NewVariable(NewBitSetDomain(9))

	if _, err := NewMin(nil, r); err == nil {
		t.Fatalf("expected error for NewMin with nil vars")
	}
	if _, err := NewMin([]*FDVariable{a}, nil); err == nil {
		t.Fatalf("expected error for NewMin with nil result")
	}
	if _, err := NewMax(nil, r); err == nil {
		t.Fatalf("expected error for NewMax with nil vars")
	}
	if _, err := NewMax([]*FDVariable{a}, nil); err == nil {
		t.Fatalf("expected error for NewMax with nil result")
	}
}

func TestMin_BasicPropagation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(3).RemoveAbove(6))
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(5).RemoveAbove(7))
	r := model.NewVariable(NewBitSetDomain(9))

	c, err := NewMin([]*FDVariable{x, y}, r)
	if err != nil {
		t.Fatalf("NewMin: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	// Propagate at root by solving for 0 solutions
	_, err = solver.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	dr := solver.GetDomain(nil, r.ID())
	if dr.Min() != 3 || dr.Max() != 6 {
		t.Fatalf("expected r in [3..6], got [%d..%d]", dr.Min(), dr.Max())
	}
	dx := solver.GetDomain(nil, x.ID())
	if dx.Min() != 3 {
		t.Fatalf("expected x.min >= 3, got %d", dx.Min())
	}
	dy := solver.GetDomain(nil, y.ID())
	if dy.Min() != 5 {
		t.Fatalf("expected y.min >= 5, got %d", dy.Min())
	}

	// Now force r >= 6; x should be pruned to >= 6 as well
	r.SetDomain(NewBitSetDomain(9).RemoveBelow(6))
	solver2 := NewSolver(model)
	_, err = solver2.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	dx2 := solver2.GetDomain(nil, x.ID())
	if dx2.Min() != 6 {
		t.Fatalf("expected x.min >= 6 after r>=6, got %d", dx2.Min())
	}
}

func TestMax_BasicPropagation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(2).RemoveAbove(4))
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(6).RemoveAbove(8))
	r := model.NewVariable(NewBitSetDomain(9))

	c, err := NewMax([]*FDVariable{x, y}, r)
	if err != nil {
		t.Fatalf("NewMax: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	_, err = solver.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	dr := solver.GetDomain(nil, r.ID())
	if dr.Min() != 6 || dr.Max() != 8 {
		t.Fatalf("expected r in [6..8], got [%d..%d]", dr.Min(), dr.Max())
	}
	dx := solver.GetDomain(nil, x.ID())
	if dx.Max() != 4 {
		t.Fatalf("expected x.max remain 4, got %d", dx.Max())
	}
	dy := solver.GetDomain(nil, y.ID())
	if dy.Max() != 8 {
		t.Fatalf("expected y.max remain 8, got %d", dy.Max())
	}

	// Now force r <= 6; y should be pruned to <= 6 as well
	r.SetDomain(NewBitSetDomain(9).RemoveAbove(6))
	solver2 := NewSolver(model)
	_, err = solver2.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	dy2 := solver2.GetDomain(nil, y.ID())
	if dy2.Max() != 6 {
		t.Fatalf("expected y.max <= 6 after r<=6, got %d", dy2.Max())
	}
}
