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
	if ok != nil {
		t.Fatalf("AddOffsetConstraint failed: %v", ok)
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
	if ok != nil {
		t.Fatalf("AddOffsetConstraint failed: %v", ok)
	}

	// remove some values from b and ensure a is pruned
	if err := s.Remove(b, 9); err != nil {
		t.Fatalf("failed to remove value from b: %v", err)
	}

	// after removing 9 from b, a cannot be 8
	if a.domain.Has(8) {
		t.Fatalf("expected a to no longer allow 8")
	}
}

func TestAddPlusConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddPlusConstraint(x, y, z); err != nil {
		t.Fatalf("AddPlusConstraint failed: %v", err)
	}

	// Test x + y = z with x=2, y=3, should give z=5
	if err := s.Assign(x, 2); err != nil {
		t.Fatalf("failed to assign x=2: %v", err)
	}
	if err := s.Assign(y, 3); err != nil {
		t.Fatalf("failed to assign y=3: %v", err)
	}

	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 5 {
		t.Fatalf("expected z=5, got domain %v", z.domain)
	}
}

func TestAddPlusConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddPlusConstraint(x, y, z); err != nil {
		t.Fatalf("AddPlusConstraint failed: %v", err)
	}

	// Assign z=7, should restrict x and y domains
	if err := s.Assign(z, 7); err != nil {
		t.Fatalf("failed to assign z=7: %v", err)
	}

	// x should be restricted to values where x + y = 7 and y >=1, x<=6
	// So x should be in {1,2,3,4,5,6}
	expectedX := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}
	for v := 1; v <= 9; v++ {
		has := x.domain.Has(v)
		if expectedX[v] && !has {
			t.Fatalf("expected x to have %d", v)
		}
		if !expectedX[v] && has {
			t.Fatalf("unexpected value %d in x domain", v)
		}
	}

	// Similarly for y
	expectedY := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}
	for v := 1; v <= 9; v++ {
		has := y.domain.Has(v)
		if expectedY[v] && !has {
			t.Fatalf("expected y to have %d", v)
		}
		if !expectedY[v] && has {
			t.Fatalf("unexpected value %d in y domain", v)
		}
	}
}

func TestAddPlusConstraintBidirectional(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddPlusConstraint(x, y, z); err != nil {
		t.Fatalf("AddPlusConstraint failed: %v", err)
	}

	// Assign x=4, should restrict z domain
	if err := s.Assign(x, 4); err != nil {
		t.Fatalf("failed to assign x=4: %v", err)
	}

	// z should be in {5,6,7,8,9} (4 + y where y in 1..5, but domain is 1..9)
	// Actually, since y is still 1..9, z could be 4+1=5 through 4+9=13, but clipped to 1..9
	// So z should be {5,6,7,8,9}
	expectedZ := map[int]bool{5: true, 6: true, 7: true, 8: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := z.domain.Has(v)
		if expectedZ[v] && !has {
			t.Fatalf("expected z to have %d", v)
		}
		if !expectedZ[v] && has {
			t.Fatalf("unexpected value %d in z domain", v)
		}
	}
}

func TestAddMultiplyConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddMultiplyConstraint(x, y, z); err != nil {
		t.Fatalf("AddMultiplyConstraint failed: %v", err)
	}

	// Assign x=2, y=3, should restrict z to 6
	if err := s.Assign(x, 2); err != nil {
		t.Fatalf("failed to assign x=2: %v", err)
	}
	if err := s.Assign(y, 3); err != nil {
		t.Fatalf("failed to assign y=3: %v", err)
	}

	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 6 {
		t.Fatalf("expected z to be 6, got domain %v", z.domain)
	}
}

func TestAddMultiplyConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddMultiplyConstraint(x, y, z); err != nil {
		t.Fatalf("AddMultiplyConstraint failed: %v", err)
	}

	// Assign z=6, should restrict x and y domains appropriately
	if err := s.Assign(z, 6); err != nil {
		t.Fatalf("failed to assign z=6: %v", err)
	}

	// x should be factors of 6: 1,2,3,6
	expectedX := map[int]bool{1: true, 2: true, 3: true, 6: true}
	for v := 1; v <= 9; v++ {
		has := x.domain.Has(v)
		if expectedX[v] && !has {
			t.Fatalf("expected x to have %d", v)
		}
		if !expectedX[v] && has {
			t.Fatalf("unexpected value %d in x domain", v)
		}
	}

	// Similarly for y
	expectedY := map[int]bool{1: true, 2: true, 3: true, 6: true}
	for v := 1; v <= 9; v++ {
		has := y.domain.Has(v)
		if expectedY[v] && !has {
			t.Fatalf("expected y to have %d", v)
		}
		if !expectedY[v] && has {
			t.Fatalf("unexpected value %d in y domain", v)
		}
	}
}

func TestAddEqualityConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddEqualityConstraint(x, y, z); err != nil {
		t.Fatalf("AddEqualityConstraint failed: %v", err)
	}

	// Assign x=5, should restrict y and z to 5
	if err := s.Assign(x, 5); err != nil {
		t.Fatalf("failed to assign x=5: %v", err)
	}

	if !y.domain.IsSingleton() || y.domain.SingletonValue() != 5 {
		t.Fatalf("expected y to be 5, got domain %v", y.domain)
	}
	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 5 {
		t.Fatalf("expected z to be 5, got domain %v", z.domain)
	}
}

func TestAddEqualityConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddEqualityConstraint(x, y, z); err != nil {
		t.Fatalf("AddEqualityConstraint failed: %v", err)
	}

	// Restrict x to {3,5}, should restrict y and z to {3,5}
	domain35 := BitSet{n: 9, words: make([]uint64, 1)}
	domain35.words[0] = (1 << 2) | (1 << 4) // values 3 and 5 (1-indexed, so bit 2 and 4)
	if err := s.IntersectDomains(x, domain35); err != nil {
		t.Fatalf("failed to restrict x: %v", err)
	}

	// y and z should also be restricted to {3,5}
	expected := map[int]bool{3: true, 5: true}
	for v := 1; v <= 9; v++ {
		hasY := y.domain.Has(v)
		hasZ := z.domain.Has(v)
		if expected[v] && (!hasY || !hasZ) {
			t.Fatalf("expected y and z to have %d", v)
		}
		if !expected[v] && (hasY || hasZ) {
			t.Fatalf("unexpected value %d in y or z domain", v)
		}
	}
}

func TestAddMinusConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddMinusConstraint(x, y, z); err != nil {
		t.Fatalf("AddMinusConstraint failed: %v", err)
	}

	// Test x - y = z with x=7, y=3, should give z=4
	if err := s.Assign(x, 7); err != nil {
		t.Fatalf("failed to assign x=7: %v", err)
	}
	if err := s.Assign(y, 3); err != nil {
		t.Fatalf("failed to assign y=3: %v", err)
	}

	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 4 {
		t.Fatalf("expected z=4, got domain %v", z.domain)
	}
}

func TestAddMinusConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddMinusConstraint(x, y, z); err != nil {
		t.Fatalf("AddMinusConstraint failed: %v", err)
	}

	// Assign z=4, should restrict x and y domains appropriately
	if err := s.Assign(z, 4); err != nil {
		t.Fatalf("failed to assign z=4: %v", err)
	}

	// x should be in {5,6,7,8,9} (y + 4 where y in 1..5, but clipped appropriately)
	// Actually, x = y + z, so with z=4, x should be {5,6,7,8,9}
	expectedX := map[int]bool{5: true, 6: true, 7: true, 8: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := x.domain.Has(v)
		if expectedX[v] && !has {
			t.Fatalf("expected x to have %d", v)
		}
		if !expectedX[v] && has {
			t.Fatalf("unexpected value %d in x domain", v)
		}
	}

	// y should be in {1,2,3,4,5} (x - 4 where x in 5..9, so y in 1..5)
	expectedY := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	for v := 1; v <= 9; v++ {
		has := y.domain.Has(v)
		if expectedY[v] && !has {
			t.Fatalf("expected y to have %d", v)
		}
		if !expectedY[v] && has {
			t.Fatalf("unexpected value %d in y domain", v)
		}
	}
}

func TestAddMinusConstraintBidirectional(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddMinusConstraint(x, y, z); err != nil {
		t.Fatalf("AddMinusConstraint failed: %v", err)
	}

	// Assign x=7, should restrict z domain
	if err := s.Assign(x, 7); err != nil {
		t.Fatalf("failed to assign x=7: %v", err)
	}

	// z should be in {1,2,3,4,5,6} (7 - y where y in 1..6, but domain is 1..9)
	// Actually, z = x - y, so with x=7, z should be {1,2,3,4,5,6}
	expectedZ := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}
	for v := 1; v <= 9; v++ {
		has := z.domain.Has(v)
		if expectedZ[v] && !has {
			t.Fatalf("expected z to have %d", v)
		}
		if !expectedZ[v] && has {
			t.Fatalf("unexpected value %d in z domain", v)
		}
	}
}

func TestAddQuotientConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddQuotientConstraint(x, y, z); err != nil {
		t.Fatalf("AddQuotientConstraint failed: %v", err)
	}

	// Test x / y = z with x=8, y=2, should give z=4
	if err := s.Assign(x, 8); err != nil {
		t.Fatalf("failed to assign x=8: %v", err)
	}
	if err := s.Assign(y, 2); err != nil {
		t.Fatalf("failed to assign y=2: %v", err)
	}

	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 4 {
		t.Fatalf("expected z=4, got domain %v", z.domain)
	}
}

func TestAddQuotientConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddQuotientConstraint(x, y, z); err != nil {
		t.Fatalf("AddQuotientConstraint failed: %v", err)
	}

	// Assign z=3, should restrict x and y domains appropriately
	if err := s.Assign(z, 3); err != nil {
		t.Fatalf("failed to assign z=3: %v", err)
	}

	// x should be in {3,6,9} (y * 3 where y in 1..3, but domain is 1..9)
	// Actually, x = y * z, so with z=3, x should be {3,6,9}
	expectedX := map[int]bool{3: true, 6: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := x.domain.Has(v)
		if expectedX[v] && !has {
			t.Fatalf("expected x to have %d", v)
		}
		if !expectedX[v] && has {
			t.Fatalf("unexpected value %d in x domain", v)
		}
	}

	// y should be in {1,2,3} (x / 3 where x in 3..9, so y in 1..3)
	expectedY := map[int]bool{1: true, 2: true, 3: true}
	for v := 1; v <= 9; v++ {
		has := y.domain.Has(v)
		if expectedY[v] && !has {
			t.Fatalf("expected y to have %d", v)
		}
		if !expectedY[v] && has {
			t.Fatalf("unexpected value %d in y domain", v)
		}
	}
}

func TestAddQuotientConstraintBidirectional(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddQuotientConstraint(x, y, z); err != nil {
		t.Fatalf("AddQuotientConstraint failed: %v", err)
	}

	// Assign x=9, should restrict z domain
	if err := s.Assign(x, 9); err != nil {
		t.Fatalf("failed to assign x=9: %v", err)
	}

	// z should be in {1,3,9} (9 / y where y divides 9, so y in {1,3,9}, z in {9,3,1})
	expectedZ := map[int]bool{1: true, 3: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := z.domain.Has(v)
		if expectedZ[v] && !has {
			t.Fatalf("expected z to have %d", v)
		}
		if !expectedZ[v] && has {
			t.Fatalf("unexpected value %d in z domain", v)
		}
	}
}

func TestAddModuloConstraintBasic(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddModuloConstraint(x, y, z); err != nil {
		t.Fatalf("AddModuloConstraint failed: %v", err)
	}

	// Test x % y = z with x=7, y=3, should give z=1 (7 % 3 = 1)
	if err := s.Assign(x, 7); err != nil {
		t.Fatalf("failed to assign x=7: %v", err)
	}
	if err := s.Assign(y, 3); err != nil {
		t.Fatalf("failed to assign y=3: %v", err)
	}

	if !z.domain.IsSingleton() || z.domain.SingletonValue() != 1 {
		t.Fatalf("expected z=1, got domain %v", z.domain)
	}
}

func TestAddModuloConstraintPropagation(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddModuloConstraint(x, y, z); err != nil {
		t.Fatalf("AddModuloConstraint failed: %v", err)
	}

	// Assign z=2, should restrict x and y domains appropriately
	if err := s.Assign(z, 2); err != nil {
		t.Fatalf("failed to assign z=2: %v", err)
	}

	// x should be in values where x % y = 2 for some y > 2
	// For y=3: x ≡ 2 mod 3 -> {2,5,8}
	// For y=4: x ≡ 2 mod 4 -> {2,6}
	// For y=5: x ≡ 2 mod 5 -> {2,7}
	// For y=6: x ≡ 2 mod 6 -> {2,8}
	// For y=7: x ≡ 2 mod 7 -> {2,9}
	// For y=8: x ≡ 2 mod 8 -> {2}
	// For y=9: x ≡ 2 mod 9 -> {2}
	// Union: {2,5,6,7,8,9}
	expectedX := map[int]bool{2: true, 5: true, 6: true, 7: true, 8: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := x.domain.Has(v)
		if expectedX[v] && !has {
			t.Fatalf("expected x to have %d", v)
		}
		if !expectedX[v] && has {
			t.Fatalf("unexpected value %d in x domain", v)
		}
	}

	// y should be in {3,4,5,6,7,8,9} (y > z = 2)
	expectedY := map[int]bool{3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true}
	for v := 1; v <= 9; v++ {
		has := y.domain.Has(v)
		if expectedY[v] && !has {
			t.Fatalf("expected y to have %d", v)
		}
		if !expectedY[v] && has {
			t.Fatalf("unexpected value %d in y domain", v)
		}
	}
}

func TestAddModuloConstraintBidirectional(t *testing.T) {
	s := NewFDStoreWithDomain(9)
	x := s.NewVar()
	y := s.NewVar()
	z := s.NewVar()

	if err := s.AddModuloConstraint(x, y, z); err != nil {
		t.Fatalf("AddModuloConstraint failed: %v", err)
	}

	// Assign x=8, should restrict z domain
	if err := s.Assign(x, 8); err != nil {
		t.Fatalf("failed to assign x=8: %v", err)
	}

	// z should be in possible values of 8 % y for y in 1..9
	// y=1: 8%1=0 (invalid)
	// y=2: 8%2=0 (invalid)
	// y=3: 8%3=2
	// y=4: 8%4=0 (invalid)
	// y=5: 8%5=3
	// y=6: 8%6=2
	// y=7: 8%7=1
	// y=8: 8%8=0 (invalid)
	// y=9: 8%9=8
	// So z can be {1,2,3,8}
	expectedZ := map[int]bool{1: true, 2: true, 3: true, 8: true}
	for v := 1; v <= 9; v++ {
		has := z.domain.Has(v)
		if expectedZ[v] && !has {
			t.Fatalf("expected z to have %d", v)
		}
		if !expectedZ[v] && has {
			t.Fatalf("unexpected value %d in z domain", v)
		}
	}
}
