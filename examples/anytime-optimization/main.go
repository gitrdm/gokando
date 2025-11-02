package main

import (
	"context"
	"errors"
	"fmt"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

// Anytime Optimization Demo: demonstrates SolveOptimalWithOptions returning
// an incumbent when a node limit is reached, along with ErrSearchLimitReached.
//
// This shows the "anytime" property: the solver can be interrupted at any point
// and will return the best solution found so far, proving useful for large
// instances or when you need a quick approximation.
func main() {
	fmt.Println("=== Anytime Optimization Demo ===")
	fmt.Println()

	// Simple model: minimize X + 2*Y where X,Y ∈ {1..5}
	model := mk.NewModel()
	x := model.NewVariableWithName(mk.NewBitSetDomain(5), "X")
	y := model.NewVariableWithName(mk.NewBitSetDomain(5), "Y")
	total := model.NewVariable(mk.NewBitSetDomain(15))

	ls, err := mk.NewLinearSum([]*mk.FDVariable{x, y}, []int{1, 2}, total)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(ls)

	solver := mk.NewSolver(model)

	// Run with a very tight node limit (e.g., 3 leaves) to trigger early stop
	fmt.Println("Minimizing X + 2*Y with a node limit of 3...")
	sol, objVal, err := solver.SolveOptimalWithOptions(
		context.Background(),
		total,
		true, // minimize
		mk.WithNodeLimit(3),
	)

	if err != nil {
		if errors.Is(err, mk.ErrSearchLimitReached) {
			fmt.Printf("✓ Node limit reached. Best incumbent found: objective = %d\n", objVal)
			if sol != nil {
				fmt.Printf("  Solution: X=%d, Y=%d\n", sol[x.ID()], sol[y.ID()])
			} else {
				fmt.Println("  (No incumbent found before limit)")
			}
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	} else {
		// If the limit wasn't reached, we found the optimum early
		fmt.Printf("✓ Optimal solution: objective = %d\n", objVal)
		fmt.Printf("  Solution: X=%d, Y=%d\n", sol[x.ID()], sol[y.ID()])
	}

	fmt.Println()
	fmt.Println("Now solving without limit to confirm the true optimum...")
	solOpt, objOpt, errOpt := solver.SolveOptimal(context.Background(), total, true)
	if errOpt != nil {
		fmt.Printf("Error: %v\n", errOpt)
		return
	}
	fmt.Printf("✓ True optimum: objective = %d (X=%d, Y=%d)\n",
		objOpt, solOpt[x.ID()], solOpt[y.ID()])
	fmt.Println()
	fmt.Println("This demonstrates anytime optimization: you can stop early and")
	fmt.Println("still get a valid (though possibly suboptimal) solution.")
}
