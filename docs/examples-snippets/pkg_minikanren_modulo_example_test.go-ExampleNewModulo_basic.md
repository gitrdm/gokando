```go
func ExampleNewModulo_basic() {
	model := NewModel()

	// Variables: day number and day of week
	dayNumber := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(1, 100))) // days 1-100
	dayOfWeek := model.NewVariable(NewBitSetDomainFromValues(8, rangeValues(1, 7)))     // 1=Mon, 2=Tue, ..., 7=Sun

	// Constraint: day_of_week = day_number mod 7
	// Note: In our encoding, modulo 0 becomes 7 for BitSetDomain compatibility
	constraint, err := NewModulo(dayNumber, 7, dayOfWeek)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	dayDomain := solver.GetDomain(nil, dayNumber.ID())
	weekDomain := solver.GetDomain(nil, dayOfWeek.ID())

	fmt.Printf("Day numbers (sample):")
	count := 0
	dayDomain.IterateValues(func(v int) {
		if count < 10 { // Show first 10
			fmt.Printf(" %d", v)
			count++
		}
	})
	fmt.Printf("...\nDays of week:")
	weekDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Day numbers (sample): 1 2 3 4 5 6 7 8 9 10...
	// Days of week: 1 2 3 4 5 6 7
}

```


