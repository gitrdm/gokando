package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleSendMoreMoney_reificationCount demonstrates how to model
// key pieces of the cryptarithm SEND + MORE = MONEY using FD reification
// and Count, without encoding the full columnar arithmetic. We:
//   - Use shifted encoding for digits 0..9 as FD values 1..10
//   - Declare no-leading-zeros via Count on [S, M]
//   - Reify M = 1 (encoded as M == 2) and force the boolean to true
//   - Keep letters AllDifferent
//
// The goal is to show declarative modeling and local pruning effects.
func Example_sendMoreMoney_reificationCount() {
	model := NewModel()

	// Digits 0..9 → FD values 1..10 (we use HLAPI IntVar/IntVarValues below)

	// Letter variables (encoded digits)
	// low-level: S := model.NewVariable(digits)
	S := model.IntVar(1, 10, "S")
	// low-level: E := model.NewVariable(digits)
	E := model.IntVar(1, 10, "E")
	// low-level: N := model.NewVariable(digits)
	N := model.IntVar(1, 10, "N")
	// low-level: D := model.NewVariable(digits)
	D := model.IntVar(1, 10, "D")
	// low-level: M := model.NewVariable(digits)
	M := model.IntVar(1, 10, "M")
	// low-level: O := model.NewVariable(digits)
	O := model.IntVar(1, 10, "O")
	// low-level: R := model.NewVariable(digits)
	R := model.IntVar(1, 10, "R")
	// low-level: Y := model.NewVariable(digits)
	Y := model.IntVar(1, 10, "Y")

	// All letters must be distinct
	ad, err := NewAllDifferent([]*FDVariable{S, E, N, D, M, O, R, Y})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ad)

	// 1) No leading zeros: S and M cannot be digit 0 (encoded as FD value 1)
	//    Count([S, M], target=1) must be 0 → encoded countVar = 1 (0+1)
	// low-level: countVar := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))
	// The countVar is encoded as count+1; to force count==0 we set countVar to {1}.
	// low-level: countVar := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "countVar")
	// Use the low-level constructor here to preserve the original universe size
	// (NewCount expects the countVar's domain MaxValue() to be >= len(vars)+1).
	countVar := model.NewVariableWithName(NewBitSetDomainFromValues(10, []int{1}), "countVar")
	if _, err := NewCount(model, []*FDVariable{S, M}, 1, countVar); err != nil {
		panic(err)
	}

	// 2) Reify M = digit 1 (common fact): encoded M == 2; force boolean to true ({2})
	// low-level: bM := model.NewVariable(NewBitSetDomainFromValues(10, []int{2})) // {2} means true
	bM := model.IntVarValues([]int{2}, "bM") // {2} means true
	reif, err := NewValueEqualsReified(M, 2, bM)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(reif)

	// Propagate and inspect domains
	solver := NewSolver(model)
	// We don't need all solutions; propagation + first solution is enough for inspection
	sols, _ := solver.Solve(context.Background(), 1)

	mDom := solver.GetDomain(nil, M.ID())
	sDom := solver.GetDomain(nil, S.ID())
	fmt.Printf("solutions: %d\n", len(sols))
	fmt.Printf("M singleton and equals 2: %v %v\n", mDom.IsSingleton(), mDom.SingletonValue() == 2)
	fmt.Printf("S allows zero? %v\n", sDom.Has(1))

	// Output:
	// solutions: 1
	// M singleton and equals 2: true true
	// S allows zero? false
}
