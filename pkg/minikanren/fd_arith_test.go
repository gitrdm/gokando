package minikanren

import "testing"

func TestAddOffsetConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	a := s.NewVar()
	b := s.NewVar()

	// set a domain to {1,2,3}
	a.domain = BitSet{n: 9, words: make([]uint64, len(a.domain.words))}
	for _, v := range []int{1, 2, 3} {
		idx := (v - 1) / 64
		off := uint((v - 1) % 64)
		a.domain.words[idx] |= 1 << off
	}

	ok := s.AddOffsetConstraint(a, 2, b) // b = a + 2
	if !ok {
		t.Fatalf("AddOffsetConstraint failed")
	}

	// expect b domain to be {3,4,5}
	expected := map[int]bool{3: true, 4: true, 5: true}
	for v := 1; v <= 9; v++ {
		has := b.domain.Has(v)
		if expected[v] && !has {
			t.Fatalf("expected b to have %d", v)
		}
		if !expected[v] && has {
			t.Fatalf("unexpected value %d in b domain", v)
		}
	}
}

func TestOffsetPropagationBidirectional(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	a := s.NewVar()
	b := s.NewVar()

	// both start full
	ok := s.AddOffsetConstraint(a, 1, b) // b = a + 1
	if !ok {
		t.Fatalf("AddOffsetConstraint failed")
	}

	// remove some values from b and ensure a is pruned
	if !s.Remove(b, 9) {
		t.Fatalf("failed to remove value from b")
	}

	// after removing 9 from b, a cannot be 8
	if a.domain.Has(8) {
		t.Fatalf("expected a to no longer allow 8")
	}
}
