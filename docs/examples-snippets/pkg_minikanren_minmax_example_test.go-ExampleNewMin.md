```go
func ExampleNewMin() {
	model := NewModel()
	// Two variables with different lower bounds
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(3).RemoveAbove(6)) // [3..6]
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(5).RemoveAbove(7)) // [5..7]
	r := model.NewVariable(NewBitSetDomain(9))                               // [1..9]

	c, _ := NewMin([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate

	// R is clamped to [min mins .. min maxes] = [3 .. 6]
	dr := solver.GetDomain(nil, r.ID())
	fmt.Printf("R: [%d..%d]\n", dr.Min(), dr.Max())

	// All Xi are pruned to be >= R.min = 3
	dx := solver.GetDomain(nil, x.ID())
	dy := solver.GetDomain(nil, y.ID())
	fmt.Printf("X.min: %d, Y.min: %d\n", dx.Min(), dy.Min())
	// Output:
	// R: [3..6]
	// X.min: 3, Y.min: 5
}

```


