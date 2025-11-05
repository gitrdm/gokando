```go
func ExampleNewModulo_cyclicPatterns() {
	model := NewModel()

	// Variables: task ID and assigned processor
	taskID := model.NewVariable(NewBitSetDomainFromValues(21, []int{5, 8, 12, 17, 20})) // specific task IDs
	processor := model.NewVariable(NewBitSetDomainFromValues(5, rangeValues(1, 4)))     // 4 processors

	// Constraint: processor = task_id mod 4
	constraint, err := NewModulo(taskID, 4, processor)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	taskDomain := solver.GetDomain(nil, taskID.ID())
	procDomain := solver.GetDomain(nil, processor.ID())

	fmt.Printf("Task IDs:")
	taskDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nAssigned processors:")
	procDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Task IDs: 5 8 12 17 20
	// Assigned processors: 1 4
}

```


