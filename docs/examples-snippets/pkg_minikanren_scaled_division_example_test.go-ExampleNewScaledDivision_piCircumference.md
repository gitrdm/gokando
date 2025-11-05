```go
func ExampleNewScaledDivision_piCircumference() {
	model := NewModel()

	// Circle diameter: 1-10 units
	diameter := model.NewVariableWithName(
		NewBitSetDomainFromValues(11, makeRangeEx(1, 10)),
		"diameter",
	)

	// Circumference (scaled by 10000): π * diameter * 10000
	// Using π ≈ 31416/10000 (more precision than 22/7)
	circumferenceScaled := model.NewVariableWithName(
		NewBitSetDomain(350000),
		"circumference_scaled",
	)

	// Actual circumference (in original units)
	circumference := model.NewVariableWithName(
		NewBitSetDomain(35),
		"circumference",
	)

	// Pattern: Fixed-point arithmetic for irrationals
	// 1. Scale the constant: π ≈ 31416/10000
	// 2. Use LinearSum with scaled constant: circumference_scaled = 31416 * diameter
	// 3. Use ScaledDivision to get final result: circumference = circumference_scaled / 10000

	// Step 1: circumference_scaled = 31416 * diameter
	coeffs := []int{31416}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{diameter},
		coeffs,
		circumferenceScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: circumference = circumference_scaled / 10000
	divConstraint, _ := NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	// Fix diameter = 7 for demonstration
	diameter7Domain := NewBitSetDomainFromValues(11, []int{7})
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")

	// Rebuild constraints with fixed diameter
	model = NewModel()
	diameter = model.NewVariableWithName(diameter7Domain, "diameter")
	circumferenceScaled = model.NewVariableWithName(NewBitSetDomain(350000), "circumference_scaled")
	circumference = model.NewVariableWithName(NewBitSetDomain(35), "circumference")

	linearConstraint, _ = NewLinearSum([]*FDVariable{diameter}, coeffs, circumferenceScaled)
	model.AddConstraint(linearConstraint)

	divConstraint, _ = NewScaledDivision(circumferenceScaled, 10000, circumference)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalCircum := solver.GetDomain(nil, circumference.ID())
	finalScaled := solver.GetDomain(nil, circumferenceScaled.ID())

	fmt.Println("Fixed-point π calculation:")
	fmt.Printf("Diameter: 7 units\n")
	fmt.Printf("π * 7 * 10000 = %d (scaled)\n", finalScaled.Min())
	fmt.Printf("Circumference: %d units (actual)\n", finalCircum.Min())
	fmt.Printf("Precision: Using π ≈ 3.1416\n")

	// Output:
	// Fixed-point π calculation:
	// Diameter: 7 units
	// π * 7 * 10000 = 219912 (scaled)
	// Circumference: 21 units (actual)
	// Precision: Using π ≈ 3.1416
}

```


