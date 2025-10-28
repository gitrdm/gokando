// Package main demonstrates a parallel N-Queens solver using the FD solver
// and GoKando's parallel execution framework.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	// Default to 8 queens (reasonable parallel workload), allow override
	n := 12
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

	// Now run parallel — actually parallelize across first-row choices using
	// a ParallelExecutor and ParallelDisj so work is split across workers.
	config := DefaultParallelConfig()
	config.MaxWorkers = runtime.NumCPU()

	executor := NewParallelExecutor(config)
	defer executor.Shutdown()

	// Build one branch per possible column for the first queen. Each branch
	// creates its own fresh variables and constrains the first queen to a
	// concrete column value, then delegates to the FD-based queens goal.
	q := Fresh("q")
	branches := make([]Goal, n)
	for col := 1; col <= n; col++ {
		c := col // capture loop variable
		branches[col-1] = func(ctx context.Context, store ConstraintStore) *Stream {
			// Create fresh queen variables for this branch
			queens := make([]*Var, n)
			for i := 0; i < n; i++ {
				queens[i] = Fresh(fmt.Sprintf("q%d", i))
			}

			// Build the result list term (q) from these queens
			termList := List()
			for i := n - 1; i >= 0; i-- {
				termList = NewPair(queens[i], termList)
			}

			// Branch: set first queen to c, run FD modeling, and return the
			// solved list in q.
			goal := Conj(
				Eq(queens[0], NewAtom(c)),
				FDQueensGoal(queens, n),
				Eq(q, termList),
			)

			return goal(ctx, store)
		}
	}

	// Execute branches in parallel and take the first solution
	parStart := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	initialStore := NewLocalConstraintStore(GetDefaultGlobalBus())
	stream := executor.ParallelDisj(branches...)(ctx, initialStore)
	solutions, _ := stream.Take(1)
	// Cancel remaining workers once we have our solution so they can tear down.
	cancel()
	parDur := time.Since(parStart)

	if len(solutions) == 0 {
		fmt.Println("Parallel: no solution found")
	} else {
		// Extract the q value from the returned constraint store
		value := solutions[0].GetSubstitution().DeepWalk(q)
		fmt.Printf("Parallel solved in %s\n", parDur)
		displayBoard(value, n)
	}
}

// done

// nQueensFD uses the FD solver for columns and models diagonals as FD-derived
// variables with offset links and AllDifferent (no Project is used for
// diagonals).
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
