package minikanren

import "testing"

// TestElementValues_BasicPropagation validates bidirectional pruning.
func TestElementValues_BasicPropagation(t *testing.T) {
	model := NewModel()

	// index in [1..5]
	idx := model.NewVariable(NewBitSetDomain(5))
	// result in [1..9]
	res := model.NewVariable(NewBitSetDomain(9))

	values := []int{3, 5, 5, 7, 9}
	c, err := NewElementValues(idx, values, res)
	if err != nil {
		t.Fatalf("NewElementValues failed: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	// Narrow result to {5,7}
	state, _ = solver.SetDomain(state, res.ID(), NewBitSetDomainFromValues(9, []int{5, 7}))

	newState, err := c.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Expected: indices {2,3,4} allowed since values[2]=5, values[3]=5, values[4]=7
	idxDom := solver.GetDomain(newState, idx.ID())
	wantIdx := NewBitSetDomainFromValues(5, []int{2, 3, 4})
	if !idxDom.Equal(wantIdx) {
		t.Errorf("unexpected idx domain: got %v, want %v", idxDom, wantIdx)
	}

	// Result should remain {5,7}
	resDom := solver.GetDomain(newState, res.ID())
	wantRes := NewBitSetDomainFromValues(9, []int{5, 7})
	if !resDom.Equal(wantRes) {
		t.Errorf("unexpected res domain: got %v, want %v", resDom, wantRes)
	}
}

// TestElementValues_IndexClamping ensures index is clamped to [1..n].
func TestElementValues_IndexClamping(t *testing.T) {
	model := NewModel()
	idx := model.NewVariable(NewBitSetDomain(20)) // overly wide
	res := model.NewVariable(NewBitSetDomain(50))
	values := []int{10, 20, 30}
	c, err := NewElementValues(idx, values, res)
	if err != nil {
		t.Fatalf("NewElementValues failed: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	state := (*SolverState)(nil)

	newState, err := c.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}
	idxDom := solver.GetDomain(newState, idx.ID())
	if idxDom.Min() != 1 || idxDom.Max() != 3 {
		t.Errorf("idx bounds not clamped: got [%d,%d], want [1,3]", idxDom.Min(), idxDom.Max())
	}
	// Result should be pruned to {10,20,30}
	resDom := solver.GetDomain(newState, res.ID())
	wantRes := NewBitSetDomainFromValues(50, []int{10, 20, 30})
	if !resDom.Equal(wantRes) {
		t.Errorf("unexpected res domain: got %v, want %v", resDom, wantRes)
	}
}

// TestElementValues_FixedIndexForcesResult ensures singleton index forces singleton result.
func TestElementValues_FixedIndexForcesResult(t *testing.T) {
	model := NewModel()
	idx := model.NewVariable(NewBitSetDomainFromValues(10, []int{4})) // fixed to 4
	res := model.NewVariable(NewBitSetDomain(100))
	values := []int{3, 7, 11, 42, 99}
	c, _ := NewElementValues(idx, values, res)
	model.AddConstraint(c)

	solver := NewSolver(model)
	state := (*SolverState)(nil)
	newState, err := c.Propagate(solver, state)
	if err != nil {
		t.Fatalf("propagation error: %v", err)
	}
	resDom := solver.GetDomain(newState, res.ID())
	want := NewBitSetDomainFromValues(100, []int{42})
	if !resDom.Equal(want) {
		t.Errorf("result not forced to 42: got %v", resDom)
	}
}

// TestElementValues_Inconsistency ensures incompatible result empties index.
func TestElementValues_Inconsistency(t *testing.T) {
	model := NewModel()
	idx := model.NewVariable(NewBitSetDomain(4))
	res := model.NewVariable(NewBitSetDomainFromValues(10, []int{1})) // result forced to 1
	values := []int{2, 2, 2, 2}                                       // never 1
	c, _ := NewElementValues(idx, values, res)
	model.AddConstraint(c)

	solver := NewSolver(model)
	state := (*SolverState)(nil)
	_, err := c.Propagate(solver, state)
	if err == nil {
		t.Fatalf("expected inconsistency, got nil")
	}
}
