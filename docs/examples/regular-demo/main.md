# main

This example demonstrates basic usage of the library.

## Source Code

```go
package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Regular Demo: pattern checker using a DFA (ends-with-A)
//
// We use the Regular global constraint to enforce that a sequence of symbols
// over the alphabet {A=1, B=2} forms a word accepted by a DFA. The DFA here
// accepts exactly the strings that end with A. With length=3, the accepted
// sequences are: AAA, BAA, ABA, BBA.
//
// The Regular constraint performs strong pruning: it reduces the last position
// to {A} immediately, and the solver enumerates the remaining degrees of
// freedom to list all accepted words.
func main() {
	fmt.Println("=== Regular Constraint Demo (DFA: ends with A) ===")

	// Alphabet encoding (1-based to match FD domain invariants)
	const (
		A = 1
		B = 2
	)
	sym := map[int]string{A: "A", B: "B"}

	// Build a simple DFA over {A,B} that accepts exactly the strings
	// that end with A.
	//
	// States (1-based):
	//   1: start
	//   2: last seen symbol is A (accepting)
	//   3: last seen symbol is B (non-accepting)
	// Transition table delta[s][v] = nextState; index 0 unused for 1-based symbols.
	numStates := 3
	start := 1
	acceptStates := []int{2}
	delta := [][]int{
		/* from state 1 */ {0, 2, 3},
		/* from state 2 */ {0, 2, 3},
		/* from state 3 */ {0, 2, 3},
	}

	// Problem length
	n := 3
	model := mk.NewModel()
	vars := make([]*mk.FDVariable, n)
	for i := 0; i < n; i++ {
		vars[i] = model.NewVariableWithName(mk.NewBitSetDomain(2), fmt.Sprintf("x%d", i+1))
	}

	// Post Regular constraint
	reg, err := mk.NewRegular(vars, numStates, start, acceptStates, delta)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(reg)

	// Solve and enumerate accepted sequences
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 50)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(solutions) == 0 {
		fmt.Println("No solutions found (unexpected)")
		return
	}

	fmt.Printf("Found %d accepted sequences (length=%d):\n", len(solutions), n)
	for _, sol := range solutions {
		// Print as symbols
		for i, v := range vars {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(sym[sol[v.ID()]])
		}
		fmt.Println()
	}
}

```

## Running the Example

To run this example:

```bash
cd regular-demo
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
