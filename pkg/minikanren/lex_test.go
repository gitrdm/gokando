package minikanren

import (
	"context"
	"testing"
	"time"
)

func TestLexLessEq_BasicPropagation(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(9, []int{2, 3, 4}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(9, []int{1, 2, 3}))
	y1 := model.NewVariable(NewBitSetDomainFromValues(9, []int{3, 4, 5}))
	y2 := model.NewVariable(NewBitSetDomainFromValues(9, []int{2, 3, 4}))

	c, err := NewLexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	if err != nil {
		t.Fatalf("NewLexLessEq error: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}

	// With eq-prefix at i=0, we should prune x1 > max(y1)=5 (none) and y1 < min(x1)=2
	gotY1 := solver.GetDomain(nil, y1.ID())
	wantY1 := NewBitSetDomainFromValues(9, []int{3, 4, 5})
	if !gotY1.Equal(wantY1) {
		t.Fatalf("y1 domain = %v, want %v", gotY1, wantY1)
	}
}

func TestLexLess_Strict_AllEqualInconsistent(t *testing.T) {
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomainFromValues(9, []int{3}))
	x2 := model.NewVariable(NewBitSetDomainFromValues(9, []int{2}))
	y1 := model.NewVariable(NewBitSetDomainFromValues(9, []int{3}))
	y2 := model.NewVariable(NewBitSetDomainFromValues(9, []int{2}))

	c, err := NewLexLess([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	if err != nil {
		t.Fatalf("NewLexLess error: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sols, err := solver.Solve(ctx, 0)
	if err != nil {
		t.Fatalf("Solve error: %v", err)
	}
	if len(sols) != 0 {
		t.Fatalf("expected no solutions due to strict lex all-equal conflict, got %d", len(sols))
	}
}

func TestLex_ConstructorValidation(t *testing.T) {
	model := NewModel()
	v := model.NewVariable(NewBitSetDomain(5))
	if _, err := NewLexLess(nil, []*FDVariable{v}); err == nil {
		t.Fatalf("expected error for nil xs")
	}
	if _, err := NewLexLess([]*FDVariable{v}, nil); err == nil {
		t.Fatalf("expected error for nil ys")
	}
	if _, err := NewLexLess([]*FDVariable{v}, []*FDVariable{}); err == nil {
		t.Fatalf("expected error for empty ys")
	}
	if _, err := NewLexLess([]*FDVariable{v}, []*FDVariable{v, v}); err == nil {
		t.Fatalf("expected error for length mismatch")
	}
}
