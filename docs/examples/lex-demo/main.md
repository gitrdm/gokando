# main

This example demonstrates basic usage of the library.

## Source Code

```go
// Lex-demo demonstrates the lexicographic ordering constraint (X ≤lex Y).
//
// This example shows how to use the NewLexLessEq constraint to enforce that
// one vector of finite domain variables is lexicographically less than or
// equal to another vector. The constraint ensures that X[0] < Y[0], or
// X[0] = Y[0] and X[1] < Y[1], or X[0] = Y[0] and X[1] = Y[1] and so on.
//
// The example creates two vectors X = [x1, x2] and Y = [y1, y2] with specific
// domain values, applies the lexicographic constraint, and shows the domain
// pruning that occurs during constraint propagation.
//
// HLAPI Features Used:
//   - IntVarValues() - Create FD variables with specific domain values
//   - NewModel() - Create constraint model
//   - NewLexLessEq() - Lexicographic ordering constraint
//   - NewSolver() - Create FD solver
//   - Solve() - Run constraint propagation
package main

import (
	"context"
	"fmt"
	"time"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// A tiny demo of lexicographic ordering X ≤lex Y.
func main() {
	fmt.Println("=== Lexicographic Constraint Demo (X ≤lex Y) ===")
	fmt.Println("\nInitial domains:")
	fmt.Println("  X = [x1, x2] where x1 ∈ {2,3,4}, x2 ∈ {1,2,3}")
	fmt.Println("  Y = [y1, y2] where y1 ∈ {3,4,5}, y2 ∈ {2,3,4}")
	fmt.Println("\nConstraint: X ≤lex Y")
	fmt.Println("(X is lexicographically less than or equal to Y)")

	model := NewModel()

	// Use HLAPI IntVarValues to create variables with specific domain values
	x1 := model.IntVarValues([]int{2, 3, 4}, "x1")
	x2 := model.IntVarValues([]int{1, 2, 3}, "x2")
	y1 := model.IntVarValues([]int{3, 4, 5}, "y1")
	y2 := model.IntVarValues([]int{2, 3, 4}, "y2")

	c, err := NewLexLessEq([]*FDVariable{x1, x2}, []*FDVariable{y1, y2})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := NewSolver(model)

	// Find all solutions
	fmt.Println("\nFinding all solutions...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	solutions, _ := solver.Solve(ctx, -1) // -1 means find all solutions

	fmt.Printf("\n✓ Found %d valid solutions:\n\n", len(solutions))

	for i, sol := range solutions {
		// Solutions map uses variable IDs as keys
		x1Val := sol[x1.ID()]
		x2Val := sol[x2.ID()]
		y1Val := sol[y1.ID()]
		y2Val := sol[y2.ID()]

		fmt.Printf("  %2d. X=[%d,%d] ≤lex Y=[%d,%d]", i+1, x1Val, x2Val, y1Val, y2Val)

		// Explain why this satisfies the constraint
		if x1Val < y1Val {
			fmt.Printf("  ✓ (x1=%d < y1=%d)\n", x1Val, y1Val)
		} else if x1Val == y1Val {
			fmt.Printf("  ✓ (x1=y1=%d and x2=%d ≤ y2=%d)\n", x1Val, x2Val, y2Val)
		}
	}

	// Calculate how many combinations were ruled out
	totalCombinations := 3 * 3 * 3 * 3 // |x1| * |x2| * |y1| * |y2|
	invalidCombinations := totalCombinations - len(solutions)

	fmt.Printf("\nConstraint filtered out %d invalid combinations (kept %d/%d)\n",
		invalidCombinations, len(solutions), totalCombinations)
}

```

## Running the Example

To run this example:

```bash
cd lex-demo
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
