```go
func ExampleNewIntervalArithmetic_difference() {
	model := NewModel()

	// Variables: revenue and net profit
	revenue := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(15, 30)))  // revenue 15-30
	netProfit := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) // net profit

	// Constraint: net_profit = revenue - [5, 12] (operating costs)
	constraint, err := NewIntervalArithmetic(revenue, 5, 12, IntervalDifference, netProfit)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	revenueDomain := solver.GetDomain(nil, revenue.ID())
	profitDomain := solver.GetDomain(nil, netProfit.ID())

	fmt.Printf("Revenue:")
	revenueDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nNet profit:")
	profitDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Revenue: $15 $16 $17 $18 $19 $20 $21 $22 $23 $24 $25 $26 $27 $28 $29 $30
	// Net profit: $3 $4 $5 $6 $7 $8 $9 $10 $11 $12 $13 $14 $15 $16 $17 $18 $19 $20
}

```


