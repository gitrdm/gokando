```go
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

```


