// Package main demonstrates a parallel N-Queens solver using the FD solver
// and GoKando's parallel execution framework.
//
// This example is written in a literate, explanatory style so you can see
// how the FD solver and the parallel execution framework work together.
//
// High-level idea
//   - We model the N-Queens problem with one logic variable per row that
//     represents the column index (1..N).
//   - Columns are constrained with an AllDifferent constraint implemented
//     with Regin's filtering (the FD engine). Diagonals are modeled as
//     FD-derived variables: each diagonal variable is linked to a row variable
//     via an offset relation (y = x + k). The FD engine propagates those
//     offset constraints so diagonals participate in AllDifferent filtering.
//   - The FD propagation prunes the search aggressively before any search
//     branching occurs — this greatly reduces the work that the search must do.
//
// Parallelization strategy
//   - Finding a single solution is the goal here. We parallelize across the
//     choice for the first row: each worker explores the subproblem where the
//     first queen is fixed to a different column. This is an easy, low-risk
//     way to get parallel speedup because those branches are independent.
//   - The example uses `ParallelExecutor` and `executor.ParallelDisj` to run
//     branches concurrently. Once the first solution is found we cancel the
//     context so remaining workers can stop early.
//
// Why not fully automatic parallelism?
//   - The library exposes helpers (ParallelDisj, ParallelMap) so callers can
//     decide how to split the work. This keeps the core small and flexible —
//     the example shows one idiomatic way the caller can use the executor.
//
// Command-line flags
//   - -n int (default 10): number of queens to place (upper bounded to 16)
//   - -sequential: run only the sequential solver
//   - -both: run sequential first, then the parallel solver
//
// Usage examples
//   - Run parallel default with 12 queens:
//     go run ./examples/n-queens-parallel-fd -n 12
//   - Run sequential only:
//     go run ./examples/n-queens-parallel-fd -n 12 -sequential
//
// Practical notes
//   - This example demonstrates the FD+Regin combination — for larger N you
//     should add symmetry breaking (e.g., canonicalize the first queen into
//     half the board) or more advanced heuristics. The parallel split is a
//     convenient way to use additional cores without changing the solver.
package main

import (
	"context"
	"flag"
	"fmt"
	"runtime"
	"time"

	minikanren "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	// Default settings: parallel run is the default. Use flags to override.
	//
	// This example is structured so you can run it three ways:
	//  - parallel (default): spawn one branch per first-row column and run
	//    those branches concurrently using the ParallelExecutor. This is the
	//    most direct way to use multiple CPUs with minimal changes to the
	//    problem formulation.
	//  - sequential (-sequential): run the same FD-based solver single-threaded.
	//  - both (-both): run sequential first, then the parallel runner so you
	//    can compare timings on the same machine.
	n := flag.Int("n", 10, "number of queens (max 16)")
	runSeq := flag.Bool("sequential", false, "run sequential solver instead of parallel")
	runBoth := flag.Bool("both", false, "run both sequential and parallel (sequential first)")
	flag.Parse()

	// Enforce a sensible upper bound to avoid extremely long runs by default.
	if *n <= 0 {
		*n = 10
	}
	if *n > 16 {
		*n = 16
	}

	fmt.Printf("=== Parallel FD %d-Queens Demo ===\n\n", *n)

	// Run sequential optionally (either only if -sequential, or first if -both)
	//
	// The sequential run uses the same `nQueensFD` goal as the parallel run.
	// This keeps the comparison fair: the only difference is whether we split
	// the initial search into independent branches and exploit multiple
	// workers.
	if *runSeq || *runBoth {
		fmt.Println("Running sequential solver...")
		seqStart := time.Now()
		seq := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal { return nQueensFD(*n, q) })
		seqDur := time.Since(seqStart)

		if len(seq) == 0 {
			fmt.Println("Sequential: no solution found")
		} else {
			fmt.Printf("Sequential solved in %s\n", seqDur)
			displayBoard(seq[0], *n)
		}
	}

	// Run parallel by default unless the user specified -sequential only.
	if !*runSeq {
		// Build and start a ParallelExecutor. The executor exposes helpers like
		// ParallelDisj which evaluate a list of goals concurrently using a pool
		// of workers. We create one goal per possible value of the first queen.
		fmt.Println("Running parallel solver (default)...")
		config := minikanren.DefaultParallelConfig()
		config.MaxWorkers = runtime.NumCPU()

		executor := minikanren.NewParallelExecutor(config)
		defer executor.Shutdown()

		// q is the top-level result variable we will bind to the list of
		// queen columns for a solution. Each branch constructs its own fresh
		// queen variables so branches do not share state.
		q := minikanren.Fresh("q")
		branches := make([]minikanren.Goal, *n)

		// For each possible first-column value we create a branch goal that:
		//  1. Freshens a set of row variables (one per row)
		//  2. Constrains the first row variable to the concrete column c
		//  3. Invokes `FDQueensGoal` which models columns + derived diagonals
		//     and applies AllDifferent filtering via the FD solver
		//  4. Constrains the top-level q variable to the resulting list
		for col := 1; col <= *n; col++ {
			c := col // capture loop variable for closure
			branches[col-1] = func(ctx context.Context, store minikanren.ConstraintStore) *minikanren.Stream {
				// Each branch gets its own fresh logical variables.
				queens := make([]*minikanren.Var, *n)
				for i := 0; i < *n; i++ {
					queens[i] = minikanren.Fresh(fmt.Sprintf("q%d", i))
				}

				// Build the pair-list term representing the vector of columns
				termList := minikanren.List()
				for i := *n - 1; i >= 0; i-- {
					termList = minikanren.NewPair(queens[i], termList)
				}

				// Compose the branch: fix the first queen and solve the rest
				// with the FD-based modeling (columns + diagonal offsets).
				goal := minikanren.Conj(
					minikanren.Eq(queens[0], minikanren.NewAtom(c)),
					minikanren.FDQueensGoal(queens, *n),
					minikanren.Eq(q, termList),
				)

				return goal(ctx, store)
			}
		}

		// Execute the parallel disjunction and take the first solution.
		// We cancel the context after the first solution to let other workers
		// stop quickly — without cancellation workers might hang trying to
		// deliver results into closed streams and cause shutdown delays.
		parStart := time.Now()
		ctx, cancel := context.WithCancel(context.Background())
		initialStore := minikanren.NewLocalConstraintStore(minikanren.GetDefaultGlobalBus())
		stream := executor.ParallelDisj(branches...)(ctx, initialStore)
		solutions, _ := stream.Take(1)
		cancel()
		parDur := time.Since(parStart)

		if len(solutions) == 0 {
			fmt.Println("Parallel: no solution found")
		} else {
			// Extract the q value from the returned constraint store and display
			// it. Note that each branch returned a store where q is bound to a
			// concrete list of numeric atoms (the columns).
			value := solutions[0].GetSubstitution().DeepWalk(q)
			fmt.Printf("Parallel solved in %s\n", parDur)
			displayBoard(value, *n)
		}
	}
}

// done

// nQueensFD uses the FD solver for columns and models diagonals as FD-derived
// variables with offset links and AllDifferent (no Project is used for
// diagonals).
func nQueensFD(n int, q *minikanren.Var) minikanren.Goal {
	// Create N logic variables for each row's column (1..n).
	//
	// Each returned solution will bind these variables to integer atoms
	// representing columns (1-based). We also show how to wrap them into
	// a single list term to return via the top-level `q` variable.
	queens := make([]*minikanren.Var, n)
	terms := make([]minikanren.Term, n)
	for i := 0; i < n; i++ {
		v := minikanren.Fresh(fmt.Sprintf("q%d", i))
		queens[i] = v
		terms[i] = v
	}

	// FDQueensGoal (implemented in the library) does the heavy lifting:
	//   - creates an FD store for the variables
	//   - constrains the column variables to 1..n
	//   - creates derived diagonal FD variables and links them via offset
	//     constraints (d = x + offset)
	//   - applies Regin AllDifferent filtering on columns and diagonals
	//   - runs the FD solver to either find a unique assignment or leave the
	//     search to the host (miniKanren) to enumerate via backtracking
	//
	// By placing this goal inside the conjunction below we ensure the FD
	// solver participates in the search and that the final substitution
	// binds the logical vars to integer atoms.
	goals := []minikanren.Goal{minikanren.FDQueensGoal(queens, n)}

	// Build the list term that represents the vector of queen columns.
	// We return that list by constraining the provided `q` variable to it.
	termList := minikanren.List()
	for i := n - 1; i >= 0; i-- {
		termList = minikanren.NewPair(queens[i], termList)
	}
	goals = append(goals, minikanren.Eq(q, termList))

	// Conjoin the FD goal and the result binding. The caller can then run
	// this goal either sequentially or inside a parallel branch as shown
	// in main().
	return minikanren.Conj(goals...)
}

// displayBoard prints the board similar to the other n-queens example
func displayBoard(result minikanren.Term, n int) {
	// displayBoard expects the result to be a miniKanren pair-list where
	// each element is an integer Atom representing a column (1-based). We
	// walk the structure and convert it into a slice of 0-based column
	// indices for printing.
	pair, ok := result.(*minikanren.Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	positions := make([]int, n)
	idx := 0
	for pair != nil && idx < n {
		colTerm := pair.Car()
		if atom, ok := colTerm.(*minikanren.Atom); ok {
			if col, ok := atom.Value().(int); ok {
				positions[idx] = col - 1 // convert to 0-based for display
			}
		}
		cdr := pair.Cdr()
		if cdr == nil {
			break
		}
		pair, _ = cdr.(*minikanren.Pair)
		idx++
	}

	// Print board: a simple ASCII board where ♛ marks the queen.
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
