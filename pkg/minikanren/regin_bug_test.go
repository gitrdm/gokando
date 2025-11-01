package minikanren

import (
	"testing"
)

// TestReginStaircaseBug reproduces the bug found in the N-Queens test,
// where "staircase" domains caused incorrect pruning.
func TestReginStaircaseBug(t *testing.T) {
	model := NewModel()

	// Reproduce the exact state from the failing test
	// Second iteration diagonal domains:
	//   var[0]: [1 2 3 4]
	//   var[1]: [2 3 4 5]
	//   var[2]: [3 4 5 6]
	//   var[3]: [4 5 6 7]

	vars := make([]*FDVariable, 4)
	for i := 0; i < 4; i++ {
		// The original test used a max value of 8.
		vars[i] = model.NewVariable(NewBitSetDomain(8))
	}

	// Set up the "staircase" domains
	solver := NewSolver(model)
	var state *SolverState // Initial state is nil

	for i := 0; i < 4; i++ {
		vals := make([]int, 4)
		for j := 0; j < 4; j++ {
			vals[j] = i + j + 1
		}
		dom := NewBitSetDomainFromValues(8, vals)
		state = solver.SetDomain(state, vars[i].ID(), dom)
	}

	// Now apply AllDifferent
	c, err := NewAllDifferent(vars)
	if err != nil {
		t.Fatalf("Failed to create AllDifferent constraint: %v", err)
	}

	newState, err := c.Propagate(solver, state)
	if err != nil {
		t.Fatalf("AllDifferent.Propagate failed unexpectedly: %v", err)
	}

	// Check the resulting domains.
	// With these staircase domains, no values should be pruned.
	// For example, v0=1 is valid with {v1=2, v2=3, v3=4}.
	// And v0=4 is valid with {v1=2, v2=3, v3=5}.
	// All initial values should have a valid support.
	for i := 0; i < 4; i++ {
		dom := solver.GetDomain(newState, vars[i].ID())
		if dom.Count() != 4 {
			t.Errorf("var[%d] was pruned incorrectly. Expected 4 values, got %d. Domain: %v", i, dom.Count(), dom)
		}
	}
}
