```go
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

```


