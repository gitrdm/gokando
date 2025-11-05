# main

This example demonstrates basic usage of the library.

## Source Code

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Simple Sudoku example using the FD layer. Puzzle format: 0 for empty.
var puzzle = [81]int{
	5, 3, 0, 0, 7, 0, 0, 0, 0,
	6, 0, 0, 1, 9, 5, 0, 0, 0,
	0, 9, 8, 0, 0, 0, 0, 6, 0,

	8, 0, 0, 0, 6, 0, 0, 0, 3,
	4, 0, 0, 8, 0, 3, 0, 0, 1,
	7, 0, 0, 0, 2, 0, 0, 0, 6,

	0, 6, 0, 0, 0, 0, 2, 8, 0,
	0, 0, 0, 4, 1, 9, 0, 0, 5,
	0, 0, 0, 0, 8, 0, 0, 7, 9,
}

func main() {
	s := minikanren.NewFDStore()
	vars := make([]*minikanren.FDVar, 81)
	for i := 0; i < 81; i++ {
		vars[i] = s.NewVar()
	}

	// apply givens first
	for i := 0; i < 81; i++ {
		v := puzzle[i]
		if v != 0 {
			if err := s.Assign(vars[i], v); err != nil {
				fmt.Println("Puzzle inconsistent at given", i, ":", err)
				return
			}
		}
	}

	// add row/col/block all-different constraints
	// rows
	for r := 0; r < 9; r++ {
		row := make([]*minikanren.FDVar, 9)
		for c := 0; c < 9; c++ {
			row[c] = vars[r*9+c]
		}
		s.AddAllDifferentRegin(row)
	}
	// cols
	for c := 0; c < 9; c++ {
		col := make([]*minikanren.FDVar, 9)
		for r := 0; r < 9; r++ {
			col[r] = vars[r*9+c]
		}
		s.AddAllDifferentRegin(col)
	}
	// blocks
	for br := 0; br < 3; br++ {
		for bc := 0; bc < 3; bc++ {
			block := make([]*minikanren.FDVar, 0, 9)
			for r := 0; r < 3; r++ {
				for c := 0; c < 3; c++ {
					idx := (br*3+r)*9 + (bc*3 + c)
					block = append(block, vars[idx])
				}
			}
			s.AddAllDifferentRegin(block)
		}
	}

	start := time.Now()
	sols, err := s.Solve(context.Background(), 1)
	dur := time.Since(start)
	if err != nil {
		fmt.Println("Solve error:", err)
		return
	}
	if len(sols) == 0 {
		fmt.Println("No solutions")
		return
	}
	fmt.Printf("Solved in %s, found %d solutions\n", dur, len(sols))
	sol := sols[0]
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			fmt.Printf("%d ", sol[r*9+c])
		}
		fmt.Printf("\n")
	}
}

```

## Running the Example

To run this example:

```bash
cd sudoku
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
