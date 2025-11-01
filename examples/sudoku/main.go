package main

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/gitrdm/gokando/pkg/minikanren"
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

// sudokuGoal defines the constraints for the Sudoku puzzle in a declarative,
// high-level style, consistent with idiomatic miniKanren usage.
func sudokuGoal(puzzle [81]int, solution *minikanren.Var) minikanren.Goal {
	// Create 81 logic variables for the board cells.
	vars := make([]minikanren.Term, 81)
	for i := 0; i < 81; i++ {
		// Using Fresh to create logic variables for each cell.
		vars[i] = minikanren.Fresh(fmt.Sprintf("c%d", i))
	}

	// A slice to hold all the constraint goals.
	var goals []minikanren.Goal

	// 1. Add constraints for the given numbers from the puzzle.
	// The Eq goal unifies a variable with a concrete value.
	for i, v := range puzzle {
		if v != 0 {
			goals = append(goals, minikanren.Eq(vars[i], minikanren.NewAtom(v)))
		}
	}

	// 2. Add AllDifferent constraints for each of the 9 rows.
	for r := 0; r < 9; r++ {
		rowVars := make([]*minikanren.Var, 9)
		for c := 0; c < 9; c++ {
			rowVars[c] = vars[r*9+c].(*minikanren.Var)
		}
		// FDAllDifferent ensures all variables in the list have a unique value.
		goals = append(goals, minikanren.FDAllDifferent(rowVars...))
	}

	// 3. Add AllDifferent constraints for each of the 9 columns.
	for c := 0; c < 9; c++ {
		colVars := make([]*minikanren.Var, 9)
		for r := 0; r < 9; r++ {
			colVars[r] = vars[r*9+c].(*minikanren.Var)
		}
		goals = append(goals, minikanren.FDAllDifferent(colVars...))
	}

	// 4. Add AllDifferent constraints for each of the 9 3x3 blocks.
	for br := 0; br < 3; br++ {
		for bc := 0; bc < 3; bc++ {
			blockVars := make([]*minikanren.Var, 0, 9)
			for r := 0; r < 3; r++ {
				for c := 0; c < 3; c++ {
					idx := (br*3+r)*9 + (bc*3 + c)
					blockVars = append(blockVars, vars[idx].(*minikanren.Var))
				}
			}
			goals = append(goals, minikanren.FDAllDifferent(blockVars...))
		}
	}

	// 5. Add domain constraints for all variables, ensuring they are digits 1-9.
	// FDIn constrains a variable to a specific set of integer values.
	for i := 0; i < 81; i++ {
		goals = append(goals, minikanren.FDIn(vars[i].(*minikanren.Var), []int{1, 2, 3, 4, 5, 6, 7, 8, 9}))
	}

	// 6. Unify the solution variable with a list of all cell variables.
	goals = append(goals, minikanren.Eq(solution, minikanren.List(vars...)))

	// 7. Combine all goals into a single conjunction and wrap with FDSolve.
	// FDSolve is the key: it collects all FD constraints and runs the solver.
	return minikanren.FDSolve(minikanren.Conj(goals...))
}

func main() {
	// Set up CPU profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Println("could not create CPU profile: ", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Println("could not start CPU profile: ", err)
		os.Exit(1)
	}
	defer pprof.StopCPUProfile()

	fmt.Println("--- Solving Sudoku using idiomatic high-level API (with 10s timeout) ---")
	start := time.Now()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run the solver to find one solution.
	results := minikanren.RunWithContext(ctx, 1, func(q *minikanren.Var) minikanren.Goal {
		return sudokuGoal(puzzle, q)
	})

	dur := time.Since(start)

	if len(results) == 0 {
		fmt.Println("No solutions found (or timeout reached).")
		return
	}

	fmt.Printf("Solved in %s, found %d solution(s)\n", dur, len(results))

	// Extract and print the solution from the result term.
	solution, ok := results[0].(*minikanren.Pair)
	if !ok {
		fmt.Println("Error: result is not a list.")
		return
	}

	var sol [81]int
	for i := 0; i < 81; i++ {
		val, ok := solution.Car().(*minikanren.Atom)
		if !ok {
			fmt.Printf("Error: cell %d is not an atom.\n", i)
			return
		}
		sol[i] = val.Value().(int)
		if cdr, ok := solution.Cdr().(*minikanren.Pair); ok {
			solution = cdr
		} else if i < 80 {
			fmt.Println("Error: solution list is too short.")
			return
		}
	}

	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			fmt.Printf("%d ", sol[r*9+c])
		}
		fmt.Println()
	}
}
