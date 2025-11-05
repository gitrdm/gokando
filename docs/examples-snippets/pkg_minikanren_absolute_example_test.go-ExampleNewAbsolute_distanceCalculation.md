```go
func ExampleNewAbsolute_distanceCalculation() {
	model := NewModel()

	// Variables: position difference and distance
	// Using offset=15 to represent position difference [-10, 10] as [5, 25]
	positionDiff := model.NewVariable(NewBitSetDomainFromValues(26, []int{8, 12, 17, 22})) // represents [-7, -3, 2, 7]
	distance := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))       // represents [0, 9]

	// Constraint: distance = |position_diff|
	constraint, err := NewAbsolute(positionDiff, 15, distance)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	diffDomain := solver.GetDomain(nil, positionDiff.ID())
	distDomain := solver.GetDomain(nil, distance.ID())

	fmt.Printf("Position differences:")
	diffDomain.IterateValues(func(v int) {
		actual := v - 15 // decode from offset
		fmt.Printf(" %d", actual)
	})
	fmt.Printf("\nDistances:")
	distDomain.IterateValues(func(v int) {
		if v == 1 {
			fmt.Printf(" 0") // BitSetDomain encodes 0 as 1
		} else {
			fmt.Printf(" %d", v)
		}
	})
	fmt.Printf("\n")

	// Output:
	// Position differences: -7 -3 2 7
	// Distances: 2 3 7
}

```


