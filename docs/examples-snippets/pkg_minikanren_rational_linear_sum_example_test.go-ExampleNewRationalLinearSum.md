```go
func ExampleNewRationalLinearSum() {
	model := NewModel()

	// Variables: hours worked
	// low-level: hours := model.NewVariable(NewBitSetDomainFromValues(50, []int{8})) // 8 hours worked
	hours := model.IntVarValues([]int{8}, "hours") // 8 hours worked
	// low-level: payment := model.NewVariable(NewBitSetDomain(1000))                 // payment in dollars
	payment := model.IntVar(1, 1000, "payment") // payment in dollars

	// Constraint: payment = 25 * hours (hourly rate of $25)
	coeffs := []Rational{NewRational(25, 1)} // $25/hour as coefficient
	rls, err := NewRationalLinearSum([]*FDVariable{hours}, coeffs, payment)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	model.AddConstraint(rls)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	paymentDomain := solver.GetDomain(nil, payment.ID()).(*BitSetDomain)
	fmt.Printf("Payment: $%d\n", paymentDomain.Min())

	// Output:
	// Payment: $200
}

```


