package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// ExampleNewModulo_basic demonstrates basic usage of the Modulo constraint.
// The Modulo constraint enforces remainder = x mod modulus relationships.
func ExampleNewModulo_basic() {
	model := NewModel()

	// Variables: day number and day of week
	dayNumber := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(1, 100))) // days 1-100
	dayOfWeek := model.NewVariable(NewBitSetDomainFromValues(8, rangeValues(1, 7)))     // 1=Mon, 2=Tue, ..., 7=Sun

	// Constraint: day_of_week = day_number mod 7
	// Note: In our encoding, modulo 0 becomes 7 for BitSetDomain compatibility
	constraint, err := NewModulo(dayNumber, 7, dayOfWeek)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	dayDomain := solver.GetDomain(nil, dayNumber.ID())
	weekDomain := solver.GetDomain(nil, dayOfWeek.ID())

	fmt.Printf("Day numbers (sample):")
	count := 0
	dayDomain.IterateValues(func(v int) {
		if count < 10 { // Show first 10
			fmt.Printf(" %d", v)
			count++
		}
	})
	fmt.Printf("...\nDays of week:")
	weekDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Day numbers (sample): 1 2 3 4 5 6 7 8 9 10...
	// Days of week: 1 2 3 4 5 6 7
}

// ExampleNewModulo_timeSlotScheduling demonstrates using Modulo for recurring time slots.
// Models how minute offsets map to recurring time slots.
func ExampleNewModulo_timeSlotScheduling() {
	model := NewModel()

	// Variables: minute offset and time slot
	minuteOffset := model.NewVariable(NewBitSetDomainFromValues(121, rangeValues(15, 75))) // minutes 15-75
	timeSlot := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15)))       // 15-minute slots

	// Constraint: time_slot = minute_offset mod 15
	constraint, err := NewModulo(minuteOffset, 15, timeSlot)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	minuteDomain := solver.GetDomain(nil, minuteOffset.ID())
	slotDomain := solver.GetDomain(nil, timeSlot.ID())

	fmt.Printf("Minute offsets:")
	count := 0
	minuteDomain.IterateValues(func(v int) {
		if count < 15 { // Show first 15
			fmt.Printf(" %d", v)
			count++
		}
	})
	fmt.Printf("...\nTime slots:")
	slotDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Minute offsets: 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29...
	// Time slots: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
}

// ExampleNewModulo_cyclicPatterns demonstrates modulo constraints for cyclic resource allocation.
// Models how tasks are distributed across a fixed number of processing units.
func ExampleNewModulo_cyclicPatterns() {
	model := NewModel()

	// Variables: task ID and assigned processor
	taskID := model.NewVariable(NewBitSetDomainFromValues(21, []int{5, 8, 12, 17, 20})) // specific task IDs
	processor := model.NewVariable(NewBitSetDomainFromValues(5, rangeValues(1, 4)))     // 4 processors

	// Constraint: processor = task_id mod 4
	constraint, err := NewModulo(taskID, 4, processor)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	taskDomain := solver.GetDomain(nil, taskID.ID())
	procDomain := solver.GetDomain(nil, processor.ID())

	fmt.Printf("Task IDs:")
	taskDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nAssigned processors:")
	procDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Task IDs: 5 8 12 17 20
	// Assigned processors: 1 4
}

// ExampleNewModulo_selfReference demonstrates the self-reference case x mod modulus = x.
// Shows how the constraint handles cases where the remainder equals the dividend.
func ExampleNewModulo_selfReference() {
	model := NewModel()

	// Variable that represents both dividend and remainder
	value := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) // values 1-20

	// Self-reference constraint: value mod 7 = value (only valid when value < 7)
	constraint, err := NewModulo(value, 7, value)
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
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Valid self-reference values: 1 2 3 4 5 6
}

// ExampleNewModulo_bidirectionalPropagation shows constraint propagation in both directions.
// Demonstrates how the constraint narrows domains through forward and backward propagation.
func ExampleNewModulo_bidirectionalPropagation() {
	model := NewModel()

	// Start with broad domains
	dividend := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(10, 30))) // values 10-30
	remainder := model.NewVariable(NewBitSetDomainFromValues(6, []int{2, 4}))         // specific remainders

	// Constraint: remainder = dividend mod 5
	constraint, err := NewModulo(dividend, 5, remainder)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	divDomain := solver.GetDomain(nil, dividend.ID())
	remDomain := solver.GetDomain(nil, remainder.ID())

	fmt.Printf("Dividend values:")
	divDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nRemainder values:")
	remDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Dividend values: 12 14 17 19 22 24 27 29
	// Remainder values: 2 4
}

// ExampleNewModulo_hashDistribution demonstrates using Modulo for hash table distribution.
// Models how hash values are distributed across buckets.
func ExampleNewModulo_hashDistribution() {
	model := NewModel()

	// Variables: hash value and bucket assignment
	hashValue := model.NewVariable(NewBitSetDomainFromValues(101, []int{23, 47, 89, 156, 234})) // hash values
	bucket := model.NewVariable(NewBitSetDomainFromValues(9, rangeValues(1, 8)))                // 8 buckets

	// Constraint: bucket = hash_value mod 8
	constraint, err := NewModulo(hashValue, 8, bucket)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	hashDomain := solver.GetDomain(nil, hashValue.ID())
	bucketDomain := solver.GetDomain(nil, bucket.ID())

	fmt.Printf("Hash values:")
	hashDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nBucket assignments:")
	bucketDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Hash values: 23 47 89
	// Bucket assignments: 1 7
}

// Test that examples actually run without panicking
func TestModuloExamples(t *testing.T) {
	// These examples should run without errors
	ExampleNewModulo_basic()
	ExampleNewModulo_timeSlotScheduling()
	ExampleNewModulo_cyclicPatterns()
	ExampleNewModulo_selfReference()
	ExampleNewModulo_bidirectionalPropagation()
	ExampleNewModulo_hashDistribution()
}
