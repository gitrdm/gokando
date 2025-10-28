// Package main demonstrates a parallel N-Queens solver using the FD solver
// and GoKando's parallel execution framework.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

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

	// Run sequential (single-threaded) first to compare
	seqStart := time.Now()
	seq := Run(1, func(q *Var) Goal { return nQueensFD(n, q) })
	seqDur := time.Since(seqStart)

	if len(seq) == 0 {
		fmt.Println("Sequential: no solution found")
	} else {
		fmt.Printf("Sequential solved in %s\n", seqDur)
		displayBoard(seq[0], n)
	}

	// Now run parallel
	config := DefaultParallelConfig()
	config.MaxWorkers = runtime.NumCPU()
	parStart := time.Now()
	par := ParallelRunWithConfig(1, func(q *Var) Goal { return nQueensFD(n, q) }, config)
	parDur := time.Since(parStart)

	if len(par) == 0 {
		fmt.Println("Parallel: no solution found")
	} else {
		fmt.Printf("Parallel solved in %s\n", parDur)
		displayBoard(par[0], n)
	}
}

// done

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
	// Use the FD-based queens goal (will assert offsets and AllDifferent on diagonals)
	goals := []Goal{FDQueensGoal(queens, n)}

	// Return the solution as a list in q
	termList := List()
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
