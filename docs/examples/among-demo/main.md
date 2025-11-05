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

// Among Demo: count how many variables take values from a set S.
// S = {1,2,3}. x1 is already within S; we set K to encode exactly 1 counted
// variable, forcing other variables to be outside S.
func main() {
	fmt.Println("=== Among Constraint Demo (count in S) ===")

	model := mk.NewModel()

	x1 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(5, []int{1, 2}), "x1")
	x2 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(5, []int{2, 3}), "x2")
	x3 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(5, []int{3, 4}), "x3")
	k := model.NewVariableWithName(mk.NewBitSetDomainFromValues(4, []int{2}), "K") // encode count=1

	// S = {1,2}
	c, err := mk.NewAmong([]*mk.FDVariable{x1, x2, x3}, []int{1, 2}, k)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Println("x1:", solver.GetDomain(nil, x1.ID()))
	fmt.Println("x2:", solver.GetDomain(nil, x2.ID()))
	fmt.Println("x3:", solver.GetDomain(nil, x3.ID()))
	fmt.Println("K:", solver.GetDomain(nil, k.ID()))
}

```

## Running the Example

To run this example:

```bash
cd among-demo
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
