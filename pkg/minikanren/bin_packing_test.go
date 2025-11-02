package minikanren

import "testing"

// TestBinPacking_BasicPruning validates that capacities prune item-to-bin domains
// using the weighted boolean sum encoding with load+1 arithmetic ties.
func TestBinPacking_BasicPruning(t *testing.T) {
	model := NewModel()

	// Three items with sizes [2,2,1]; two bins with capacities [4,1]
	//  - Bin2 capacity=1 can only host the size-1 item (x3)
	//  - Bin1 capacity=4 can host both size-2 items but not also x3
	//  ⇒ x1=1, x2=1, x3=2 forced by propagation
	bdom := NewBitSetDomain(2) // bins {1,2}
	x1 := model.NewVariable(bdom)
	x2 := model.NewVariable(bdom)
	x3 := model.NewVariable(bdom)

	sizes := []int{2, 2, 1}
	capacities := []int{4, 1}
	_, err := NewBinPacking(model, []*FDVariable{x1, x2, x3}, sizes, capacities)
	if err != nil {
		t.Fatalf("NewBinPacking failed: %v", err)
	}

	solver := NewSolver(model)
	state, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagate failed: %v", err)
	}

	d1 := solver.GetDomain(state, x1.ID())
	d2 := solver.GetDomain(state, x2.ID())
	d3 := solver.GetDomain(state, x3.ID())

	if !d1.Equal(NewBitSetDomainFromValues(2, []int{1})) {
		t.Fatalf("x1 domain mismatch: got %s want {1}", d1.String())
	}
	if !d2.Equal(NewBitSetDomainFromValues(2, []int{1})) {
		t.Fatalf("x2 domain mismatch: got %s want {1}", d2.String())
	}
	if !d3.Equal(NewBitSetDomainFromValues(2, []int{2})) {
		t.Fatalf("x3 domain mismatch: got %s want {2}", d3.String())
	}
}
