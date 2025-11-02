package minikanren

import "testing"

// buildEndsWith1DFA returns a DFA over symbols {1,2} that accepts
// exactly those strings whose last symbol is 1.
func buildEndsWith1DFA() (numStates, start int, accept []int, delta [][]int) {
	// States: 1=start, 2=last=1, 3=last=2; accept={2}
	numStates, start = 3, 1
	accept = []int{2}
	// delta rows sized to alphabetMax+1=3 (index 0 unused)
	delta = [][]int{
		// s=1
		{0, 2, 3},
		// s=2
		{0, 2, 3},
		// s=3
		{0, 2, 3},
	}
	return
}

// buildSeq12DFA returns a DFA that accepts exactly the sequence [1,2]
// for length-2 words over symbols {1,2}. Used to test strong pruning.
func buildSeq12DFA() (numStates, start int, accept []int, delta [][]int) {
	// States: 1=start, 2=seen1, 3=dead; accept at position 2 is state 2 then on 2 -> accept state 2? Simpler: model two-step DFA
	numStates, start = 3, 1
	accept = []int{2} // Accepting after reading second symbol must land in 2
	delta = [][]int{
		// from s=1: on 1 -> 2, on 2 -> 3 (dead)
		{0, 2, 3},
		// from s=2: on 1 -> 3 (dead), on 2 -> 2 (stay accept)
		{0, 3, 2},
		// from s=3 (dead): both -> 3
		{0, 3, 3},
	}
	return
}

func TestRegular_PrunesEndsWith1(t *testing.T) {
	num, start, acc, delta := buildEndsWith1DFA()
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomain(2))
	x2 := model.NewVariable(NewBitSetDomain(2))
	x3 := model.NewVariable(NewBitSetDomain(2))

	c, err := NewRegular([]*FDVariable{x1, x2, x3}, num, start, acc, delta)
	if err != nil {
		t.Fatalf("NewRegular error: %v", err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	st, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate error: %v", err)
	}
	// x3 must be {1}
	d3 := solver.GetDomain(st, x3.ID())
	if !d3.IsSingleton() || d3.SingletonValue() != 1 {
		t.Fatalf("expected x3 to be {1}, got %s", d3.String())
	}
	// x1,x2 remain {1,2}
	if !solver.GetDomain(st, x1.ID()).Equal(NewBitSetDomain(2)) {
		t.Fatalf("unexpected x1 domain: %s", solver.GetDomain(st, x1.ID()).String())
	}
	if !solver.GetDomain(st, x2.ID()).Equal(NewBitSetDomain(2)) {
		t.Fatalf("unexpected x2 domain: %s", solver.GetDomain(st, x2.ID()).String())
	}
}

func TestRegular_InconsistencyWhenNoAccept(t *testing.T) {
	num, start, acc, delta := buildEndsWith1DFA()
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomain(2))
	x2 := model.NewVariable(NewBitSetDomain(2))
	// x3 forbids 1, making acceptance impossible
	x3 := model.NewVariable(NewBitSetDomainFromValues(2, []int{2}))

	c, err := NewRegular([]*FDVariable{x1, x2, x3}, num, start, acc, delta)
	if err != nil {
		t.Fatalf("NewRegular error: %v", err)
	}
	model.AddConstraint(c)
	solver := NewSolver(model)
	if _, err := solver.propagate(nil); err == nil {
		t.Fatalf("expected inconsistency but got no error")
	}
}

func TestRegular_StrongPruningSequence12(t *testing.T) {
	num, start, acc, delta := buildSeq12DFA()
	model := NewModel()
	x1 := model.NewVariable(NewBitSetDomain(2))
	x2 := model.NewVariable(NewBitSetDomain(2))

	c, err := NewRegular([]*FDVariable{x1, x2}, num, start, acc, delta)
	if err != nil {
		t.Fatalf("NewRegular error: %v", err)
	}
	model.AddConstraint(c)
	solver := NewSolver(model)
	st, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate error: %v", err)
	}
	d1 := solver.GetDomain(st, x1.ID())
	d2 := solver.GetDomain(st, x2.ID())
	if !d1.IsSingleton() || d1.SingletonValue() != 1 {
		t.Fatalf("x1 expected {1}, got %s", d1.String())
	}
	if !d2.IsSingleton() || d2.SingletonValue() != 2 {
		t.Fatalf("x2 expected {2}, got %s", d2.String())
	}
}
