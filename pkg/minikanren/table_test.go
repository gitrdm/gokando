package minikanren

import "testing"

func TestTable_BasicPropagation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	c, err := NewTable([]*FDVariable{x, y}, rows)
	if err != nil {
		t.Fatalf("NewTable error: %v", err)
	}

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Narrow y to {1,2}
	initY := NewBitSetDomainFromValues(5, []int{1, 2})
	state, _ = solver.SetDomain(state, y.ID(), initY)

	newState, err := c.Propagate(solver, state)
	if err != nil {
		t.Fatalf("Propagate error: %v", err)
	}

	xd := solver.GetDomain(newState, x.ID())
	yd := solver.GetDomain(newState, y.ID())

	expX := NewBitSetDomainFromValues(5, []int{1, 3})
	if !xd.Equal(expX) {
		t.Fatalf("x domain = %v, want %v", xd, expX)
	}
	expY := NewBitSetDomainFromValues(5, []int{1, 2})
	if !yd.Equal(expY) {
		t.Fatalf("y domain = %v, want %v", yd, expY)
	}
}

func TestTable_Inconsistency(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))

	rows := [][]int{
		{1, 1},
		{2, 3},
		{3, 2},
	}
	c, err := NewTable([]*FDVariable{x, y}, rows)
	if err != nil {
		t.Fatalf("NewTable error: %v", err)
	}

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Force y to {3} and x to {1} -> no compatible row
	state, _ = solver.SetDomain(state, y.ID(), NewBitSetDomainFromValues(5, []int{3}))
	state, _ = solver.SetDomain(state, x.ID(), NewBitSetDomainFromValues(5, []int{1}))

	if _, err := c.Propagate(solver, state); err == nil {
		t.Fatalf("expected inconsistency, got nil error")
	}
}

func TestNewTable_Validation(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))

	// Empty vars
	if _, err := NewTable(nil, [][]int{{1, 2}}); err == nil {
		t.Errorf("expected error for empty vars")
	}
	// Nil var
	if _, err := NewTable([]*FDVariable{x, nil}, [][]int{{1, 2}}); err == nil {
		t.Errorf("expected error for nil var")
	}
	// Empty rows
	if _, err := NewTable([]*FDVariable{x, y}, nil); err == nil {
		t.Errorf("expected error for empty rows")
	}
	// Arity mismatch
	if _, err := NewTable([]*FDVariable{x, y}, [][]int{{1}}); err == nil {
		t.Errorf("expected error for arity mismatch")
	}
	// Non-positive value
	if _, err := NewTable([]*FDVariable{x, y}, [][]int{{0, 1}}); err == nil {
		t.Errorf("expected error for non-positive value")
	}
}
