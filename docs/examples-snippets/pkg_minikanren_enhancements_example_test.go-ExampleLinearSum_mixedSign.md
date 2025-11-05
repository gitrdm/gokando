```go
func ExampleLinearSum_mixedSign() {
	m := NewModel()
	// Profit model: revenue - cost = profit
	// revenue = 10*units, cost = 3*units, profit = 7*units
	// Or more realistically: profit = 5*productA - 2*productB
	productA := m.NewVariable(NewBitSetDomain(3))
	productB := m.NewVariable(NewBitSetDomain(3))
	profit := m.NewVariable(NewBitSetDomain(20))

	// Maximize: 5*A - 2*B
	ls, _ := NewLinearSum([]*FDVariable{productA, productB}, []int{5, -2}, profit)
	m.AddConstraint(ls)

	solver := NewSolver(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Find maximum profit
	sol, objVal, _ := solver.SolveOptimal(ctx, profit, false) // maximize
	fmt.Printf("Maximum profit: %d (A=%d, B=%d)\n", objVal, sol[productA.ID()], sol[productB.ID()])
	// Output: Maximum profit: 13 (A=3, B=1)
}

```


