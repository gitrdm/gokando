package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// ExampleNewAbsolute_basic demonstrates basic usage of the Absolute constraint.
// The Absolute constraint enforces abs_value = |x| relationships using offset encoding.
func ExampleNewAbsolute_basic() {
	model := NewModel()

	// Variables: temperature difference and its absolute value
	// Using offset=20 to represent temperature range [-10, 10] as [10, 30]
	tempDiff := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(15, 25)))   // represents [-5, 5]
	absTempDiff := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) // represents [0, 9]

	// Constraint: abs_temp_diff = |temp_diff| with offset=20
	constraint, err := NewAbsolute(tempDiff, 20, absTempDiff)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	tempDomain := solver.GetDomain(nil, tempDiff.ID())
	absDomain := solver.GetDomain(nil, absTempDiff.ID())

	fmt.Printf("Temperature differences (encoded):")
	tempDomain.IterateValues(func(v int) {
		actual := v - 20 // decode from offset
		fmt.Printf(" %d°C", actual)
	})
	fmt.Printf("\nAbsolute differences:")
	absDomain.IterateValues(func(v int) {
		if v == 1 {
			fmt.Printf(" 0°C") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d°C", v)
		}
	})
	fmt.Printf("\n")

	// Output: Temperature differences (encoded): -5°C -4°C -3°C -2°C -1°C 0°C 1°C 2°C 3°C 4°C 5°C
	// Absolute differences: 0°C 2°C 3°C 4°C 5°C
}

// ExampleNewAbsolute_errorCalculation demonstrates using Absolute for error magnitude calculations.
// Models the relationship between measurement error and absolute error bounds.
func ExampleNewAbsolute_errorCalculation() {
	model := NewModel()

	// Variables: measurement error and error magnitude
	// Using offset=50 to represent error range [-30, 30] as [20, 80]
	measurementError := model.NewVariable(NewBitSetDomainFromValues(81, rangeValues(35, 65))) // represents [-15, 15]
	errorMagnitude := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))    // represents [0, 19]

	// Constraint: error_magnitude = |measurement_error|
	constraint, err := NewAbsolute(measurementError, 50, errorMagnitude)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	errorDomain := solver.GetDomain(nil, measurementError.ID())
	magnitudeDomain := solver.GetDomain(nil, errorMagnitude.ID())

	fmt.Printf("Measurement errors:")
	first := true
	errorDomain.IterateValues(func(v int) {
		if !first {
			fmt.Printf(",")
		}
		actual := v - 50 // decode from offset
		fmt.Printf(" %d", actual)
		first = false
	})
	fmt.Printf("\nError magnitudes:")
	first = true
	magnitudeDomain.IterateValues(func(v int) {
		if !first {
			fmt.Printf(",")
		}
		if v == 1 {
			fmt.Printf(" 0") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d", v)
		}
		first = false
	})
	fmt.Printf("\n")

	// Output: Measurement errors: -15, -14, -13, -12, -11, -10, -9, -8, -7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
	// Error magnitudes: 0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
}

// ExampleNewAbsolute_selfReference demonstrates the self-reference case |x| = x.
// Shows how the constraint handles the special case where absolute value equals the input.
func ExampleNewAbsolute_selfReference() {
	model := NewModel()

	// Variable that represents both input and absolute value
	// Using offset=10 to represent range [-5, 10] as [5, 20]
	value := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 20))) // represents [-5, 10]

	// Self-reference constraint: |value| = value (only valid for non-negative values)
	constraint, err := NewAbsolute(value, 10, value)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domain after propagation
	valueDomain := solver.GetDomain(nil, value.ID())

	fmt.Printf("Valid self-reference values:")
	valueDomain.IterateValues(func(v int) {
		actual := v - 10 // decode from offset
		fmt.Printf(" %d", actual)
	})
	fmt.Printf("\n")

	// Output:
	// Valid self-reference values: 0 1 2 3 4 5 6 7 8 9 10
}

// ExampleNewAbsolute_bidirectionalPropagation shows constraint propagation in both directions.
// Demonstrates how the constraint narrows domains through forward and backward propagation.
func ExampleNewAbsolute_bidirectionalPropagation() {
	model := NewModel()

	// Start with broad domains
	// Using offset=25 to represent input range [-15, 15] as [10, 40]
	inputValue := model.NewVariable(NewBitSetDomainFromValues(41, rangeValues(10, 40)))   // represents [-15, 15]
	absoluteValue := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15))) // represents [0, 14]

	// Constraint: absolute_value = |input_value|
	constraint, err := NewAbsolute(inputValue, 25, absoluteValue)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	inputDomain := solver.GetDomain(nil, inputValue.ID())
	absDomain := solver.GetDomain(nil, absoluteValue.ID())

	fmt.Printf("Input domain (decoded):")
	inputDomain.IterateValues(func(v int) {
		actual := v - 25 // decode from offset
		fmt.Printf(" %d", actual)
	})
	fmt.Printf("\nAbsolute value domain:")
	absDomain.IterateValues(func(v int) {
		if v == 1 {
			fmt.Printf(" 0") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d", v)
		}
	})
	fmt.Printf("\n")

	// Output: Input domain (decoded): -15 -14 -13 -12 -11 -10 -9 -8 -7 -6 -5 -4 -3 -2 -1 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
	// Absolute value domain: 0 2 3 4 5 6 7 8 9 10 11 12 13 14 15
}

// ExampleNewAbsolute_distanceCalculation demonstrates using Absolute for distance calculations.
// Models absolute distance between two points on a number line.
func ExampleNewAbsolute_distanceCalculation() {
	model := NewModel()

	// Variables: position difference and distance
	// Using offset=15 to represent position difference [-10, 10] as [5, 25]
	positionDiff := model.NewVariable(NewBitSetDomainFromValues(26, []int{8, 12, 17, 22})) // represents [-7, -3, 2, 7]
	distance := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))       // represents [0, 9]

	// Constraint: distance = |position_diff|
	constraint, err := NewAbsolute(positionDiff, 15, distance)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	diffDomain := solver.GetDomain(nil, positionDiff.ID())
	distDomain := solver.GetDomain(nil, distance.ID())

	fmt.Printf("Position differences:")
	diffDomain.IterateValues(func(v int) {
		actual := v - 15 // decode from offset
		fmt.Printf(" %d", actual)
	})
	fmt.Printf("\nDistances:")
	distDomain.IterateValues(func(v int) {
		if v == 1 {
			fmt.Printf(" 0") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d", v)
		}
	})
	fmt.Printf("\n")

	// Output:
	// Position differences: -7 -3 2 7
	// Distances: 2 3 7
}

// Test that examples actually run without panicking
func TestAbsoluteExamples(t *testing.T) {
	// These examples should run without errors
	ExampleNewAbsolute_basic()
	ExampleNewAbsolute_errorCalculation()
	ExampleNewAbsolute_selfReference()
	ExampleNewAbsolute_bidirectionalPropagation()
	ExampleNewAbsolute_distanceCalculation()
}
