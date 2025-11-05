```go
func ExampleNewIntervalArithmetic_sum() {
	model := NewModel()

	// Variables: base cost and total cost
	baseCost := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(3, 8)))   // base cost 3-8
	totalCost := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) // total cost

	// Constraint: total_cost = base_cost + [5, 10] (additional fees)
	constraint, err := NewIntervalArithmetic(baseCost, 5, 10, IntervalSum, totalCost)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	baseDomain := solver.GetDomain(nil, baseCost.ID())
	totalDomain := solver.GetDomain(nil, totalCost.ID())

	fmt.Printf("Base cost:")
	baseDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nTotal cost:")
	totalDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Base cost: $3 $4 $5 $6 $7 $8
	// Total cost: $8 $9 $10 $11 $12 $13 $14 $15 $16 $17 $18
}

```


