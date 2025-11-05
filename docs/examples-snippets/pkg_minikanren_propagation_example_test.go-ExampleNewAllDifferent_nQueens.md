```go
func ExampleNewAllDifferent_nQueens() {
	n := 4
	model := NewModel()

	// Column positions for each row
	cols := model.NewVariables(n, NewBitSetDomain(n))

	// Diagonal variables (need larger domain to accommodate offsets)
	diag1 := model.NewVariables(n, NewBitSetDomain(2*n))
	diag2 := model.NewVariables(n, NewBitSetDomain(2*n))

	// Link diagonals to columns
	for i := 0; i < n; i++ {
		// diag1[i] = col[i] + i
		c, err := NewArithmetic(cols[i], diag1[i], i)
		if err != nil {
			panic(err)
		}
		model.AddConstraint(c)
		// diag2[i] = col[i] - i + n (offset to keep positive)
		c, err = NewArithmetic(cols[i], diag2[i], -i+n)
		if err != nil {
			panic(err)
		}
		model.AddConstraint(c)
	}

	// All queens in different columns, and different diagonals
	c, err := NewAllDifferent(cols)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag1)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewAllDifferent(diag2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 2) // Get 2 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: [", i+1)
		for row := 0; row < n; row++ {
			if row > 0 {
				fmt.Print(" ")
			}
			fmt.Print(sol[cols[row].ID()])
		}
		fmt.Println("]")
	}

	// Output:
	// Solution 1: [2 4 1 3]
	// Solution 2: [3 1 4 2]
}

```


