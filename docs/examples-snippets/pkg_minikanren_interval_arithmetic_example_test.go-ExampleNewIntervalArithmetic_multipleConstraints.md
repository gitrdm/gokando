```go
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

```


