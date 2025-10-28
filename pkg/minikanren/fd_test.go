package minikanren

import (
	"context"
	"fmt"
	"testing"
)

func TestDomainBasics(t *testing.T) {
	d := NewBitSet(9)
	if d.Count() != 9 {
		t.Fatalf("expected 9, got %d", d.Count())
	}
	if !d.Has(5) {
		t.Fatalf("expected domain to have 5")
	}
	d2 := d.RemoveValue(5)
	if d2.Has(5) {
		t.Fatalf("expected 5 removed")
	}
}

func TestDomainOperations(t *testing.T) {
	d1 := NewBitSet(5)     // {1,2,3,4,5}
	d2 := NewBitSet(5)     // {1,2,3,4,5}
	d2 = d2.RemoveValue(3) // {1,2,4,5}

	// Test intersection
	intersect := d1.Intersect(d2)
	if intersect.Count() != 4 || intersect.Has(3) {
		t.Error("Intersection should be {1,2,4,5}")
	}

	// Test union
	union := d1.Union(d2)
	if union.Count() != 5 {
		t.Error("Union should have 5 elements")
	}

	// Test complement
	comp := d1.Complement()
	if comp.Count() != 0 {
		t.Error("Complement of full domain should be empty")
	}

	// Test FDStore domain operations
	store := NewFDStoreWithDomain(5)
	v := store.NewVar()

	// Intersect domain
	subset := NewBitSet(5)
	subset = subset.RemoveValue(4)
	subset = subset.RemoveValue(5)

	err := store.IntersectDomains(v, subset)
	if err != nil {
		t.Fatalf("IntersectDomains failed: %v", err)
	}

	if store.GetDomain(v).Count() != 3 {
		t.Error("Domain should have 3 values after intersection")
	}

	// Union domain
	err = store.UnionDomains(v, NewBitSet(5))
	if err != nil {
		t.Fatalf("UnionDomains failed: %v", err)
	}

	if store.GetDomain(v).Count() != 5 {
		t.Error("Domain should have 5 values after union")
	}
}

func TestInequalityConstraints(t *testing.T) {
	store := NewFDStoreWithDomain(5)
	x := store.NewVar()
	y := store.NewVar()

	// Test X < Y
	err := store.AddInequalityConstraint(x, y, IneqLessThan)
	if err != nil {
		t.Fatalf("AddInequalityConstraint failed: %v", err)
	}

	// Assign x=3, should constrain y to 4,5
	err = store.Assign(x, 3)
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}

	if store.GetDomain(y).Has(1) || store.GetDomain(y).Has(2) || store.GetDomain(y).Has(3) {
		t.Error("Y should not have values <= 3")
	}

	// Test X != Y
	store2 := NewFDStoreWithDomain(3)
	a := store2.NewVar()
	b := store2.NewVar()

	err = store2.AddInequalityConstraint(a, b, IneqNotEqual)
	if err != nil {
		t.Fatalf("AddInequalityConstraint failed: %v", err)
	}

	err = store2.Assign(a, 2)
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}

	if store2.GetDomain(b).Has(2) {
		t.Error("B should not have value 2")
	}
}

func TestCustomConstraints(t *testing.T) {
	// Test SumConstraint
	sumConstraint := NewSumConstraint([]*FDVar{}, 10) // Empty for now

	store := NewFDStoreWithDomain(5)
	vars := store.MakeFDVars(3)

	sumConstraint = NewSumConstraint(vars, 6)
	err := store.AddCustomConstraint(sumConstraint)
	if err != nil {
		t.Fatalf("AddCustomConstraint failed: %v", err)
	}

	// Assign first two variables
	err = store.Assign(vars[0], 1)
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}
	err = store.Assign(vars[1], 2)
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}

	// Third variable should be constrained to 3
	if !store.GetDomain(vars[2]).Has(3) || store.GetDomain(vars[2]).Count() != 1 {
		t.Error("Third variable should be constrained to 3")
	}

	// Test AllDifferentConstraint
	store2 := NewFDStoreWithDomain(3)
	vars2 := store2.MakeFDVars(3)

	allDiff := NewAllDifferentConstraint(vars2)
	err = store2.AddCustomConstraint(allDiff)
	if err != nil {
		t.Fatalf("AddCustomConstraint failed: %v", err)
	}

	err = store2.Assign(vars2[0], 1)
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}

	if store2.GetDomain(vars2[1]).Has(1) || store2.GetDomain(vars2[2]).Has(1) {
		t.Error("Other variables should not have value 1")
	}
}

func TestAdvancedHeuristics(t *testing.T) {
	store := NewFDStoreWithDomain(4)

	// Create config with different heuristics
	config := &SolverConfig{
		VariableHeuristic: HeuristicDom,
		ValueHeuristic:    ValueOrderDesc,
		RandomSeed:        123,
	}

	store = NewFDStoreWithConfig(4, config)
	vars := store.MakeFDVars(3)

	// Add some constraints
	store.AddAllDifferent(vars)

	// Solve with custom config
	solutions, err := store.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Should find at least one solution")
	}

	// Test different heuristic
	config.VariableHeuristic = HeuristicRandom
	store2 := NewFDStoreWithConfig(4, config)
	vars2 := store2.MakeFDVars(3)
	store2.AddAllDifferent(vars2)

	solutions2, err := store2.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if len(solutions2) == 0 {
		t.Error("Should find at least one solution with random heuristic")
	}
}

func TestMonitoring(t *testing.T) {
	store := NewFDStoreWithDomain(4)
	monitor := NewSolverMonitor()
	store.SetMonitor(monitor)

	vars := store.MakeFDVars(3)
	store.AddAllDifferent(vars)

	// Solve and check stats
	_, err := store.Solve(context.Background(), 1)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	stats := store.GetStats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats.NodesExplored == 0 {
		t.Error("Should have explored some nodes")
	}

	if stats.ConstraintsAdded == 0 {
		t.Error("Should have recorded constraint additions")
	}

	// Check string representation
	statsStr := stats.String()
	if len(statsStr) == 0 {
		t.Error("Stats string should not be empty")
	}
}

func TestFDStoreSimple(t *testing.T) {
	s := NewFDStore()
	a := s.NewVar()
	b := s.NewVar()
	c := s.NewVar()
	s.AddAllDifferent([]*FDVar{a, b, c})

	// assign a=1
	if err := s.Assign(a, 1); err != nil {
		t.Fatalf("assign failed: %v", err)
	}
	// propagate should remove 1 from peers
	if b.domain.Has(1) || c.domain.Has(1) {
		t.Fatalf("peer domains not pruned")
	}

	// backtrack via undo
	snap := s.snapshot()
	if err := s.Assign(b, 2); err != nil {
		t.Fatalf("assign b failed: %v", err)
	}
	s.undo(snap)
	if b.domain.IsSingleton() {
		t.Fatalf("undo failed")
	}
}

// BenchmarkFDSolveNQueens benchmarks FD solving for N-Queens
func BenchmarkFDSolveNQueens(b *testing.B) {
	for _, n := range []int{4, 8, 10} {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				store := NewFDStoreWithDomain(2 * n)
				cols := store.MakeFDVars(n)
				d1 := store.MakeFDVars(n)
				d2 := store.MakeFDVars(n)
				for i := 0; i < n; i++ {
					if err := store.AddOffsetLink(cols[i], i, d1[i]); err != nil {
						b.Fatal(err)
					}
					if err := store.AddOffsetLink(cols[i], -i+n, d2[i]); err != nil {
						b.Fatal(err)
					}
				}
				if err := store.ApplyAllDifferentRegin(cols); err != nil {
					b.Fatal(err)
				}
				if err := store.ApplyAllDifferentRegin(d1); err != nil {
					b.Fatal(err)
				}
				if err := store.ApplyAllDifferentRegin(d2); err != nil {
					b.Fatal(err)
				}
				// constrain columns to 1..n
				for _, v := range cols {
					for j := n + 1; j <= 2*n; j++ {
						if err := store.Remove(v, j); err != nil {
							b.Fatal(err)
						}
					}
				}
				_, _ = store.Solve(context.Background(), 1)
			}
		})
	}
}
