```go
func ExampleNewArithmetic_chain() {
	model := NewModel()

	a := model.NewVariable(NewBitSetDomainFromValues(20, []int{2, 5}))
	b := model.NewVariable(NewBitSetDomain(20))
	c := model.NewVariable(NewBitSetDomain(20))

	// Create chain: B = A + 5, C = B + 3, so C = A + 8
	constraint1, err := NewArithmetic(a, b, 5)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint1)
	constraint2, err := NewArithmetic(b, c, 3)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(constraint2)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("a=%d, b=%d, c=%d (c = a + 8)\n",
			sol[a.ID()], sol[b.ID()], sol[c.ID()])
	}

	// Output:
	// a=2, b=7, c=10 (c = a + 8)
	// a=5, b=10, c=13 (c = a + 8)
}

```


