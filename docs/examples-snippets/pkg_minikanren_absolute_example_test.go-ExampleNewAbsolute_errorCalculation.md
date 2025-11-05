```go
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

```


