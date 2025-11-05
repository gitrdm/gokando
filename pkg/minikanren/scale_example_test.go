package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// ExampleNewScale_basic demonstrates basic usage of the Scale constraint.
// The Scale constraint enforces result = x * multiplier relationships.
func ExampleNewScale_basic() {
	model := NewModel()

	// Variables: hourly_rate and weekly_cost for a 40-hour work week
	hourlyRate := model.NewVariable(NewBitSetDomainFromValues(31, []int{20, 25, 30})) // $20, $25, $30
	weeklyCost := model.NewVariable(NewBitSetDomainFromValues(1201, rangeValues(800, 1200)))

	// Constraint: weekly_cost = hourly_rate * 40
	constraint, err := NewScale(hourlyRate, 40, weeklyCost)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	hourlyDomain := solver.GetDomain(nil, hourlyRate.ID())
	weeklyDomain := solver.GetDomain(nil, weeklyCost.ID())

	fmt.Printf("Hourly rates:")
	hourlyDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nWeekly costs:")
	weeklyDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Hourly rates: $20 $25 $30
	// Weekly costs: $800 $1000 $1200
}

// ExampleNewScale_resourceAllocation demonstrates using Scale for resource planning.
// Models the relationship between number of workers and total resource consumption.
func ExampleNewScale_resourceAllocation() {
	model := NewModel()

	// Variables: number of workers and total daily resource units needed
	workers := model.NewVariable(NewBitSetDomainFromValues(11, []int{2, 3, 4, 5}))            // 2-5 workers
	totalResources := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(10, 100))) // available resources

	// Each worker needs 15 resource units per day
	constraint, err := NewScale(workers, 15, totalResources)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	workersDomain := solver.GetDomain(nil, workers.ID())
	resourcesDomain := solver.GetDomain(nil, totalResources.ID())

	fmt.Printf("Feasible team sizes:")
	workersDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nResource requirements:")
	resourcesDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Feasible team sizes: 2 3 4 5
	// Resource requirements: 30 45 60 75
}

// ExampleNewScale_manufacturing demonstrates scaling constraints in production planning.
// Models the relationship between production quantity and raw material consumption.
func ExampleNewScale_manufacturing() {
	model := NewModel()

	// Variables: production units and raw material consumption
	units := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 20)))        // 5-20 units
	materials := model.NewVariable(NewBitSetDomainFromValues(301, rangeValues(50, 300))) // material inventory

	// Each unit requires 12 kg of raw material
	constraint, err := NewScale(units, 12, materials)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	unitsDomain := solver.GetDomain(nil, units.ID())
	materialsDomain := solver.GetDomain(nil, materials.ID())

	fmt.Printf("Production options:")
	unitsDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nMaterial usage:")
	materialsDomain.IterateValues(func(v int) {
		fmt.Printf(" %dkg", v)
	})
	fmt.Printf("\n")

	// Output:
	// Production options: 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20
	// Material usage: 60kg 72kg 84kg 96kg 108kg 120kg 132kg 144kg 156kg 168kg 180kg 192kg 204kg 216kg 228kg 240kg
}

// ExampleNewScale_bidirectionalPropagation shows how the constraint propagates in both directions.
func ExampleNewScale_bidirectionalPropagation() {
	model := NewModel()

	// Start with loose constraints
	baseValue := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))   // 1-10
	scaledValue := model.NewVariable(NewBitSetDomainFromValues(51, rangeValues(1, 50))) // 1-50

	// Constraint: scaled = base * 7
	constraint, err := NewScale(baseValue, 7, scaledValue)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	baseDomain := solver.GetDomain(nil, baseValue.ID())
	scaledDomain := solver.GetDomain(nil, scaledValue.ID())

	fmt.Printf("Base values:")
	baseDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nScaled values:")
	scaledDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Base values: 1 2 3 4 5 6 7
	// Scaled values: 7 14 21 28 35 42 49
}

// Helper function to generate ranges for examples
func rangeValues(min, max int) []int {
	result := make([]int, max-min+1)
	for i := 0; i < len(result); i++ {
		result[i] = min + i
	}
	return result
}

// Test that examples actually run without panicking
func TestScaleExamples(t *testing.T) {
	// These examples should run without errors
	ExampleNewScale_basic()
	ExampleNewScale_resourceAllocation()
	ExampleNewScale_manufacturing()
	ExampleNewScale_bidirectionalPropagation()
}
