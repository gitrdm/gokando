# main

This example demonstrates basic usage of the library.

## Source Code

```go
package main

import (
	"context"
	"fmt"

	minikanren "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
	fmt.Println("=== Apartment puzzle (FD variant) ===")

	// Build FD model
	m := minikanren.NewModel()

	baker := m.IntVar(1, 5, "baker")
	cooper := m.IntVar(1, 5, "cooper")
	fletcher := m.IntVar(1, 5, "fletcher")
	miller := m.IntVar(1, 5, "miller")
	smith := m.IntVar(1, 5, "smith")

	// Global AllDifferent (HLAPI)
	_ = m.AllDifferent(baker, cooper, fletcher, miller, smith)

	// Baker does not live on the top floor (5)
	top := m.IntVar(5, 5, "top")
	c1, _ := minikanren.NewInequality(baker, top, minikanren.NotEqual)
	m.AddConstraint(c1)
	bottom := m.IntVar(1, 1, "bottom")
	c2, _ := minikanren.NewInequality(cooper, bottom, minikanren.NotEqual)
	m.AddConstraint(c2)

	// Fletcher does not live on top or bottom
	c3, _ := minikanren.NewInequality(fletcher, top, minikanren.NotEqual)
	m.AddConstraint(c3)
	c4, _ := minikanren.NewInequality(fletcher, bottom, minikanren.NotEqual)
	m.AddConstraint(c4)

	// Miller lives on a higher floor than Cooper: cooper < miller
	c5, _ := minikanren.NewInequality(cooper, miller, minikanren.LessThan)
	m.AddConstraint(c5)

	// Smith not adjacent to Fletcher
	m.AddConstraint(mustNotAdjacentConstraint(m, smith, fletcher))

	// Fletcher not adjacent to Cooper
	m.AddConstraint(mustNotAdjacentConstraint(m, fletcher, cooper))

	// Build hybrid solver and populated UnifiedStore via HLAPI helper
	solver, store, err := minikanren.NewHybridSolverFromModel(m)
	if err != nil {
		panic(err)
	}

	// Run propagation to a fixed point
	result, err := solver.Propagate(store)
	if err != nil {
		panic(err)
	}

	// Print resulting domains
	printDomain := func(name string, v *minikanren.FDVariable) {
		d := result.GetDomain(v.ID())
		if d == nil {
			fmt.Printf("%s: <nil>\n", name)
			return
		}
		fmt.Printf("%s: %s\n", name, d.String())
	}

	printDomain("baker", baker)
	printDomain("cooper", cooper)
	printDomain("fletcher", fletcher)
	printDomain("miller", miller)
	printDomain("smith", smith)

	// Now run a full solver search to produce a concrete assignment (one solution)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	searcher := minikanren.NewSolver(m)
	solutions, err := searcher.Solve(ctx, 1)
	if err != nil {
		panic(err)
	}

	if len(solutions) == 0 {
		fmt.Println("No solution found by search")
		return
	}

	// Print the concrete assignment found by the solver
	fmt.Println()
	fmt.Println("Concrete assignment (after search):")
	fmt.Println("Person    | Floor")
	fmt.Println("----------|------")

	sol := solutions[0]
	// Print only the named person variables in the original order
	people := []*minikanren.FDVariable{baker, cooper, fletcher, miller, smith}
	for _, v := range people {
		name := v.Name()
		val := sol[v.ID()]
		fmt.Printf("%-9s | %d\n", name, val)
	}
}

// mustNotAdjacentConstraint builds the three model constraints that
// implement "A not adjacent to B" for FD variables using HLAPI constructors.
// It returns a ModelConstraint that is a conjunction of the three constraints.
func mustNotAdjacentConstraint(m *minikanren.Model, a, b *minikanren.FDVariable) minikanren.ModelConstraint {
	// A != B
	c1, _ := minikanren.NewInequality(a, b, minikanren.NotEqual)

	// A != B + 1
	bplus := m.IntVar(1, 5, "")
	c_ar1, _ := minikanren.NewArithmetic(b, bplus, 1)
	c_neq1, _ := minikanren.NewInequality(a, bplus, minikanren.NotEqual)

	// A != B - 1
	bminus := m.IntVar(1, 5, "")
	c_ar2, _ := minikanren.NewArithmetic(b, bminus, -1)
	c_neq2, _ := minikanren.NewInequality(a, bminus, minikanren.NotEqual)

	return NewCompositeModelConstraint([]minikanren.ModelConstraint{c1, c_ar1, c_neq1, c_ar2, c_neq2})
}

// NewCompositeModelConstraint composes multiple ModelConstraints into one
// ModelConstraint for example convenience.
func NewCompositeModelConstraint(children []minikanren.ModelConstraint) minikanren.ModelConstraint {
	return &compositeModelConstraint{children: children}
}

type compositeModelConstraint struct{ children []minikanren.ModelConstraint }

func (c *compositeModelConstraint) Variables() []*minikanren.FDVariable {
	var out []*minikanren.FDVariable
	seen := make(map[int]bool)
	for _, ch := range c.children {
		for _, v := range ch.Variables() {
			if !seen[v.ID()] {
				out = append(out, v)
				seen[v.ID()] = true
			}
		}
	}
	return out
}

func (c *compositeModelConstraint) Type() string { return "Composite" }

func (c *compositeModelConstraint) String() string { return "CompositeConstraint" }

func (c *compositeModelConstraint) Clone() minikanren.ModelConstraint {
	copyChildren := make([]minikanren.ModelConstraint, len(c.children))
	copy(copyChildren, c.children)
	return &compositeModelConstraint{children: copyChildren}
}

```

## Running the Example

To run this example:

```bash
cd apartment_fd
go run main.go
```

## Expected Output

```
Hello from Proton examples!
```
