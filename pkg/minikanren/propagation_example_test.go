package minikanren_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleNewAllDifferent demonstrates creating an AllDifferent constraint
// to ensure variables take distinct values.
func ExampleNewAllDifferent() {
	model := NewModel()

	// Create three variables with domain {1, 2, 3}
	// low-level: x := model.NewVariable(NewBitSetDomain(3))
	x := model.IntVar(1, 3, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(3))
	y := model.IntVar(1, 3, "y")
	// low-level: z := model.NewVariable(NewBitSetDomain(3))
	z := model.IntVar(1, 3, "z")

	// Ensure all three variables have different values
	c, err := NewAllDifferent([]*FDVariable{x, y, z})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 2) // Get first 2 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d, z=%d\n", i+1, sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2, z=3
	// Solution 2: x=1, y=3, z=2
}

// ExampleNewArithmetic demonstrates creating an arithmetic constraint
// to enforce relationships like X + offset = Y.
func ExampleNewArithmetic() {
	model := NewModel()

	// Create variables with specific domains
	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 5, 7}))
	x := model.IntVarValues([]int{2, 5, 7}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X + 3
	c, err := NewArithmetic(x, y, 3)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0) // Get all solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x + 3)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2, y=5 (y = x + 3)
	// x=5, y=8 (y = x + 3)
	// x=7, y=10 (y = x + 3)
}

// ExampleNewArithmetic_negative demonstrates arithmetic with negative offsets
// to create subtraction relationships.
func ExampleNewArithmetic_negative() {
	model := NewModel()

	// low-level: x := model.NewVariable(NewBitSetDomainFromValues(10, []int{3, 5, 8}))
	x := model.IntVarValues([]int{3, 5, 8}, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(10))
	y := model.IntVar(1, 10, "y")

	// Enforce: Y = X - 2 (using negative offset)
	c, err := NewArithmetic(x, y, -2)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d, y=%d (y = x - 2)\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=3, y=1 (y = x - 2)
	// x=5, y=3 (y = x - 2)
	// x=8, y=6 (y = x - 2)
}

// ExampleNewInequality_lessThan demonstrates creating a less-than constraint.
func ExampleNewInequality_lessThan() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{2}))
	y := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 0)

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d\n", sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// x=2 < y=3
	// x=2 < y=4
	// x=2 < y=5
}

// ExampleNewInequality_notEqual demonstrates the not-equal constraint.
func ExampleNewInequality_notEqual() {
	model := NewModel()

	// low-level: x := model.NewVariable(NewBitSetDomain(3))
	x := model.IntVar(1, 3, "x")
	// low-level: y := model.NewVariable(NewBitSetDomain(3))
	y := model.IntVar(1, 3, "y")

	// Enforce: X ≠ Y
	c, err := NewInequality(x, y, NotEqual)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 3) // Get first 3 solutions

	for i, sol := range solutions {
		fmt.Printf("Solution %d: x=%d, y=%d\n", i+1, sol[x.ID()], sol[y.ID()])
	}

	// Output:
	// Solution 1: x=1, y=2
	// Solution 2: x=1, y=3
	// Solution 3: x=2, y=1
}

// ExampleNewAllDifferent_nQueens demonstrates solving the 4-Queens problem
// using AllDifferent constraints with arithmetic constraints for diagonals.
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

// ExampleNewInequality_ordering demonstrates combining multiple inequality
// constraints to enforce a total ordering.
func ExampleNewInequality_ordering() {
	model := NewModel()

	x := model.NewVariable(NewBitSetDomain(5))
	y := model.NewVariable(NewBitSetDomain(5))
	z := model.NewVariable(NewBitSetDomain(5))

	// Enforce: X < Y < Z (ascending order)
	c, err := NewInequality(x, y, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)
	c, err = NewInequality(y, z, LessThan)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	solutions, _ := solver.Solve(ctx, 5) // Get first 5 solutions

	for _, sol := range solutions {
		fmt.Printf("x=%d < y=%d < z=%d\n", sol[x.ID()], sol[y.ID()], sol[z.ID()])
	}

	// Output:
	// x=1 < y=2 < z=3
	// x=1 < y=2 < z=4
	// x=1 < y=2 < z=5
	// x=1 < y=3 < z=4
	// x=1 < y=3 < z=5
}

// ExampleNewArithmetic_chain demonstrates chaining arithmetic constraints
// to create relationships between multiple variables.
func ExampleNewArithmetic_chain() {
	model := NewModel()

	// low-level: a := model.NewVariable(NewBitSetDomainFromValues(20, []int{2, 5}))
	a := model.IntVarValues([]int{2, 5}, "a")
	// low-level: b := model.NewVariable(NewBitSetDomain(20))
	b := model.IntVar(1, 20, "b")
	// low-level: c := model.NewVariable(NewBitSetDomain(20))
	c := model.IntVar(1, 20, "c")

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
