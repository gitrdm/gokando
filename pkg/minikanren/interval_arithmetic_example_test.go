package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// ExampleNewIntervalArithmetic_containment demonstrates basic interval containment.
// The IntervalArithmetic constraint can enforce that a variable's domain falls within specified bounds.
func ExampleNewIntervalArithmetic_containment() {
	model := NewModel()

	// Variable: temperature reading that must be within valid sensor range
	temperature := model.NewVariable(NewBitSetDomainFromValues(151, rangeValues(1, 150))) // broad initial range

	// Constraint: temperature must be within [20, 80] degrees
	constraint, err := NewIntervalArithmetic(temperature, 20, 80, IntervalContainment, nil)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domain after propagation
	tempDomain := solver.GetDomain(nil, temperature.ID())

	fmt.Printf("Valid temperature range:")
	count := 0
	tempDomain.IterateValues(func(v int) {
		if count < 10 { // Show first 10 values
			fmt.Printf(" %d°C", v)
			count++
		}
	})
	fmt.Printf("... (20-80°C)\n")

	// Output:
	// Valid temperature range: 20°C 21°C 22°C 23°C 24°C 25°C 26°C 27°C 28°C 29°C... (20-80°C)
}

// ExampleNewIntervalArithmetic_intersection demonstrates interval intersection operations.
// Shows how to compute the intersection of two intervals.
func ExampleNewIntervalArithmetic_intersection() {
	model := NewModel()

	// Variables: process time and acceptable time window
	processTime := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 20))) // process takes 5-20 minutes
	timeWindow := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15)))  // acceptable window

	// Constraint: time_window = intersection of process_time with [8, 15]
	constraint, err := NewIntervalArithmetic(processTime, 8, 15, IntervalIntersection, timeWindow)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	processDomain := solver.GetDomain(nil, processTime.ID())
	windowDomain := solver.GetDomain(nil, timeWindow.ID())

	fmt.Printf("Process time range:")
	processDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nAcceptable time window:")
	windowDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Process time range: 8 9 10 11 12 13 14 15
	// Acceptable time window: 8 9 10 11 12 13 14 15
}

// ExampleNewIntervalArithmetic_union demonstrates interval union operations.
// Shows how to compute the union (convex hull) of two intervals.
func ExampleNewIntervalArithmetic_union() {
	model := NewModel()

	// Variables: resource availability and total coverage
	availability := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(5, 12))) // available slots 5-12
	coverage := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))     // total coverage

	// Constraint: coverage = union of availability with [8, 18]
	constraint, err := NewIntervalArithmetic(availability, 8, 18, IntervalUnion, coverage)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	availDomain := solver.GetDomain(nil, availability.ID())
	coverageDomain := solver.GetDomain(nil, coverage.ID())

	fmt.Printf("Availability range:")
	availDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nTotal coverage:")
	coverageDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Availability range: 5 6 7 8 9 10 11 12
	// Total coverage: 5 6 7 8 9 10 11 12 13 14 15 16 17 18
}

// ExampleNewIntervalArithmetic_sum demonstrates interval sum operations.
// Shows how to compute the sum of two intervals: [a,b] + [c,d] = [a+c, b+d].
func ExampleNewIntervalArithmetic_sum() {
	model := NewModel()

	// Variables: base cost and total cost
	baseCost := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(3, 8)))   // base cost 3-8
	totalCost := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) // total cost

	// Constraint: total_cost = base_cost + [5, 10] (additional fees)
	constraint, err := NewIntervalArithmetic(baseCost, 5, 10, IntervalSum, totalCost)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	baseDomain := solver.GetDomain(nil, baseCost.ID())
	totalDomain := solver.GetDomain(nil, totalCost.ID())

	fmt.Printf("Base cost:")
	baseDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nTotal cost:")
	totalDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Base cost: $3 $4 $5 $6 $7 $8
	// Total cost: $8 $9 $10 $11 $12 $13 $14 $15 $16 $17 $18
}

// ExampleNewIntervalArithmetic_difference demonstrates interval difference operations.
// Shows how to compute the difference of two intervals: [a,b] - [c,d] = [a-d, b-c].
func ExampleNewIntervalArithmetic_difference() {
	model := NewModel()

	// Variables: revenue and net profit
	revenue := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(15, 30)))  // revenue 15-30
	netProfit := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) // net profit

	// Constraint: net_profit = revenue - [5, 12] (operating costs)
	constraint, err := NewIntervalArithmetic(revenue, 5, 12, IntervalDifference, netProfit)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	revenueDomain := solver.GetDomain(nil, revenue.ID())
	profitDomain := solver.GetDomain(nil, netProfit.ID())

	fmt.Printf("Revenue:")
	revenueDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nNet profit:")
	profitDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Revenue: $15 $16 $17 $18 $19 $20 $21 $22 $23 $24 $25 $26 $27 $28 $29 $30
	// Net profit: $3 $4 $5 $6 $7 $8 $9 $10 $11 $12 $13 $14 $15 $16 $17 $18 $19 $20
}

// ExampleNewIntervalArithmetic_bidirectionalSum shows bidirectional propagation in sum operations.
// Demonstrates how the constraint propagates constraints in both directions.
func ExampleNewIntervalArithmetic_bidirectionalSum() {
	model := NewModel()

	// Variables: component cost and total budget
	componentCost := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(1, 30))) // broad range
	totalBudget := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(12, 18)))  // constrained budget

	// Constraint: total_budget = component_cost + [3, 7] (assembly costs)
	constraint, err := NewIntervalArithmetic(componentCost, 3, 7, IntervalSum, totalBudget)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see bidirectional propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	costDomain := solver.GetDomain(nil, componentCost.ID())
	budgetDomain := solver.GetDomain(nil, totalBudget.ID())

	fmt.Printf("Component cost (constrained by budget):")
	costDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nTotal budget:")
	budgetDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Component cost (constrained by budget): $5 $6 $7 $8 $9 $10 $11 $12 $13
	// Total budget: $12 $13 $14 $15 $16
}

// ExampleNewIntervalArithmetic_multipleConstraints shows combining multiple interval constraints.
// Demonstrates using several interval arithmetic constraints together.
func ExampleNewIntervalArithmetic_multipleConstraints() {
	model := NewModel()

	// Variables for a resource allocation problem
	baseAllocation := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 15))) // base allocation
	totalNeeded := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(1, 30)))    // total needed

	// Constraint 1: total_needed = base_allocation + [2, 8] (overhead)
	constraint1, err := NewIntervalArithmetic(baseAllocation, 2, 8, IntervalSum, totalNeeded)
	if err != nil {
		panic(err)
	}

	// Constraint 2: total_needed must be within [10, 20] (resource limits)
	constraint2, err := NewIntervalArithmetic(totalNeeded, 10, 20, IntervalContainment, nil)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint1)
	model.AddConstraint(constraint2)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	baseDomain := solver.GetDomain(nil, baseAllocation.ID())
	totalDomain := solver.GetDomain(nil, totalNeeded.ID())

	fmt.Printf("Base allocation:")
	baseDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nTotal needed:")
	totalDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Base allocation: 5 6 7 8 9 10 11 12 13 14 15
	// Total needed: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30
}

// Test that examples actually run without panicking
func TestIntervalArithmeticExamples(t *testing.T) {
	// These examples should run without errors
	ExampleNewIntervalArithmetic_containment()
	ExampleNewIntervalArithmetic_intersection()
	ExampleNewIntervalArithmetic_union()
	ExampleNewIntervalArithmetic_sum()
	ExampleNewIntervalArithmetic_difference()
	ExampleNewIntervalArithmetic_bidirectionalSum()
	ExampleNewIntervalArithmetic_multipleConstraints()
}
