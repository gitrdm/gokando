```go
func ExampleNewMax() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(9).RemoveBelow(2).RemoveAbove(4)) // [2..4]
	y := model.NewVariable(NewBitSetDomain(9).RemoveBelow(6).RemoveAbove(8)) // [6..8]
	r := model.NewVariable(NewBitSetDomain(9))                               // [1..9]

	c, _ := NewMax([]*FDVariable{x, y}, r)
	model.AddConstraint(c)

	solver := NewSolver(model)
	_, _ = solver.Solve(context.Background(), 0) // propagate

	dr := solver.GetDomain(nil, r.ID())
	fmt.Printf("R: [%d..%d]\n", dr.Min(), dr.Max())

	// Xi are pruned to be <= R.max = 8 (no change for these domains)
	dx := solver.GetDomain(nil, x.ID())
	dy := solver.GetDomain(nil, y.ID())
	fmt.Printf("X.max: %d, Y.max: %d\n", dx.Max(), dy.Max())
	// Output:
	// R: [6..8]
	// X.max: 4, Y.max: 8
}

```


