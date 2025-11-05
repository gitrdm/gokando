```go
func ExampleNewIntervalArithmetic_intersection() {
	model := NewModel()

	// Variables: process time and acceptable time window
	processTime := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 20))) // process takes 5-20 minutes
	timeWindow := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15)))  // acceptable window

	// Constraint: time_window = intersection of process_time with [8, 15]
	constraint, err := NewIntervalArithmetic(processTime, 8, 15, IntervalIntersection, timeWindow)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	processDomain := solver.GetDomain(nil, processTime.ID())
	windowDomain := solver.GetDomain(nil, timeWindow.ID())

	fmt.Printf("Process time range:")
	processDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nAcceptable time window:")
	windowDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Process time range: 8 9 10 11 12 13 14 15
	// Acceptable time window: 8 9 10 11 12 13 14 15
}

```


