package minikanren

import (
	"context"
	"testing"
)

// TestCount_Distribution enumerates all assignments and checks the distribution
// of counts for a small case: 3 vars in {1,2,3}, counting value=2.
func TestCount_Distribution(t *testing.T) {
	model := NewModel()
	// X,Y,Z in {1..3}
	dom := NewBitSetDomain(3)
	vars := []*FDVariable{
		model.NewVariable(dom),
		model.NewVariable(dom),
		model.NewVariable(dom),
	}
	// countVar encodes count+1, so domain is [1..4] for n=3
	countVar := model.NewVariable(NewBitSetDomain(4))

	c, err := NewCount(model, vars, 2, countVar)
	if err != nil {
		t.Fatalf("NewCount failed: %v", err)
	}
	// Count itself is a constraint; add it for completeness (no-op propagate)
	model.AddConstraint(c)

	solver := NewSolver(model)
	solutions, err := solver.Solve(context.Background(), 1000)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if len(solutions) != 27 { // 3^3
		t.Fatalf("Expected 27 solutions, got %d", len(solutions))
	}

	// Distribution: Binomial(n=3, p=1/3)
	// counts: 0->8, 1->12, 2->6, 3->1
	got := map[int]int{}
	for _, sol := range solutions {
		cnt := sol[countVar.ID()] - 1 // decode
		got[cnt]++
		// Validate the count semantics
		actual := 0
		for _, v := range vars {
			if sol[v.ID()] == 2 {
				actual++
			}
		}
		if actual != cnt {
			t.Fatalf("Solution %v has inconsistent count: reported=%d actual=%d", sol, cnt, actual)
		}
	}

	want := map[int]int{0: 8, 1: 12, 2: 6, 3: 1}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("count=%d: got %d, want %d", k, got[k], w)
		}
	}
}

// TestCount_ForcedExtremes checks propagation when count is forced to 0 or n.
func TestCount_ForcedExtremes(t *testing.T) {
	model := NewModel()
	dom := NewBitSetDomain(5) // {1..5}
	vars := []*FDVariable{
		model.NewVariable(dom),
		model.NewVariable(dom),
		model.NewVariable(dom),
	}
	countVar0 := model.NewVariable(NewBitSetDomainFromValues(4, []int{1})) // count=0 => encoded 1

	// Case 1: count=0 => all vars != 5
	c0, err := NewCount(model, vars, 5, countVar0)
	if err != nil {
		t.Fatalf("NewCount failed: %v", err)
	}
	model.AddConstraint(c0)

	solver := NewSolver(model)
	_, err = solver.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	// Check propagation: each var domain should exclude 5
	for i, v := range vars {
		d := solver.GetDomain(nil, v.ID())
		if d.Has(5) {
			t.Errorf("var[%d] still has value 5 despite count=0", i)
		}
	}

	// Case 2: force all equal to 5 (count=3)
	model2 := NewModel()
	vars2 := []*FDVariable{
		model2.NewVariable(dom),
		model2.NewVariable(dom),
		model2.NewVariable(dom),
	}
	// Create count variable in the same model (encoded 4 for count=3)
	countVar3_2 := model2.NewVariable(NewBitSetDomainFromValues(4, []int{4}))
	c3, err := NewCount(model2, vars2, 5, countVar3_2)
	if err != nil {
		t.Fatalf("NewCount failed: %v", err)
	}
	model2.AddConstraint(c3)
	// Solving should propagate all vars to singleton {5}
	solver2 := NewSolver(model2)
	_, err = solver2.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	for i, v := range vars2 {
		d := solver2.GetDomain(nil, v.ID())
		if !d.IsSingleton() || !d.Has(5) {
			t.Errorf("var2[%d] expected {5}, got %s", i, d.String())
		}
	}
}

// TestCount_BoundsPropagation ensures bounds reasoning prunes both the total and booleans.
func TestCount_BoundsPropagation(t *testing.T) {
	model := NewModel()
	// X in {2}, Y in {2,3}, Z in {1,3}
	x := model.NewVariable(NewBitSetDomainFromValues(3, []int{2}))
	y := model.NewVariable(NewBitSetDomainFromValues(3, []int{2, 3}))
	z := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 3}))
	countVar := model.NewVariable(NewBitSetDomain(4)) // [1..4] encodes 0..3

	_, err := NewCount(model, []*FDVariable{x, y, z}, 2, countVar)
	if err != nil {
		t.Fatalf("NewCount failed: %v", err)
	}

	solver := NewSolver(model)
	_, err = solver.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// With X=2, Y∈{2,3}, Z∈{1,3}: count in [1..2] → encoded [2..3]
	cd := solver.GetDomain(nil, countVar.ID())
	if cd.Min() != 2 || cd.Max() != 3 {
		t.Errorf("countVar bounds: got [%d,%d], want [2,3] (encoded)", cd.Min(), cd.Max())
	}

	// If we now force count=2 (encoded 3), Y must be 2 (to reach count 2)
	model2 := NewModel()
	x2 := model2.NewVariable(NewBitSetDomainFromValues(3, []int{2}))
	y2 := model2.NewVariable(NewBitSetDomainFromValues(3, []int{2, 3}))
	z2 := model2.NewVariable(NewBitSetDomainFromValues(3, []int{1, 3}))
	count2 := model2.NewVariable(NewBitSetDomainFromValues(4, []int{3})) // count=2 encoded
	_, err = NewCount(model2, []*FDVariable{x2, y2, z2}, 2, count2)
	if err != nil {
		t.Fatalf("NewCount failed: %v", err)
	}
	solver2 := NewSolver(model2)
	_, err = solver2.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}
	yd := solver2.GetDomain(nil, y2.ID())
	if !yd.IsSingleton() || !yd.Has(2) {
		t.Errorf("Y must be 2 when count=2 and X=2,Z!=2, got %s", yd.String())
	}
}
