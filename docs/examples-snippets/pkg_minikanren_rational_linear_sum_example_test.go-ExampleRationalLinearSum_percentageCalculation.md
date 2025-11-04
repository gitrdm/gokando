```go
func ExampleRationalLinearSum_percentageCalculation() {
	model := NewModel()

	// Base salary: $50,000
	baseSalary := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000}))
	// Total with 10% bonus. Use a realistic, narrower domain to keep the example fast.
	// Wide dense domains cause ScaledDivision to enumerate large ranges for arc-consistency.
	// Here we bound to [54_000..56_000] which still demonstrates propagation clearly
	// while keeping runtime well under a second.
	totalPay := model.NewVariable(DomainRange(54000, 56000))

	// Constraint: totalPay = 1.1 * baseSalary = (11/10) * baseSalary
	coeffs := []Rational{NewRational(11, 10)} // 110% = 11/10

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{baseSalary},
		coeffs,
		totalPay,
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

	totalDomain := solver.GetDomain(nil, totalPay.ID()).(*BitSetDomain)
	fmt.Printf("Base salary: $50,000\n")
	fmt.Printf("With 10%% bonus: $%d\n", totalDomain.Min())

	// Output:
	// Base salary: $50,000
	// With 10% bonus: $55000
}

```


