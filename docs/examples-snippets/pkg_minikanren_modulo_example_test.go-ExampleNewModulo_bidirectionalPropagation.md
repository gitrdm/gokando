```go
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

```


