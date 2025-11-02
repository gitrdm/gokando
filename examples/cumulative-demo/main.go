package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

// Cumulative Demo: simple resource scheduling with capacity
//
// Three tasks with fixed durations and demands share a resource of capacity 3.
// We use the Cumulative global constraint to ensure that at any time the sum
// of demands of running tasks does not exceed 3. We enumerate a few feasible
// start-time assignments.
func main() {
	fmt.Println("=== Cumulative Constraint Demo (capacity=3) ===")

	// Tasks: durations and demands
	durations := []int{2, 2, 3}
	demands := []int{2, 1, 1}
	capacity := 3

	// Horizon: discrete time 1..5
	model := mk.NewModel()
	starts := make([]*mk.FDVariable, 3)
	for i := 0; i < 3; i++ {
		starts[i] = model.NewVariableWithName(mk.NewBitSetDomain(5), fmt.Sprintf("S%d", i+1))
	}

	cum, err := mk.NewCumulative(starts, durations, demands, capacity)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(cum)

	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 50)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(solutions) == 0 {
		fmt.Println("No feasible schedules found")
		return
	}

	fmt.Printf("Found %d feasible schedules (showing up to %d):\n", len(solutions), 50)
	for _, sol := range solutions {
		for i, v := range starts {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("S%d=%d", i+1, sol[v.ID()])
		}
		fmt.Println()
	}
}
