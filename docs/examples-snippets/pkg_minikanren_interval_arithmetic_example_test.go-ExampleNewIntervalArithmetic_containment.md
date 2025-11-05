```go
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

```


