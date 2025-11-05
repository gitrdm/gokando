package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleNewAmong demonstrates basic pruning with the Among constraint.
// We model S={1,2,3} over three variables. x1 is already restricted to S,
// and we set K to encode exactly 1 counted variable. The constraint then
// forces all other variables that could take values in S to be OUT of S.
//
// HLAPI note: this example uses the HLAPI wrapper `model.Among(...)` which
// posts the constraint concisely; the equivalent low-level constructor
// (`NewAmong`) is kept as a comment for educational clarity.
func ExampleNewAmong() {
	model := NewModel()

	// Low-level API (kept as comments):
	// x1 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x1 := model.IntVarValues([]int{1, 2}, "x1")
	// x2 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{2, 3}), "x2")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	// x3 := model.NewVariableWithName(NewBitSetDomainFromValues(5, []int{3, 4}), "x3")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	// K encodes count+1; here we want exactly 1 variable in S → K={2}
	// k := model.NewVariableWithName(NewBitSetDomainFromValues(4, []int{2}), "K")
	k := model.IntVarValues([]int{2}, "K")

	// S = {1,2}. With K=1 (encoded 2) and x1⊆S, x2 is forced OUT of S
	// Low-level API (kept as comment):
	// c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	// model.AddConstraint(c)
	// HLAPI:
	_ = model.Among([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Printf("x2: %s\n", solver.GetDomain(nil, x2.ID()))
	fmt.Printf("x3: %s\n", solver.GetDomain(nil, x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}

// ExampleNewAmong_hybrid demonstrates the same Among constraint solved via the
// hybrid `UnifiedStore` + `HybridSolver` path. This demonstrates how the FD
// propagation engine can be invoked inside the hybrid framework and how the
// UnifiedStore holds FD domains and constraints for plugin-based propagation.
func ExampleNewAmong_hybrid() {
	model := NewModel()

	x1 := model.IntVarValues([]int{1, 2}, "x1")
	x2 := model.IntVarValues([]int{2, 3}, "x2")
	x3 := model.IntVarValues([]int{3, 4}, "x3")
	k := model.IntVarValues([]int{2}, "K")

	// Build the propagation constraint and register it with the model so the FD plugin can discover it.
	c, _ := NewAmong([]*FDVariable{x1, x2, x3}, []int{1, 2}, k)
	model.AddConstraint(c)

	// Use HLAPI helper to build a HybridSolver and a UnifiedStore populated
	// from the model (domains + constraints). This reduces boilerplate.
	solver, store, err := NewHybridSolverFromModel(model)
	if err != nil {
		panic(err)
	}

	// Run propagation to a fixed point.
	result, _ := solver.Propagate(store)

	fmt.Printf("x2: %s\n", result.GetDomain(x2.ID()))
	fmt.Printf("x3: %s\n", result.GetDomain(x3.ID()))
	// Output:
	// x2: {3}
	// x3: {3..4}
}
