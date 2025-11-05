```go
func ExampleNewModulo_timeSlotScheduling() {
	model := NewModel()

	// Variables: minute offset and time slot
	minuteOffset := model.NewVariable(NewBitSetDomainFromValues(121, rangeValues(15, 75))) // minutes 15-75
	timeSlot := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15)))       // 15-minute slots

	// Constraint: time_slot = minute_offset mod 15
	constraint, err := NewModulo(minuteOffset, 15, timeSlot)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	minuteDomain := solver.GetDomain(nil, minuteOffset.ID())
	slotDomain := solver.GetDomain(nil, timeSlot.ID())

	fmt.Printf("Minute offsets:")
	count := 0
	minuteDomain.IterateValues(func(v int) {
		if count < 15 { // Show first 15
			fmt.Printf(" %d", v)
			count++
		}
	})
	fmt.Printf("...\nTime slots:")
	slotDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Minute offsets: 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29...
	// Time slots: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
}

```


