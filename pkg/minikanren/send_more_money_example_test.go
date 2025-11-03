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

	// Digits 0..9 → FD values 1..10
	digits := NewBitSetDomain(10)

	// Letter variables (encoded digits)
	S := model.NewVariable(digits)
	E := model.NewVariable(digits)
	N := model.NewVariable(digits)
	D := model.NewVariable(digits)
	M := model.NewVariable(digits)
	O := model.NewVariable(digits)
	R := model.NewVariable(digits)
	Y := model.NewVariable(digits)

	// All letters must be distinct
	ad, err := NewAllDifferent([]*FDVariable{S, E, N, D, M, O, R, Y})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ad)

	// 1) No leading zeros: S and M cannot be digit 0 (encoded as FD value 1)
	//    Count([S, M], target=1) must be 0 → encoded countVar = 1 (0+1)
	countVar := model.NewVariable(NewBitSetDomainFromValues(10, []int{1}))
	if _, err := NewCount(model, []*FDVariable{S, M}, 1, countVar); err != nil {
		panic(err)
	}

	// 2) Reify M = digit 1 (common fact): encoded M == 2; force boolean to true ({2})
	bM := model.NewVariable(NewBitSetDomainFromValues(10, []int{2})) // {2} means true
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
