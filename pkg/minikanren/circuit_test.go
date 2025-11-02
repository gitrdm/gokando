package minikanren

import (
	"testing"
)

// TestCircuit_BasicShapes verifies initial shaping:
// - self loops are forbidden (succ[i] domain does not contain i)
// - order variables domains are set as specified (u[start]=1, others in [2..n])
func TestCircuit_BasicShapes(t *testing.T) {
	model := NewModel()
	n := 4

	// succ[i] ∈ [1..n]
	succ := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		succ[i] = model.NewVariable(NewBitSetDomain(n))
	}

	circ, err := NewCircuit(model, succ, 1)
	if err != nil {
		t.Fatalf("NewCircuit failed: %v", err)
	}
	if circ == nil {
		t.Fatalf("NewCircuit returned nil")
	}

	solver := NewSolver(model)

	// Run a single propagation to apply immediate pruning
	state, err := solver.propagate(nil)
	if err != nil {
		t.Fatalf("propagation failed: %v", err)
	}

	// Check self loops removed
	for i := 0; i < n; i++ {
		d := solver.GetDomain(state, succ[i].ID())
		if d == nil || d.Count() == 0 {
			t.Fatalf("succ[%d] has nil/empty domain", i+1)
		}
		if d.Has(i + 1) {
			t.Errorf("succ[%d] should not contain self index %d, got %s", i+1, i+1, d.String())
		}
	}

	// Order vars are internal; we can spot-check via their naming in the model
	// However, they are not exposed. We instead assert that no error occurred
	// and rely on subtour test below for functional coverage.
}

// TestCircuit_SubtourEliminationConflict forces a 2-cycle not involving start
// and expects propagation to detect inconsistency (no such circuit exists).
func TestCircuit_SubtourEliminationConflict(t *testing.T) {
	model := NewModel()
	n := 4
	succ := make([]*FDVariable, n)
	for i := 0; i < n; i++ {
		succ[i] = model.NewVariable(NewBitSetDomain(n))
	}

	_, err := NewCircuit(model, succ, 1) // start at node 1
	if err != nil {
		t.Fatalf("NewCircuit failed: %v", err)
	}

	solver := NewSolver(model)

	// Force a 2-cycle between nodes 2 and 3: 2->3 and 3->2
	state := (*SolverState)(nil)
	// Remove all but {3} from succ[2]
	state, _ = solver.SetDomain(state, succ[1].ID(), NewBitSetDomainFromValues(n, []int{3}))
	// Remove all but {2} from succ[3]
	state, _ = solver.SetDomain(state, succ[2].ID(), NewBitSetDomainFromValues(n, []int{2}))

	// Propagation should fail because the reified order constraints imply
	// u3 = u2 + 1 and u2 = u3 + 1, which is inconsistent over integers.
	if _, err := solver.propagate(state); err == nil {
		t.Fatalf("expected propagation conflict for 2-cycle subtour, got no error")
	}
}
