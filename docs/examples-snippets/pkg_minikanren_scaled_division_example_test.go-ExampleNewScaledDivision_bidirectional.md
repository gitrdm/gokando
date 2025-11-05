```go
func ExampleNewScaledDivision_bidirectional() {
	model := NewModel()

	// Price scaled by 100 (cents)
	priceVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(2000, makeRangeEx(500, 1500)),
		"price_cents",
	)

	// Discount rate (percentage): 10-20%
	discountVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(21, makeRangeEx(10, 20)),
		"discount_pct",
	)

	// Constraint: discount_pct = price_cents / 100
	// This means price must be divisible by 100 for exact percentage
	constraint, _ := NewScaledDivision(priceVar, 100, discountVar)
	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalPrice := solver.GetDomain(nil, priceVar.ID())
	finalDiscount := solver.GetDomain(nil, discountVar.ID())

	fmt.Println("Bidirectional propagation:")
	fmt.Printf("Price: $%d.%02d - $%d.%02d\n",
		finalPrice.Min()/100, finalPrice.Min()%100,
		finalPrice.Max()/100, finalPrice.Max()%100)
	fmt.Printf("Discount: %d%% - %d%%\n",
		finalDiscount.Min(), finalDiscount.Max())

	// Output:
	// Bidirectional propagation:
	// Price: $10.00 - $15.00
	// Discount: 10% - 15%
}

```


