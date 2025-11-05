```go
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

```


