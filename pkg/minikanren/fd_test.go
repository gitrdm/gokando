package minikanren

import "testing"

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
	if !s.Assign(a, 1) {
		t.Fatalf("assign failed")
	}
	// propagate should remove 1 from peers
	if b.domain.Has(1) || c.domain.Has(1) {
		t.Fatalf("peer domains not pruned")
	}

	// backtrack via undo
	snap := s.snapshot()
	if !s.Assign(b, 2) {
		t.Fatalf("assign b failed")
	}
	s.undo(snap)
	if b.domain.IsSingleton() {
		t.Fatalf("undo failed")
	}
}
