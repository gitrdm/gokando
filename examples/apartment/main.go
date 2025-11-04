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
	"context"
	"fmt"

	minikanren "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Apartment Floor Puzzle ===")
	fmt.Println()

	// Build FD model using HLAPI
	m := minikanren.NewModel()

	// Create FD variables for each person's floor (1-5)
	baker := m.IntVar(1, 5, "baker")
	cooper := m.IntVar(1, 5, "cooper")
	fletcher := m.IntVar(1, 5, "fletcher")
	miller := m.IntVar(1, 5, "miller")
	smith := m.IntVar(1, 5, "smith")

	// All people must be on different floors (HLAPI global constraint)
	m.AllDifferent(baker, cooper, fletcher, miller, smith)

	// Baker does not live on the top floor (5)
	top := m.IntVar(5, 5, "top")
	c1, _ := minikanren.NewInequality(baker, top, minikanren.NotEqual)
	m.AddConstraint(c1)

	// Cooper does not live on the bottom floor (1)
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
	m.AddConstraint(notAdjacentConstraint(m, smith, fletcher))

	// Fletcher not adjacent to Cooper
	m.AddConstraint(notAdjacentConstraint(m, fletcher, cooper))

	// Solve using HLAPI
	ctx := context.Background()
	solver := minikanren.NewSolver(m)
	solutions, err := solver.Solve(ctx, 1)
	if err != nil {
		fmt.Printf("❌ Error solving: %v\n", err)
		return
	}

	if len(solutions) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Println("✓ Solution found!")
	fmt.Println()

	displaySolution(solutions[0], baker, cooper, fletcher, miller, smith)
}

// notAdjacentConstraint builds model constraints for "A not adjacent to B"
// using FD variables and HLAPI constructors.
func notAdjacentConstraint(m *minikanren.Model, a, b *minikanren.FDVariable) minikanren.ModelConstraint {
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

	return newCompositeModelConstraint([]minikanren.ModelConstraint{c1, c_ar1, c_neq1, c_ar2, c_neq2})
}

// newCompositeModelConstraint combines multiple ModelConstraints into one.
func newCompositeModelConstraint(children []minikanren.ModelConstraint) minikanren.ModelConstraint {
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

// displaySolution pretty-prints the FD solution
func displaySolution(solution []int, baker, cooper, fletcher, miller, smith *minikanren.FDVariable) {
	fmt.Println("Person    | Floor")
	fmt.Println("----------|------")

	people := []struct {
		name string
		v    *minikanren.FDVariable
	}{
		{"Baker", baker},
		{"Cooper", cooper},
		{"Fletcher", fletcher},
		{"Miller", miller},
		{"Smith", smith},
	}

	for _, p := range people {
		fmt.Printf("%-9s | %d\n", p.name, solution[p.v.ID()])
	}

	fmt.Println()
	fmt.Println("✅ All constraints satisfied!")
}
