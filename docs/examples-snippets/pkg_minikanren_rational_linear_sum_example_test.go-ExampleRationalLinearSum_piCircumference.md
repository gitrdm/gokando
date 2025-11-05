```go
func ExampleRationalLinearSum_piCircumference() {
	model := NewModel()

	// Circle with diameter = 7 units
	// low-level: diameter := model.NewVariable(NewBitSetDomainFromValues(10, []int{7}))
	diameter := model.IntVarValues([]int{7}, "diameter")
	// low-level: circumference := model.NewVariable(NewBitSetDomain(100))
	circumference := model.IntVar(1, 100, "circumference")

	// Constraint: circumference = π * diameter
	// Using Archimedes' approximation: π ≈ 22/7
	pi := CommonIrrationals.PiArchimedes
	coeffs := []Rational{pi}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{diameter},
		coeffs,
		circumference,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	circumDomain := solver.GetDomain(nil, circumference.ID()).(*BitSetDomain)
	fmt.Printf("Diameter: %d units\n", 7)
	fmt.Printf("Circumference: %d units (using π ≈ 22/7)\n", circumDomain.Min())

	// Output:
	// Diameter: 7 units
	// Circumference: 22 units (using π ≈ 22/7)
}

```


