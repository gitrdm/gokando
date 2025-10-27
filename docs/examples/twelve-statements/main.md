# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main solves the Twelve Statements puzzle using GoKando.
//
// The puzzle: Given twelve statements about themselves, determine which are true.
//
//  1. This is a numbered list of twelve statements.
//  2. Exactly 3 of the last 6 statements are true.
//  3. Exactly 2 of the even-numbered statements are true.
//  4. If statement 5 is true, then statements 6 and 7 are both true.
//  5. The 3 preceding statements are all false.
//  6. Exactly 4 of the odd-numbered statements are true.
//  7. Either statement 2 or 3 is true, but not both.
//  8. If statement 7 is true, then 5 and 6 are both true.
//  9. Exactly 3 of the first 6 statements are true.
//
// 10. The next two statements are both true.
// 11. Exactly 1 of statements 7, 8 and 9 are true.
// 12. Exactly 4 of the preceding statements are true.
package main

import (
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Twelve Statements Puzzle ===")
	fmt.Println()

	// Find all solutions
	results := RunStar(twelveStatements)

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Printf("✓ Found %d solution(s)!\n\n", len(results))

	for i, result := range results {
		if len(results) > 1 {
			fmt.Printf("Solution %d:\n", i+1)
		}
		displaySolution(result)
		if i < len(results)-1 {
			fmt.Println()
		}
	}
}

// twelveStatements defines the puzzle as a miniKanren goal.
// Each statement is either true (1) or false (0).
//
// Note: This puzzle demonstrates using Project as a constraint verification oracle.
// Unlike the Zebra and Apartment puzzles which use idiomatic relational programming,
// this problem requires counting and self-referential logic that doesn't map naturally
// to pure miniKanren. We use Project to extract all variable bindings and verify them
// in Go, essentially performing guided search over a finite boolean domain (2^12 possibilities).
//
// This is a pragmatic approach for constraint satisfaction problems that need:
// - Counting constraints ("exactly N of these are true")
// - Self-referential logic (statements about their own truth values)
// - Boolean satisfiability checking
//
// For more idiomatic miniKanren examples using relational helpers and constraint
// propagation, see the Zebra and Apartment puzzles.
func twelveStatements(q *Var) Goal {
	// Create variables for each statement's truth value
	s := make([]Term, 12)
	for i := 0; i < 12; i++ {
		s[i] = Fresh(fmt.Sprintf("s%d", i+1))
	}

	// Helper: constrain variable to be boolean (0 or 1)
	boolean := func(v Term) Goal {
		return Disj(Eq(v, NewAtom(0)), Eq(v, NewAtom(1)))
	}

	// Create solution list
	solution := List(s...)

	return Conj(
		Eq(q, solution),

		// All statements are boolean
		boolean(s[0]), boolean(s[1]), boolean(s[2]), boolean(s[3]),
		boolean(s[4]), boolean(s[5]), boolean(s[6]), boolean(s[7]),
		boolean(s[8]), boolean(s[9]), boolean(s[10]), boolean(s[11]),

		// Now use Project to verify all constraints together
		Project(s, func(vals []Term) Goal {
			// Helper to check if statement is true
			isTrue := func(idx int) bool {
				if atom, ok := vals[idx].(*Atom); ok {
					if v, ok := atom.Value().(int); ok {
						return v == 1
					}
				}
				return false
			}

			count := func(indices ...int) int {
				c := 0
				for _, idx := range indices {
					if isTrue(idx) {
						c++
					}
				}
				return c
			}

			// Statement 1: This is a numbered list of twelve statements (always true)
			if !isTrue(0) {
				return Failure
			}

			// Statement 2: Exactly 3 of the last 6 statements are true (7-12)
			stmt2Valid := count(6, 7, 8, 9, 10, 11) == 3
			if isTrue(1) != stmt2Valid {
				return Failure
			}

			// Statement 3: Exactly 2 of the even-numbered statements are true (2,4,6,8,10,12)
			stmt3Valid := count(1, 3, 5, 7, 9, 11) == 2
			if isTrue(2) != stmt3Valid {
				return Failure
			}

			// Statement 4: If statement 5 is true, then statements 6 and 7 are both true
			stmt4Valid := !isTrue(4) || (isTrue(5) && isTrue(6))
			if isTrue(3) != stmt4Valid {
				return Failure
			}

			// Statement 5: The 3 preceding statements are all false (2, 3, 4)
			stmt5Valid := !isTrue(1) && !isTrue(2) && !isTrue(3)
			if isTrue(4) != stmt5Valid {
				return Failure
			}

			// Statement 6: Exactly 4 of the odd-numbered statements are true (1,3,5,7,9,11)
			stmt6Valid := count(0, 2, 4, 6, 8, 10) == 4
			if isTrue(5) != stmt6Valid {
				return Failure
			}

			// Statement 7: Either statement 2 or 3 is true, but not both (XOR)
			stmt7Valid := isTrue(1) != isTrue(2) // XOR
			if isTrue(6) != stmt7Valid {
				return Failure
			}

			// Statement 8: If statement 7 is true, then 5 and 6 are both true
			stmt8Valid := !isTrue(6) || (isTrue(4) && isTrue(5))
			if isTrue(7) != stmt8Valid {
				return Failure
			}

			// Statement 9: Exactly 3 of the first 6 statements are true (1-6)
			stmt9Valid := count(0, 1, 2, 3, 4, 5) == 3
			if isTrue(8) != stmt9Valid {
				return Failure
			}

			// Statement 10: The next two statements are both true (11 and 12)
			stmt10Valid := isTrue(10) && isTrue(11)
			if isTrue(9) != stmt10Valid {
				return Failure
			}

			// Statement 11: Exactly 1 of statements 7, 8 and 9 are true
			stmt11Valid := count(6, 7, 8) == 1
			if isTrue(10) != stmt11Valid {
				return Failure
			}

			// Statement 12: Exactly 4 of the preceding statements are true (1-11)
			stmt12Valid := count(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) == 4
			if isTrue(11) != stmt12Valid {
				return Failure
			}

			// All constraints satisfied
			return Success
		}),
	)
}

// displaySolution pretty-prints the solution
func displaySolution(result Term) {
	statements := []string{
		"This is a numbered list of twelve statements.",
		"Exactly 3 of the last 6 statements are true.",
		"Exactly 2 of the even-numbered statements are true.",
		"If statement 5 is true, then statements 6 and 7 are both true.",
		"The 3 preceding statements are all false.",
		"Exactly 4 of the odd-numbered statements are true.",
		"Either statement 2 or 3 is true, but not both.",
		"If statement 7 is true, then 5 and 6 are both true.",
		"Exactly 3 of the first 6 statements are true.",
		"The next two statements are both true.",
		"Exactly 1 of statements 7, 8 and 9 are true.",
		"Exactly 4 of the preceding statements are true.",
	}

	pair, ok := result.(*Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	var trueStatements []int
	var falseStatements []int
	idx := 1

	for pair != nil {
		valTerm := pair.Car()
		if atom, ok := valTerm.(*Atom); ok {
			if v, ok := atom.Value().(int); ok {
				if v == 1 {
					trueStatements = append(trueStatements, idx)
				} else {
					falseStatements = append(falseStatements, idx)
				}
			}
		}

		pair, _ = pair.Cdr().(*Pair)
		idx++
	}

	fmt.Println("TRUE statements:")
	for _, n := range trueStatements {
		fmt.Printf("%2d. %s\n", n, statements[n-1])
	}

	fmt.Println("\nFALSE statements:")
	for _, n := range falseStatements {
		fmt.Printf("%2d. %s\n", n, statements[n-1])
	}

	fmt.Printf("\n✅ %d true, %d false\n", len(trueStatements), len(falseStatements))
}

```

## Running the Example

To run this example:

```bash
cd twelve-statements
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
