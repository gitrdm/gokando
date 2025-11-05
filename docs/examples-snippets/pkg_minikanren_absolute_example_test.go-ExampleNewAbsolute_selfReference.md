```go
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

```


