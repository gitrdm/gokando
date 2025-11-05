```go
func ExampleNewScale_basic() {
	model := NewModel()

	// Variables: hourly_rate and weekly_cost for a 40-hour work week
	hourlyRate := model.NewVariable(NewBitSetDomainFromValues(31, []int{20, 25, 30})) // $20, $25, $30
	weeklyCost := model.NewVariable(NewBitSetDomainFromValues(1201, rangeValues(800, 1200)))

	// Constraint: weekly_cost = hourly_rate * 40
	constraint, err := NewScale(hourlyRate, 40, weeklyCost)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	hourlyDomain := solver.GetDomain(nil, hourlyRate.ID())
	weeklyDomain := solver.GetDomain(nil, weeklyCost.ID())

	fmt.Printf("Hourly rates:")
	hourlyDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nWeekly costs:")
	weeklyDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Hourly rates: $20 $25 $30
	// Weekly costs: $800 $1000 $1200
}

```


