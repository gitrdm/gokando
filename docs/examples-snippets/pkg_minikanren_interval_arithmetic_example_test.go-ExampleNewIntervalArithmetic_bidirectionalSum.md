```go
func ExampleNewIntervalArithmetic_bidirectionalSum() {
	model := NewModel()

	// Variables: component cost and total budget
	componentCost := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(1, 30))) // broad range
	totalBudget := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(12, 18)))  // constrained budget

	// Constraint: total_budget = component_cost + [3, 7] (assembly costs)
	constraint, err := NewIntervalArithmetic(componentCost, 3, 7, IntervalSum, totalBudget)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve to see bidirectional propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	costDomain := solver.GetDomain(nil, componentCost.ID())
	budgetDomain := solver.GetDomain(nil, totalBudget.ID())

	fmt.Printf("Component cost (constrained by budget):")
	costDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\nTotal budget:")
	budgetDomain.IterateValues(func(v int) {
		fmt.Printf(" $%d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Component cost (constrained by budget): $5 $6 $7 $8 $9 $10 $11 $12 $13
	// Total budget: $12 $13 $14 $15 $16
}

```


