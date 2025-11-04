```go
func ExampleReifiedConstraint() {
	model := NewModel()
	x := model.NewVariableWithName(NewBitSetDomain(3), "X")
	y := model.NewVariableWithName(NewBitSetDomain(3), "Y")
	b := model.NewVariableWithName(NewBitSetDomain(2), "B") // {1,2} maps to {false,true}

	arith, _ := NewArithmetic(x, y, 0) // X + 0 = Y
	reified, _ := NewReifiedConstraint(arith, b)
	model.AddConstraint(reified)

	solver := NewSolver(model)
	solutions, _ := solver.Solve(context.Background(), 0)

	// Collect and sort output to make the example deterministic.
	var lines []string
	for _, sol := range solutions {
		lines = append(lines, fmt.Sprintf("X=%d Y=%d B=%t", sol[x.ID()], sol[y.ID()], sol[b.ID()] == 2))
	}
	sort.Strings(lines)

	for i := 0; i < 3 && i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	// Output:
	// X=1 Y=1 B=true
	// X=1 Y=2 B=false
	// X=1 Y=3 B=false
}

```


