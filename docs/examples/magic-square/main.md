# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main demonstrates solving a 3x3 magic square using the
// production FD solver with global constraints (no hybrid workaround).
//
// A magic square is a 3x3 grid using digits 1..9 such that every row,
// column, and both diagonals sum to 15. We model this with:
// - 9 FD variables with domain 1..9
// - AllDifferent over all 9 cells
// - LinearSum per row/column/diagonal equated to a total variable fixed at 15
// Optionally, symmetry breaking like center=5 can be added for speed, but the
// solver is fast enough here without it.
package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
	fmt.Println("=== FD Magic Square (3x3) ===")

	model := mk.NewModel()
	d1to9 := mk.NewBitSetDomain(9) // {1..9}

	// Create grid variables
	grid := make([][]*mk.FDVariable, 3)
	names := [][]string{{"a11", "a12", "a13"}, {"a21", "a22", "a23"}, {"a31", "a32", "a33"}}
	for i := 0; i < 3; i++ {
		grid[i] = make([]*mk.FDVariable, 3)
		for j := 0; j < 3; j++ {
			grid[i][j] = model.NewVariableWithName(d1to9, names[i][j])
		}
	}

	// AllDifferent over all cells
	all := []*mk.FDVariable{}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			all = append(all, grid[i][j])
		}
	}
	ad, _ := mk.NewAllDifferent(all)
	model.AddConstraint(ad)

	// Helper to post sum == 15 using LinearSum to a fixed total variable
	addSum15 := func(vars ...*mk.FDVariable) {
		total := model.NewVariableWithName(mk.NewBitSetDomainFromValues(50, []int{15}), "sum15")
		coeffs := make([]int, len(vars))
		for i := range coeffs {
			coeffs[i] = 1
		}
		ls, _ := mk.NewLinearSum(vars, coeffs, total)
		model.AddConstraint(ls)
	}

	// Rows
	for i := 0; i < 3; i++ {
		addSum15(grid[i][0], grid[i][1], grid[i][2])
	}
	// Columns
	for j := 0; j < 3; j++ {
		addSum15(grid[0][j], grid[1][j], grid[2][j])
	}
	// Diagonals
	addSum15(grid[0][0], grid[1][1], grid[2][2])
	addSum15(grid[0][2], grid[1][1], grid[2][0])

	// Solve
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sols, _ := solver.Solve(ctx, 1)

	if len(sols) == 0 {
		fmt.Println("No solution found (unexpected)")
		return
	}

	// Print solution
	fmt.Println("Solution:")
	sol := sols[0]
	// The first 9 variables created were the grid cells in row-major order
	idx := 0
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf(" %d ", sol[idx])
			idx++
		}
		fmt.Println()
	}
}

```

## Running the Example

To run this example:

```bash
cd magic-square
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
