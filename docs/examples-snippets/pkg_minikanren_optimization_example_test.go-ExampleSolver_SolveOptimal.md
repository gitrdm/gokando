```go
func ExampleSolver_SolveOptimal() {
	model := NewModel()
	// x,y in {1,2,3}
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3}))
	// total T = x + 2*y
	tvar := model.NewVariable(NewBitSetDomain(20))
	ls, _ := NewLinearSum([]*FDVariable{x, y}, []int{1, 2}, tvar)
	model.AddConstraint(ls)

	solver := NewSolver(model)
	sol, obj, err := solver.SolveOptimal(context.Background(), tvar, true)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("best objective: %d\n", obj)
	_ = sol // values per variable in model order
	// Output:
	// best objective: 3
}

```


