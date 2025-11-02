package minikanren

import (
	"fmt"
)

func ExampleNewBinPacking() {
	model := NewModel()

	// Items: sizes [2,2,1], bins: 2 with capacities [4,1]
	bdom := NewBitSetDomain(2) // bins {1,2}
	x1 := model.NewVariableWithName(bdom, "x1")
	x2 := model.NewVariableWithName(bdom, "x2")
	x3 := model.NewVariableWithName(bdom, "x3")

	sizes := []int{2, 2, 1}
	capacities := []int{4, 1}

	_, _ = NewBinPacking(model, []*FDVariable{x1, x2, x3}, sizes, capacities)

	solver := NewSolver(model)
	st, _ := solver.propagate(nil)

	// After propagation:
	//  - Bin 2 (cap=1) can only host size-1 ⇒ x3=2
	//  - Bin 1 (cap=4) must host both size-2 ⇒ x1=1, x2=1
	fmt.Printf("x1: %s\n", solver.GetDomain(st, x1.ID()))
	fmt.Printf("x2: %s\n", solver.GetDomain(st, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(st, x3.ID()))
	// Output:
	// x1: {1}
	// x2: {1}
	// x3: {2}
}
