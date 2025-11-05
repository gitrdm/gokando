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

// Global Cardinality (GCC) Demo: simple assignment with value usage bounds.
//
// Three variables over {1,2,3}; enforce that value 1 is used exactly once and
// value 2 at most twice. Enumerates a subset of feasible assignments.
func main() {
	fmt.Println("=== Global Cardinality Constraint Demo ===")

	model := mk.NewModel()
	vars := make([]*mk.FDVariable, 3)
	for i := 0; i < 3; i++ {
		vars[i] = model.NewVariableWithName(mk.NewBitSetDomain(3), fmt.Sprintf("X%d", i+1))
	}

	min := make([]int, 4)
	max := make([]int, 4)
	min[1], max[1] = 1, 1 // value 1 exactly once
	min[2], max[2] = 0, 2 // value 2 at most twice
	min[3], max[3] = 0, 3 // value 3 otherwise

	gcc, err := mk.NewGlobalCardinality(vars, min, max)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(gcc)

	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 50)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(solutions) == 0 {
		fmt.Println("No solutions found")
		return
	}

	fmt.Printf("Found %d feasible assignments (showing up to %d):\n", len(solutions), 50)
	for _, sol := range solutions {
		for i, v := range vars {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("%s=%d", v.Name(), sol[v.ID()])
		}
		fmt.Println()
	}
}

```

## Running the Example

To run this example:

```bash
cd gcc-demo
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
