```go
func ExampleNewScale_manufacturing() {
	model := NewModel()

	// Variables: production units and raw material consumption
	units := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(5, 20)))        // 5-20 units
	materials := model.NewVariable(NewBitSetDomainFromValues(301, rangeValues(50, 300))) // material inventory

	// Each unit requires 12 kg of raw material
	constraint, err := NewScale(units, 12, materials)
	if err != nil {
		panic(err)
	}

	model.AddConstraint(constraint)

	// Solve
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Get final domains after propagation
	unitsDomain := solver.GetDomain(nil, units.ID())
	materialsDomain := solver.GetDomain(nil, materials.ID())

	fmt.Printf("Production options:")
	unitsDomain.IterateValues(func(v int) {
		fmt.Printf(" %d", v)
	})
	fmt.Printf("\nMaterial usage:")
	materialsDomain.IterateValues(func(v int) {
		fmt.Printf(" %dkg", v)
	})
	fmt.Printf("\n")

	// Output:
	// Production options: 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20
	// Material usage: 60kg 72kg 84kg 96kg 108kg 120kg 132kg 144kg 156kg 168kg 180kg 192kg 204kg 216kg 228kg 240kg
}

```


