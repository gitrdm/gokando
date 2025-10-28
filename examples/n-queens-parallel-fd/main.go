// Package main demonstrates a parallel N-Queens solver using the FD solverpackage nqueensparallelfd

// and GoKando's parallel execution framework.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	// Default to 8 queens (reasonable parallel workload), allow override
	n := 10
	if len(os.Args) > 1 {
		if parsed, err := strconv.Atoi(os.Args[1]); err == nil && parsed > 0 && parsed <= 12 {
			n = parsed
		}
	}

	fmt.Printf("=== Parallel FD %d-Queens Demo ===\n\n", n)

	// Use a small parallel config to demonstrate multi-worker execution
	config := DefaultParallelConfig()
	config.MaxWorkers = runtime.NumCPU()

	results := ParallelRunWithConfig(1, func(q *Var) Goal {
		return nQueensFD(n, q)
	}, config)

	if len(results) == 0 {
		fmt.Println("No solution found")
		return
	}

	fmt.Printf("✓ Parallel FD solution for %d queens:\n\n", n)
	displayBoard(results[0], n)
}

// nQueensFD uses the FD solver for column constraints and Project for diagonals.
func nQueensFD(n int, q *Var) Goal {
	// Create N logic variables for each row's column (1..n)
	queens := make([]*Var, n)
	terms := make([]Term, n)
	for i := 0; i < n; i++ {
		v := Fresh(fmt.Sprintf("q%d", i))
		queens[i] = v
		terms[i] = v
	}

	var goals []Goal

	// Enforce column domain 1..n and all-different using the FD engine
	goals = append(goals, FDAllDifferentGoal(queens, n))

	// Diagonal check using Project: ensure no two queens share a diagonal
	goals = append(goals, Project(terms, func(vals []Term) Goal {
		cols := make([]int, n)
		for i, val := range vals {
			atom, ok := val.(*Atom)
			if !ok {
				return Failure
			}
			c, ok2 := atom.Value().(int)
			if !ok2 {
				return Failure
			}
			cols[i] = c
		}

		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				rd := j - i
				cd := cols[j] - cols[i]
				if cd < 0 {
					cd = -cd
				}
				if rd == cd {
					return Failure
				}
			}
		}
		return Success
	}))

	// Return the solution as a list in q
	termList := List()
	// Build list from right to left to match List helper (List constructs with NewAtom(nil) as nil)
	for i := n - 1; i >= 0; i-- {
		termList = NewPair(queens[i], termList)
	}
	goals = append(goals, Eq(q, termList))

	return Conj(goals...)
}

// displayBoard prints the board similar to the other n-queens example
func displayBoard(result Term, n int) {
	pair, ok := result.(*Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	positions := make([]int, n)
	idx := 0
	for pair != nil && idx < n {
		colTerm := pair.Car()
		if atom, ok := colTerm.(*Atom); ok {
			if col, ok := atom.Value().(int); ok {
				positions[idx] = col - 1 // convert to 0-based for display
			}
		}
		cdr := pair.Cdr()
		if cdr == nil {
			break
		}
		pair, _ = cdr.(*Pair)
		idx++
	}

	// Print board
	for row := 0; row < n; row++ {
		for col := 0; col < n; col++ {
			if positions[row] == col {
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
