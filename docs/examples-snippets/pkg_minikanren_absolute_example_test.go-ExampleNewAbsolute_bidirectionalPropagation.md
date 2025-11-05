```go
func ExampleNewAbsolute_bidirectionalPropagation() {
	model := NewModel()

	// Start with broad domains
	// Using offset=25 to represent input range [-15, 15] as [10, 40]
	inputValue := model.NewVariable(NewBitSetDomainFromValues(41, rangeValues(10, 40)))   // represents [-15, 15]
	absoluteValue := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15))) // represents [0, 14]

	// Constraint: absolute_value = |input_value|
	constraint, err := NewAbsolute(inputValue, 25, absoluteValue)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	inputDomain := solver.GetDomain(nil, inputValue.ID())
	absDomain := solver.GetDomain(nil, absoluteValue.ID())

	fmt.Printf("Input domain (decoded):")
	inputDomain.IterateValues(func(v int) {
		actual := v - 25 // decode from offset
		fmt.Printf(" %d", actual)
	})
	fmt.Printf("\nAbsolute value domain:")
	absDomain.IterateValues(func(v int) {
		if v == 1 {
			fmt.Printf(" 0") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d", v)
		}
	})
	fmt.Printf("\n")

	// Output: Input domain (decoded): -15 -14 -13 -12 -11 -10 -9 -8 -7 -6 -5 -4 -3 -2 -1 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
	// Absolute value domain: 0 2 3 4 5 6 7 8 9 10 11 12 13 14 15
}

```


