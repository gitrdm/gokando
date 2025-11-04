```go
func ExampleNewScaledDivision() {
	model := NewModel()

	// All monetary values scaled by 100 (cents)
	// Salary range: $500-$700 → 50000-70000 cents
	salaryValues := make([]int, 0)
	for s := 50000; s <= 70000; s += 10000 {
		salaryValues = append(salaryValues, s)
	}
	salaryVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(70001, salaryValues),
		"salary",
	)

	// Bonus initially unconstrained
	bonusVar := model.NewVariableWithName(
		NewBitSetDomainFromValues(10000, makeRangeEx(1, 10000)),
		"bonus",
	)

	// Constraint: bonus = salary / 10 (10% bonus)
	// Since values are in cents, this gives us exact integer division
	constraint, err := NewScaledDivision(salaryVar, 10, bonusVar)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint)

	// Solve to propagate constraints
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check propagated bonus domain
	finalBonus := solver.GetDomain(nil, bonusVar.ID())

	fmt.Println("Salary-to-Bonus constraint (10% bonus):")
	fmt.Printf("Salary range: $500.00 - $700.00\n")
	fmt.Printf("Bonus range: $%d.%02d - $%d.%02d\n",
		finalBonus.Min()/100, finalBonus.Min()%100,
		finalBonus.Max()/100, finalBonus.Max()%100)
	fmt.Printf("Possible bonuses: ")

	bonuses := finalBonus.ToSlice()
	for i, b := range bonuses {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("$%d.%02d", b/100, b%100)
	}
	fmt.Println()

	// Output:
	// Salary-to-Bonus constraint (10% bonus):
	// Salary range: $500.00 - $700.00
	// Bonus range: $50.00 - $70.00
	// Possible bonuses: $50.00, $60.00, $70.00
}

```


