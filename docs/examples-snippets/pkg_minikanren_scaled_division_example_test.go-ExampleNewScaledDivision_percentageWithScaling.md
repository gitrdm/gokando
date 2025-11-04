```go
func ExampleNewScaledDivision_percentageWithScaling() {
	model := NewModel()

	// Investment amount: $1000 (in dollars, not cents for this example)
	principal := model.NewVariableWithName(
		NewBitSetDomainFromValues(10001, []int{1000}), // $1000
		"principal",
	)

	// Annual interest rate: 5.25% → stored as 525 basis points (5.25 * 100)
	// Calculate: $1000 * 5.25 / 100 = $52.50
	// For integer result, scale by 100: 1000 * 525 / 100 = 5250 (in cents)
	interestScaled := model.NewVariableWithName(
		NewBitSetDomain(1000000),
		"interest_scaled",
	)

	interestCents := model.NewVariableWithName(
		NewBitSetDomain(10000),
		"interest_cents",
	)

	// Pattern: principal * 525 / 100 = interest_cents
	// Step 1: interest_scaled = principal * 525
	coeffs := []int{525}
	linearConstraint, _ := NewLinearSum(
		[]*FDVariable{principal},
		coeffs,
		interestScaled,
	)
	model.AddConstraint(linearConstraint)

	// Step 2: interest_cents = interest_scaled / 100
	divConstraint, _ := NewScaledDivision(interestScaled, 100, interestCents)
	model.AddConstraint(divConstraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalInterest := solver.GetDomain(nil, interestCents.ID())

	fmt.Println("Fixed-point percentage calculation:")
	fmt.Printf("Principal: $1,000.00\n")
	fmt.Printf("Rate: 5.25%% (525 basis points)\n")
	fmt.Printf("Interest: $%d.%02d\n",
		finalInterest.Min()/100, finalInterest.Min()%100)

	// Output:
	// Fixed-point percentage calculation:
	// Principal: $1,000.00
	// Rate: 5.25% (525 basis points)
	// Interest: $52.50
}

```


