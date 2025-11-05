```go
func ExampleNewScale_resourceAllocation() {
	model := NewModel()

	// Variables: number of workers and total daily resource units needed
	workers := model.NewVariable(NewBitSetDomainFromValues(11, []int{2, 3, 4, 5}))            // 2-5 workers
	totalResources := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(10, 100))) // available resources

	// Each worker needs 15 resource units per day
	constraint, err := NewScale(workers, 15, totalResources)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	workersDomain := solver.GetDomain(nil, workers.ID())
	resourcesDomain := solver.GetDomain(nil, totalResources.ID())

	fmt.Printf("Feasible team sizes:")
	workersDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nResource requirements:")
	resourcesDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\n")

	// Output:
	// Feasible team sizes: 2 3 4 5
	// Resource requirements: 30 45 60 75
}

```


