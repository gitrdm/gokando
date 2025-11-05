```go
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

```


