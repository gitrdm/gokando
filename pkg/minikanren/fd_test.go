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
