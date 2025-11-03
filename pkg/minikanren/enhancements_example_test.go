package minikanren

import (
	"context"
	"fmt"
	"time"
)

// ExampleLinearSum_mixedSign demonstrates LinearSum with negative coefficients for profit maximization.
func ExampleLinearSum_mixedSign() {
	m := NewModel()
	// Profit model: revenue - cost = profit
	// revenue = 10*units, cost = 3*units, profit = 7*units
	// Or more realistically: profit = 5*productA - 2*productB
	productA := m.NewVariable(NewBitSetDomain(3))
	productB := m.NewVariable(NewBitSetDomain(3))
	profit := m.NewVariable(NewBitSetDomain(20))

	// Maximize: 5*A - 2*B
	ls, _ := NewLinearSum([]*FDVariable{productA, productB}, []int{5, -2}, profit)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Find maximum profit
	sol, objVal, _ := solver.SolveOptimal(ctx, profit, false) // maximize
	fmt.Printf("Maximum profit: %d (A=%d, B=%d)\n", objVal, sol[productA.ID()], sol[productB.ID()])
	// Output: Maximum profit: 13 (A=3, B=1)
}

// ExampleSolver_SolveOptimal_impactHeuristic demonstrates optimization-aware variable ordering.
func ExampleSolver_SolveOptimal_impactHeuristic() {
	m := NewModel()
	// Minimize total cost: cost = 2*x + 3*y
	x := m.NewVariable(NewBitSetDomain(4))
	y := m.NewVariable(NewBitSetDomain(4))
	cost := m.NewVariable(NewBitSetDomain(30))

	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{2, 3}, cost)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Use impact-based heuristic to focus on objective-relevant variables
	sol, objVal, _ := solver.SolveOptimalWithOptions(ctx, cost, true,
		WithHeuristics(HeuristicImpact, ValueOrderObjImproving, 42))

	fmt.Printf("Minimum cost: %d (x=%d, y=%d)\n", objVal, sol[x.ID()], sol[y.ID()])
	// Output: Minimum cost: 5 (x=1, y=1)
}

// ExampleSolver_SolveOptimal_boolSum demonstrates optimizing a count objective.
func ExampleSolver_SolveOptimal_boolSum() {
	m := NewModel()
	// Maximize the number of satisfied conditions (booleans set to true)
	b1 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b2 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	b3 := m.NewVariable(NewBitSetDomainFromValues(2, []int{1, 2}))
	count := m.NewVariable(NewBitSetDomain(4)) // encoded count+1

	bs, _ := NewBoolSum([]*FDVariable{b1, b2, b3}, count)
	m.AddConstraint(bs)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Maximize count (all booleans true)
	sol, objVal, _ := solver.SolveOptimal(ctx, count, false)
	actualCount := objVal - 1 // decode from encoded value
	fmt.Printf("Maximum count: %d (all satisfied: %v)\n", actualCount,
		sol[b1.ID()] == 2 && sol[b2.ID()] == 2 && sol[b3.ID()] == 2)
	// Output: Maximum count: 3 (all satisfied: true)
}

// ExampleCumulative_energeticReasoning demonstrates energetic reasoning detecting infeasibility.
func ExampleCumulative_energeticReasoning() {
	m := NewModel()
	// Three heavy tasks that cannot fit in the time window
	// Tasks: each dur=4, dem=3, capacity=5, window=[1..6]
	// Energy required: 3 * 4 * 3 = 36 work units
	// Energy available: 6 time * 5 capacity = 30 work units → OVERLOAD
	s1 := m.NewVariable(NewBitSetDomain(3))
	s2 := m.NewVariable(NewBitSetDomain(3))
	s3 := m.NewVariable(NewBitSetDomain(3))

	cum, _ := NewCumulative(
		[]*FDVariable{s1, s2, s3},
		[]int{4, 4, 4}, // durations
		[]int{3, 3, 3}, // demands
		5,              // capacity
	)
	m.AddConstraint(cum)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sols, _ := solver.Solve(ctx, 1)
	fmt.Printf("Solutions found: %d (energetic reasoning detects overload)\n", len(sols))
	// Output: Solutions found: 0 (energetic reasoning detects overload)
}
