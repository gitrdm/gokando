```go
func ExampleNewRationalLinearSumWithScaling() {
	model := NewModel()

	// Three investors with different ownership percentages
	// low-level: investorA := model.NewVariable(NewBitSetDomainFromValues(10000, []int{3000})) // $3000 invested
	investorA := model.IntVarValues([]int{3000}, "investorA") // $3000 invested
	// low-level: investorB := model.NewVariable(NewBitSetDomainFromValues(10000, []int{2000})) // $2000 invested
	investorB := model.IntVarValues([]int{2000}, "investorB") // $2000 invested
	// Total investment
	total := model.IntVar(1, 10000, "total")

	// Constraint: total = (1/3)*A + (1/2)*B (fractional ownership)
	// Note: This is a simplified example; in reality you'd sum all investments
	coeffs := []Rational{
		NewRational(1, 3), // investor A owns 1/3 of their contribution
		NewRational(1, 2), // investor B owns 1/2 of their contribution
	}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{investorA, investorB},
		coeffs,
		total,
		model,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)
	if div != nil {
		// The scaling helper created an intermediate variable and ScaledDivision constraint
		model.AddConstraint(div)
		fmt.Println("Scaling was needed (LCM > 1)")
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	totalDomain := solver.GetDomain(nil, total.ID()).(*BitSetDomain)
	fmt.Printf("Total: $%d\n", totalDomain.Min())

	// Output:
	// Scaling was needed (LCM > 1)
	// Total: $2000
}

```


