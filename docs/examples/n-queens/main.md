# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main solves the N-Queens puzzle using GoKando.
//
// The N-Queens puzzle: Place N queens on an N×N chessboard such that no two queens
// attack each other. Queens can attack any piece on the same row, column, or diagonal.
//
// This implementation uses Project to verify the constraints efficiently for small N (4-8).
// For larger boards, a more sophisticated constraint propagation approach would be needed.
package main

import (
	"fmt"
	"os"
	"strconv"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	// Default to 6 queens (fast), allow command-line override
	n := 6
	if len(os.Args) > 1 {
		if parsed, err := strconv.Atoi(os.Args[1]); err == nil && parsed > 0 && parsed <= 8 {
			n = parsed
		}
	}

	fmt.Printf("=== Solving the %d-Queens Puzzle ===\n\n", n)

	// Find first solution
	results := Run(1, func(q *Var) Goal {
		return nQueens(n, q)
	})

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Printf("✓ Solution found for %d queens!\n\n", n)
	displayBoard(results[0], n)
}

// nQueens solves the N-Queens problem.
// Returns a list of N column positions, where position i is the column for queen in row i.
func nQueens(n int, q *Var) Goal {
	// Create variables for each queen's column position (0-indexed)
	queens := make([]Term, n)
	for i := 0; i < n; i++ {
		queens[i] = Fresh(fmt.Sprintf("q%d", i))
	}

	// Helper: ensure queen is in valid column (0 to n-1)
	validColumn := func(queen Term) Goal {
		goals := make([]Goal, n)
		for col := 0; col < n; col++ {
			goals[col] = Eq(queen, NewAtom(col))
		}
		return Disj(goals...)
	}

	// Build constraints
	var goals []Goal

	// Each queen must be in a valid column
	for i := 0; i < n; i++ {
		goals = append(goals, validColumn(queens[i]))
	}

	// All queens must be in different columns
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			goals = append(goals, Neq(queens[i], queens[j]))
		}
	}

	// Use Project to verify no diagonal attacks
	goals = append(goals, Project(queens, func(vals []Term) Goal {
		// Extract column positions
		cols := make([]int, n)
		for i, val := range vals {
			if atom, ok := val.(*Atom); ok {
				if col, ok := atom.Value().(int); ok {
					cols[i] = col
				} else {
					return Failure
				}
			} else {
				return Failure
			}
		}

		// Check no two queens on same diagonal
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				rowDiff := j - i
				colDiff := cols[j] - cols[i]
				if colDiff < 0 {
					colDiff = -colDiff
				}

				if rowDiff == colDiff {
					return Failure // Same diagonal
				}
			}
		}

		return Success
	}))

	// Return solution as list
	goals = append(goals, Eq(q, List(queens...)))

	return Conj(goals...)
}

// displayBoard pretty-prints the chessboard
func displayBoard(result Term, n int) {
	pair, ok := result.(*Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	// Extract queen positions
	positions := make([]int, n)
	idx := 0

	for pair != nil && idx < n {
		colTerm := pair.Car()
		if atom, ok := colTerm.(*Atom); ok {
			if col, ok := atom.Value().(int); ok {
				positions[idx] = col
			}
		}

		pair, _ = pair.Cdr().(*Pair)
		idx++
	}

	// Print board
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			if positions[row] == col {
				fmt.Print("♛ ")
			} else {
				// Checkerboard pattern
				if (row+col)%2 == 0 {
					fmt.Print("□ ")
				} else {
					fmt.Print("■ ")
				}
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Print("Queen positions (row, col): ")
	for row := 0; row < n; row++ {
		if row > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("(%d,%d)", row, positions[row])
	}
	fmt.Println()
}

```

## Running the Example

To run this example:

```bash
cd n-queens
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
