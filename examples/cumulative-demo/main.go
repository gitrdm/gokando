package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Cumulative Demo: simple resource scheduling with capacity
//
// Three tasks with fixed durations and demands share a resource of capacity 3.
// We use the Cumulative global constraint to ensure that at any time the sum
// of demands of running tasks does not exceed 3.
//
// Part 1: Enumerate a few feasible start-time assignments.
// Part 2: Minimize the makespan (latest end time) using SolveOptimal.
func main() {
	fmt.Println("=== Cumulative Constraint Demo (capacity=3) ===")
	fmt.Println()

	// Tasks: durations and demands
	durations := []int{2, 2, 3}
	demands := []int{2, 1, 1}
	capacity := 3

	// Horizon: discrete time 1..8 (allow enough slack for makespan)
	model := mk.NewModel()
	starts := make([]*mk.FDVariable, 3)
	for i := 0; i < 3; i++ {
		starts[i] = model.NewVariableWithName(mk.NewBitSetDomain(8), fmt.Sprintf("S%d", i+1))
	}

	cum, err := mk.NewCumulative(starts, durations, demands, capacity)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(cum)

	// Part 1: Enumerate feasible schedules
	fmt.Println("Part 1: Enumerating feasible schedules...")
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(solutions) == 0 {
		fmt.Println("No feasible schedules found")
		return
	}

	fmt.Printf("Found %d feasible schedules (showing up to 10):\n", len(solutions))
	for _, sol := range solutions {
		for i, v := range starts {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("S%d=%d", i+1, sol[v.ID()])
		}
		fmt.Println()
	}
	fmt.Println()

	// Part 2: Minimize makespan
	fmt.Println("Part 2: Minimizing makespan (latest end time)...")
	// Create a fresh model for optimization (could reuse, but clean separation is safer)
	modelOpt := mk.NewModel()
	startsOpt := make([]*mk.FDVariable, 3)
	for i := 0; i < 3; i++ {
		startsOpt[i] = modelOpt.NewVariableWithName(mk.NewBitSetDomain(8), fmt.Sprintf("S%d", i+1))
	}

	cumOpt, err := mk.NewCumulative(startsOpt, durations, demands, capacity)
	if err != nil {
		panic(err)
	}
	modelOpt.AddConstraint(cumOpt)

	// End times: e_i = s_i + dur_i - 1 (inclusive range)
	ends := make([]*mk.FDVariable, 3)
	for i := 0; i < 3; i++ {
		ends[i] = modelOpt.NewVariableWithName(mk.NewBitSetDomain(10), fmt.Sprintf("E%d", i+1))
		arith, err := mk.NewArithmetic(startsOpt[i], ends[i], durations[i]-1)
		if err != nil {
			panic(err)
		}
		modelOpt.AddConstraint(arith)
	}

	// Makespan M >= e_i for all i
	makespan := modelOpt.NewVariableWithName(mk.NewBitSetDomain(10), "M")
	for i := 0; i < 3; i++ {
		ineq, err := mk.NewInequality(makespan, ends[i], mk.GreaterEqual)
		if err != nil {
			panic(err)
		}
		modelOpt.AddConstraint(ineq)
	}

	solverOpt := mk.NewSolver(modelOpt)
	solOpt, objVal, err := solverOpt.SolveOptimal(context.Background(), makespan, true)
	if err != nil {
		fmt.Printf("SolveOptimal error: %v\n", err)
		return
	}
	if solOpt == nil {
		fmt.Println("No solution found (infeasible)")
		return
	}

	fmt.Printf("✓ Minimum makespan: %d\n", objVal)
	fmt.Print("  Optimal schedule: ")
	for i, v := range startsOpt {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("S%d=%d", i+1, solOpt[v.ID()])
	}
	fmt.Println()
	fmt.Print("  End times:        ")
	for i, v := range ends {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("E%d=%d", i+1, solOpt[v.ID()])
	}
	fmt.Println()
}
