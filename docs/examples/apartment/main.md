# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Package main solves the apartment floor puzzle using GoKando.
//
// The puzzle: Baker, Cooper, Fletcher, Miller, and Smith live on different
// floors of an apartment house that contains only five floors.
//
// Constraints:
//   - Baker does not live on the top floor.
//   - Cooper does not live on the bottom floor.
//   - Fletcher does not live on either the top or the bottom floor.
//   - Miller lives on a higher floor than does Cooper.
//   - Smith does not live on a floor adjacent to Fletcher's.
//   - Fletcher does not live on a floor adjacent to Cooper's.
//
// Question: Where does everyone live?
package main

import (
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Apartment Floor Puzzle ===")
	fmt.Println()

	// Solve the puzzle - we expect exactly one solution
	results := Run(1, floorPuzzle)

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Println("✓ Solution found!")
	fmt.Println()

	displaySolution(results[0])
}

// floorPuzzle defines the complete puzzle as a miniKanren goal.
// Each person is assigned a floor (1-5, where 5 is the top floor).
func floorPuzzle(q *Var) Goal {
	// Create variables for each person's floor
	baker := Fresh("baker")
	cooper := Fresh("cooper")
	fletcher := Fresh("fletcher")
	miller := Fresh("miller")
	smith := Fresh("smith")

	// Create the solution structure
	solution := List(
		List(NewAtom("Baker"), baker),
		List(NewAtom("Cooper"), cooper),
		List(NewAtom("Fletcher"), fletcher),
		List(NewAtom("Miller"), miller),
		List(NewAtom("Smith"), smith),
	)

	// Valid floors are 1-5
	floors := []Term{
		NewAtom(1),
		NewAtom(2),
		NewAtom(3),
		NewAtom(4),
		NewAtom(5),
	}

	// Helper to check if a variable is in the valid floor range
	validFloor := func(floor Term) Goal {
		goals := make([]Goal, len(floors))
		for i, f := range floors {
			goals[i] = Eq(floor, f)
		}
		return Disj(goals...)
	}

	// Helper to check if one floor is higher than another
	// Uses Project to extract values and compare arithmetically
	higherThan := func(floor1, floor2 Term) Goal {
		return Project([]Term{floor1, floor2}, func(vals []Term) Goal {
			f1, ok1 := vals[0].(*Atom)
			f2, ok2 := vals[1].(*Atom)
			if !ok1 || !ok2 {
				return Failure
			}

			v1, ok1 := f1.Value().(int)
			v2, ok2 := f2.Value().(int)
			if !ok1 || !ok2 {
				return Failure
			}

			if v1 > v2 {
				return Success
			}
			return Failure
		})
	}

	// Helper to check if two floors are NOT adjacent
	// Uses Project to extract values and check difference > 1
	notAdjacent := func(floor1, floor2 Term) Goal {
		return Project([]Term{floor1, floor2}, func(vals []Term) Goal {
			f1, ok1 := vals[0].(*Atom)
			f2, ok2 := vals[1].(*Atom)
			if !ok1 || !ok2 {
				return Failure
			}

			v1, ok1 := f1.Value().(int)
			v2, ok2 := f2.Value().(int)
			if !ok1 || !ok2 {
				return Failure
			}

			diff := v1 - v2
			if diff < 0 {
				diff = -diff
			}

			if diff > 1 {
				return Success
			}
			return Failure
		})
	}

	return Conj(
		// Return the solution structure
		Eq(q, solution),

		// Each person must be on a valid floor
		validFloor(baker),
		validFloor(cooper),
		validFloor(fletcher),
		validFloor(miller),
		validFloor(smith),

		// All people must be on different floors
		allDiff(baker, cooper, fletcher, miller, smith),

		// Constraint 1: Baker does not live on the top floor
		Neq(baker, NewAtom(5)),

		// Constraint 2: Cooper does not live on the bottom floor
		Neq(cooper, NewAtom(1)),

		// Constraint 3: Fletcher does not live on either the top or the bottom floor
		Neq(fletcher, NewAtom(1)),
		Neq(fletcher, NewAtom(5)),

		// Constraint 4: Miller lives on a higher floor than does Cooper
		higherThan(miller, cooper),

		// Constraint 5: Smith does not live on a floor adjacent to Fletcher's
		notAdjacent(smith, fletcher),

		// Constraint 6: Fletcher does not live on a floor adjacent to Cooper's
		notAdjacent(fletcher, cooper),
	)
}

// allDiff ensures all terms are different from each other
func allDiff(terms ...Term) Goal {
	var goals []Goal
	for i := 0; i < len(terms); i++ {
		for j := i + 1; j < len(terms); j++ {
			goals = append(goals, Neq(terms[i], terms[j]))
		}
	}
	return Conj(goals...)
}

// displaySolution pretty-prints the puzzle solution
func displaySolution(result Term) {
	pair, ok := result.(*Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	fmt.Println("Person    | Floor")
	fmt.Println("----------|------")

	// Extract each person-floor pair
	for pair != nil {
		personFloorPair := pair.Car()
		if personFloorPair == nil {
			break
		}

		pairObj, ok := personFloorPair.(*Pair)
		if !ok {
			break
		}

		name := extractAtom(pairObj.Car())
		pairObj, _ = pairObj.Cdr().(*Pair)
		if pairObj == nil {
			break
		}

		floor := extractAtom(pairObj.Car())

		fmt.Printf("%-9s | %s\n", name, floor)

		pair, _ = pair.Cdr().(*Pair)
	}

	fmt.Println()
	fmt.Println("✅ All constraints satisfied!")
}

// extractAtom extracts the string value from an Atom term
func extractAtom(term Term) string {
	if atom, ok := term.(*Atom); ok {
		return fmt.Sprintf("%v", atom.Value())
	}
	return "?"
}

```

## Running the Example

To run this example:

```bash
cd apartment
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
