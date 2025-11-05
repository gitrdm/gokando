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

// Table Demo: extensional constraint over (Color, Pet, Drink)
//
// We define three enumerated variables:
//   - Color ∈ {Red, Green, Blue}
//   - Pet   ∈ {Cat, Dog, Bird}
//   - Drink ∈ {Tea, Coffee, Water}
//
// and restrict the triple (Color, Pet, Drink) to be one of a small set of
// allowed tuples via the Table global constraint.
//
// The solver enumerates all satisfying triples and prints them in a
// human-readable form.
func main() {
	fmt.Println("=== Table Constraint Demo (Color, Pet, Drink) ===")

	// Enumerations (1-based to match FD domain invariants)
	const (
		Red = 1 + iota
		Green
		Blue
	)
	colorNames := map[int]string{Red: "Red", Green: "Green", Blue: "Blue"}

	const (
		Cat = 1 + iota
		Dog
		Bird
	)
	petNames := map[int]string{Cat: "Cat", Dog: "Dog", Bird: "Bird"}

	const (
		Tea = 1 + iota
		Coffee
		Water
	)
	drinkNames := map[int]string{Tea: "Tea", Coffee: "Coffee", Water: "Water"}

	model := mk.NewModel()

	color := model.NewVariableWithName(mk.NewBitSetDomain(3), "Color")
	pet := model.NewVariableWithName(mk.NewBitSetDomain(3), "Pet")
	drink := model.NewVariableWithName(mk.NewBitSetDomain(3), "Drink")

	// Allowed rows (Color, Pet, Drink)
	rows := [][]int{
		{Red, Dog, Coffee},
		{Green, Bird, Tea},
		{Blue, Cat, Water},
		{Blue, Dog, Tea},
	}

	table, err := mk.NewTable([]*mk.FDVariable{color, pet, drink}, rows)
	if err != nil {
		panic(err)
	}
	model.AddConstraint(table)

	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	solutions, err := solver.Solve(ctx, 50)
	if err != nil {
		fmt.Printf("Solve error: %v\n", err)
		return
	}
	if len(solutions) == 0 {
		fmt.Println("No solutions found (unexpected with non-empty table)")
		return
	}

	fmt.Printf("Found %d solutions:\n", len(solutions))
	for _, sol := range solutions {
		c := sol[color.ID()]
		p := sol[pet.ID()]
		d := sol[drink.ID()]
		fmt.Printf("  Color=%s, Pet=%s, Drink=%s\n", colorNames[c], petNames[p], drinkNames[d])
	}
}

```

## Running the Example

To run this example:

```bash
cd table-demo
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
