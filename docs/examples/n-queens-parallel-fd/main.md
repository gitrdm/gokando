# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main demonstrates a parallel N-Queens solver using the modern FD solver
// and gokanlogic's parallel execution framework.
//
// This example uses the modern Model/Solver API with global constraints to solve
// the N-Queens problem efficiently with parallel search.
//
// High-level approach:
//   - Model N-Queens with FD variables for queen columns (1..N per row)
//   - Use AllDifferent constraint for column uniqueness
//   - Model diagonals with derived variables and offset constraints
//   - Apply AllDifferent to both diagonal sets for diagonal uniqueness
//   - Use parallel search to explore first-queen placement concurrently
//
// Modern FD solver features demonstrated:
//   - Model/Solver API with sophisticated constraint propagation
//   - AllDifferent global constraint with efficient filtering
//   - Derived variables with arithmetic relationships
//   - Parallel execution with ParallelExecutor and context cancellation
//
// Performance benefits:
//   - Global constraints provide strong constraint propagation
//   - Parallel search explores independent subproblems concurrently
//   - Early termination when first solution is found
//
// Command-line options:
//   - N (positional arg, default 8): number of queens
//   - -sequential: run sequential solver only
//   - -both: run both sequential and parallel for comparison
package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
	// Parse command line arguments
	var n int = 8 // default
	var runSeq bool = false
	var runBoth bool = false

	// Parse flags and positional argument
	flag.BoolVar(&runSeq, "sequential", false, "run sequential solver only")
	flag.BoolVar(&runBoth, "both", false, "run both sequential and parallel")
	flag.Parse()

	// Parse positional argument for N (number of queens)
	if flag.NArg() > 0 {
		if parsed, err := strconv.Atoi(flag.Arg(0)); err == nil && parsed > 0 {
			n = parsed
		}
	}

	// Enforce reasonable bounds
	if n > 16 {
		n = 16
	}
	if n < 4 {
		n = 4
	}

	fmt.Printf("=== Parallel FD %d-Queens Demo ===\n\n", n)

	// Sequential solver (if requested)
	if runSeq || runBoth {
		fmt.Println("Running sequential FD solver...")
		start := time.Now()
		solution := solveNQueensSequential(n)
		duration := time.Since(start)

		if solution != nil {
			fmt.Printf("Sequential solved in %s\n", duration)
			displaySolution(solution, n)
		} else {
			fmt.Println("Sequential: no solution found")
		}
		fmt.Println()
	}

	// Parallel solver (default unless -sequential specified)
	if !runSeq {
		fmt.Println("Running parallel FD solver...")
		start := time.Now()
		solution := solveNQueensParallel(n)
		duration := time.Since(start)

		if solution != nil {
			fmt.Printf("Parallel solved in %s\n", duration)
			displaySolution(solution, n)
		} else {
			fmt.Println("Parallel: no solution found")
		}
	}
}

// solveNQueensSequential solves N-Queens using the modern FD solver sequentially
func solveNQueensSequential(n int) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("  Building single model with %d queen variables\n", n)

	model := mk.NewModel()

	// Create N variables for queen columns (1-based domains)
	domain := mk.NewBitSetDomain(n)
	queens := make([]*mk.FDVariable, n)
	for i := 0; i < n; i++ {
		queens[i] = model.NewVariableWithName(domain, fmt.Sprintf("Q%d", i+1))
	}

	// Add AllDifferent constraint for columns (no two queens in same column)
	allDiff, err := mk.NewAllDifferent(queens)
	if err != nil {
		return nil
	}
	model.AddConstraint(allDiff)

	// Add diagonal constraints using Table constraints
	// For each pair of queens (i,j), disallow positions where |i-j| == |col_i - col_j|
	constraintCount := 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			rowDiff := j - i // |i - j| since j > i

			// Generate all valid (col_i, col_j) pairs that satisfy diagonal constraint
			var validPairs [][]int
			for col_i := 1; col_i <= n; col_i++ {
				for col_j := 1; col_j <= n; col_j++ {
					colDiff := col_i - col_j
					if colDiff < 0 {
						colDiff = -colDiff
					}
					// Allow this pair if they're not on the same diagonal
					if colDiff != rowDiff {
						validPairs = append(validPairs, []int{col_i, col_j})
					}
				}
			}

			// Create table constraint
			if len(validPairs) > 0 {
				table, err := mk.NewTable([]*mk.FDVariable{queens[i], queens[j]}, validPairs)
				if err != nil {
					return nil
				}
				model.AddConstraint(table)
				constraintCount++
			}
		}
	}

	fmt.Printf("  Model built: %d variables, %d constraints\n", n, constraintCount+1)
	fmt.Printf("  Starting single sequential solver\n")

	// Solve and return first solution
	solver := mk.NewSolver(model)
	solutions, err := solver.Solve(ctx, 1)
	if err != nil || len(solutions) == 0 {
		fmt.Printf("  Sequential solver: no solution found\n")
		return nil
	}

	fmt.Printf("  Sequential solver: FOUND SOLUTION!\n")

	// Extract queen positions
	result := make([]int, n)
	for i, queen := range queens {
		result[i] = solutions[0][queen.ID()]
	}
	return result
}

// solveNQueensParallel solves N-Queens using parallel search over first queen placement
func solveNQueensParallel(n int) []int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("  Starting %d parallel workers (one per first queen position)\n", n)

	// Use a channel to collect the first solution from any worker
	solutionChan := make(chan []int, 1)
	workerCtx, workerCancel := context.WithCancel(ctx)

	// Track which worker finds the solution
	winnerChan := make(chan int, 1)

	// Create one goroutine for each possible first queen position
	for firstCol := 1; firstCol <= n; firstCol++ {
		go func(col int) {
			defer func() {
				if r := recover(); r != nil {
					// Handle any panics gracefully
					fmt.Printf("  Worker %d panicked: %v\n", col, r)
				}
			}()

			fmt.Printf("  Worker %d: starting (first queen at column %d)\n", col, col)

			model := mk.NewModel()

			// Create N variables for queen columns, with first queen fixed
			domain := mk.NewBitSetDomain(n)
			queens := make([]*mk.FDVariable, n)

			// Fix first queen to this column
			queens[0] = model.NewVariableWithName(mk.NewBitSetDomainFromValues(n, []int{col}), "Q1")

			// Create remaining queens with full domain
			for i := 1; i < n; i++ {
				queens[i] = model.NewVariableWithName(domain, fmt.Sprintf("Q%d", i+1))
			}

			// Add AllDifferent constraint for columns
			allDiff, err := mk.NewAllDifferent(queens)
			if err != nil {
				fmt.Printf("  Worker %d: AllDifferent error: %v\n", col, err)
				return
			}
			model.AddConstraint(allDiff)

			// Add diagonal constraints using Table constraints
			for i := 0; i < n; i++ {
				for j := i + 1; j < n; j++ {
					rowDiff := j - i

					var validPairs [][]int
					for col_i := 1; col_i <= n; col_i++ {
						for col_j := 1; col_j <= n; col_j++ {
							colDiff := col_i - col_j
							if colDiff < 0 {
								colDiff = -colDiff
							}
							if colDiff != rowDiff {
								validPairs = append(validPairs, []int{col_i, col_j})
							}
						}
					}

					if len(validPairs) > 0 {
						table, err := mk.NewTable([]*mk.FDVariable{queens[i], queens[j]}, validPairs)
						if err != nil {
							fmt.Printf("  Worker %d: Table constraint error: %v\n", col, err)
							return
						}
						model.AddConstraint(table)
					}
				}
			}

			fmt.Printf("  Worker %d: model built, starting solver\n", col)

			// Try to solve this branch
			solver := mk.NewSolver(model)
			solutions, err := solver.Solve(workerCtx, 1)

			if err != nil {
				if workerCtx.Err() == context.Canceled {
					fmt.Printf("  Worker %d: canceled (another worker found solution)\n", col)
				} else {
					fmt.Printf("  Worker %d: solver error: %v\n", col, err)
				}
				return
			}

			if len(solutions) > 0 {
				fmt.Printf("  Worker %d: FOUND SOLUTION!\n", col)
				result := make([]int, n)
				for i, queen := range queens {
					result[i] = solutions[0][queen.ID()]
				}

				// Try to send solution (non-blocking)
				select {
				case solutionChan <- result:
					winnerChan <- col
					workerCancel() // Cancel other workers
				case <-workerCtx.Done():
					fmt.Printf("  Worker %d: solution found but another worker was faster\n", col)
				}
			} else {
				fmt.Printf("  Worker %d: no solution found\n", col)
			}
		}(firstCol)
	}

	// Wait for first solution or timeout
	select {
	case solution := <-solutionChan:
		winner := <-winnerChan
		fmt.Printf("  Worker %d won!\n", winner)
		workerCancel()
		return solution
	case <-ctx.Done():
		fmt.Printf("  Timeout: no solution found\n")
		workerCancel()
		return nil
	}
}

// displaySolution prints the N-Queens solution as a chessboard
func displaySolution(solution []int, n int) {
	if solution == nil {
		return
	}

	fmt.Println("\nSolution:")
	for row := 0; row < n; row++ {
		for col := 1; col <= n; col++ {
			if solution[row] == col {
				fmt.Print("♛ ")
			} else {
				if (row+col)%2 == 0 {
					fmt.Print("□ ")
				} else {
					fmt.Print("■ ")
				}
			}
		}
		fmt.Println()
	}

	fmt.Print("\nQueen positions (row, col): ")
	for row := 0; row < n; row++ {
		if row > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("(%d,%d)", row+1, solution[row])
	}
	fmt.Println()
}

```

## Running the Example

To run this example:

```bash
cd n-queens-parallel-fd
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
