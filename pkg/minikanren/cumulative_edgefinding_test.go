package minikanren

import (
	"context"
	"testing"
)

// Construct a small instance where energetic overload is detectable in a window.
func TestCumulative_EnergeticOverloadDetection(t *testing.T) {
	model := NewModel()
	// Three tasks, duration 3, demand 2 each, capacity 4.
	// Windows of length 3 cannot host all three: minimal energy 3*3*2=18 over L=3 ⇒ cap*L=12 ⇒ overload.
	a := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2}))
	b := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2}))
	c := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2}))

	cum, err := NewCumulative([]*FDVariable{a, b, c}, []int{3, 3, 3}, []int{2, 2, 2}, 4)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)

	solver := NewSolver(model)
	// On root-level inconsistency, Solve returns zero solutions without error.
	sols, err := solver.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error from Solve: %v", err)
	}
	if len(sols) != 0 {
		t.Fatalf("expected no solutions due to energetic overload, got %d", len(sols))
	}
}

// Edge-finding-like pruning: if placing a task fully inside a tight window would overload,
// exclude its starts there while keeping a feasible value outside the window.
func TestCumulative_EdgeFindingPrunesStarts(t *testing.T) {
	model := NewModel()
	// Capacity 3. T2 is fixed at start=4 with dur=3,dem=2. T1 can be {1,4} with dur=3,dem=2 and
	// will be pruned to 1 (to avoid overlap with T2). With those bounds, in window [2..4] (L=3)
	// the minimal energy of T1,T2 is 6. Placing K fully inside [2..4] requires 4 additional energy,
	// which would overload the window (6+4 > 3*3). Therefore, K's starts {2,3} should be pruned,
	// leaving only 7 which lies outside the window and admits a feasible schedule.
	t1 := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 4}))
	t2 := model.NewVariable(NewBitSetDomainFromValues(10, []int{4}))
	k := model.NewVariable(NewBitSetDomainFromValues(10, []int{2, 3, 7}))

	cum, err := NewCumulative([]*FDVariable{t1, t2, k}, []int{3, 3, 2}, []int{2, 2, 2}, 3)
	if err != nil {
		t.Fatalf("NewCumulative error: %v", err)
	}
	model.AddConstraint(cum)

	solver := NewSolver(model)
	sols, err := solver.Solve(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected solve error: %v", err)
	}
	if len(sols) == 0 {
		t.Fatalf("expected at least one solution after pruning, got none")
	}

	// Expect K's starts no longer allow {2,3} fully inside overloaded window; only 7 should remain.
	dK := solver.GetDomain(nil, k.ID())
	want := NewBitSetDomainFromValues(dK.MaxValue(), []int{7})
	if !dK.Equal(want) {
		t.Fatalf("expected K domain to be %v, got %v", want, dK)
	}
}
