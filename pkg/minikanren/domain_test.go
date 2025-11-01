package minikanren

import (
	"testing"
)

func TestNewBitSetDomain(t *testing.T) {
	tests := []struct {
		name     string
		maxValue int
		wantSize int
	}{
		{"small domain", 5, 5},
		{"sudoku domain", 9, 9},
		{"large domain", 100, 100},
		{"single value", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := NewBitSetDomain(tt.maxValue)
			if domain.Count() != tt.wantSize {
				t.Errorf("Count() = %d, want %d", domain.Count(), tt.wantSize)
			}
			if domain.MaxValue() != tt.maxValue {
				t.Errorf("MaxValue() = %d, want %d", domain.MaxValue(), tt.maxValue)
			}
			// Verify all values are present
			for i := 1; i <= tt.maxValue; i++ {
				if !domain.Has(i) {
					t.Errorf("domain should contain %d", i)
				}
			}
			// Verify out-of-range values are not present
			if domain.Has(0) {
				t.Error("domain should not contain 0")
			}
			if domain.Has(tt.maxValue + 1) {
				t.Errorf("domain should not contain %d", tt.maxValue+1)
			}
		})
	}
}

func TestNewBitSetDomainFromValues(t *testing.T) {
	tests := []struct {
		name     string
		maxValue int
		values   []int
		want     []int
	}{
		{"even digits", 9, []int{2, 4, 6, 8}, []int{2, 4, 6, 8}},
		{"sparse values", 20, []int{1, 5, 10, 15, 20}, []int{1, 5, 10, 15, 20}},
		{"single value", 10, []int{7}, []int{7}},
		{"empty values", 10, []int{}, []int{}},
		{"values with duplicates", 5, []int{1, 2, 2, 3, 3, 3}, []int{1, 2, 3}},
		{"values outside range", 5, []int{-1, 0, 3, 6, 10}, []int{3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := NewBitSetDomainFromValues(tt.maxValue, tt.values)
			if domain.Count() != len(tt.want) {
				t.Errorf("Count() = %d, want %d", domain.Count(), len(tt.want))
			}
			for _, v := range tt.want {
				if !domain.Has(v) {
					t.Errorf("domain should contain %d", v)
				}
			}
		})
	}
}

func TestBitSetDomain_Has(t *testing.T) {
	domain := NewBitSetDomainFromValues(10, []int{2, 5, 7})

	tests := []struct {
		value int
		want  bool
	}{
		{1, false},
		{2, true},
		{3, false},
		{5, true},
		{7, true},
		{10, false},
		{0, false},
		{-1, false},
		{11, false},
	}

	for _, tt := range tests {
		if got := domain.Has(tt.value); got != tt.want {
			t.Errorf("Has(%d) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestBitSetDomain_Remove(t *testing.T) {
	original := NewBitSetDomain(5)

	// Remove value that exists
	d1 := original.Remove(3)
	if d1.Has(3) {
		t.Error("domain should not contain 3 after removal")
	}
	if d1.Count() != 4 {
		t.Errorf("Count() = %d, want 4", d1.Count())
	}

	// Original should be unchanged (immutable)
	if !original.Has(3) {
		t.Error("original domain should still contain 3")
	}
	if original.Count() != 5 {
		t.Errorf("original Count() = %d, want 5", original.Count())
	}

	// Remove value that doesn't exist
	d2 := d1.Remove(3)
	if d2.Count() != d1.Count() {
		t.Error("removing non-existent value should not change count")
	}

	// Remove all values
	domain := NewBitSetDomain(3)
	domain = domain.Remove(1).(*BitSetDomain)
	domain = domain.Remove(2).(*BitSetDomain)
	domain = domain.Remove(3).(*BitSetDomain)
	if domain.Count() != 0 {
		t.Errorf("Count() = %d, want 0", domain.Count())
	}
}

func TestBitSetDomain_IsSingleton(t *testing.T) {
	tests := []struct {
		name   string
		domain Domain
		want   bool
	}{
		{"empty domain", NewBitSetDomainFromValues(10, []int{}), false},
		{"singleton domain", NewBitSetDomainFromValues(10, []int{5}), true},
		{"two values", NewBitSetDomainFromValues(10, []int{3, 7}), false},
		{"full domain", NewBitSetDomain(10), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.IsSingleton(); got != tt.want {
				t.Errorf("IsSingleton() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBitSetDomain_SingletonValue(t *testing.T) {
	tests := []struct {
		value int
	}{
		{1},
		{5},
		{9},
		{100},
	}

	for _, tt := range tests {
		domain := NewBitSetDomainFromValues(100, []int{tt.value})
		if !domain.IsSingleton() {
			t.Fatal("domain should be singleton")
		}
		if got := domain.SingletonValue(); got != tt.value {
			t.Errorf("SingletonValue() = %d, want %d", got, tt.value)
		}
	}
}

func TestBitSetDomain_IterateValues(t *testing.T) {
	domain := NewBitSetDomainFromValues(10, []int{2, 5, 7, 9})

	var values []int
	domain.IterateValues(func(v int) {
		values = append(values, v)
	})

	want := []int{2, 5, 7, 9}
	if len(values) != len(want) {
		t.Fatalf("IterateValues got %v, want %v", values, want)
	}
	for i, v := range values {
		if v != want[i] {
			t.Errorf("values[%d] = %d, want %d", i, v, want[i])
		}
	}

	// Test that values are in ascending order
	prev := 0
	domain.IterateValues(func(v int) {
		if v <= prev {
			t.Errorf("values not in ascending order: %d after %d", v, prev)
		}
		prev = v
	})
}

func TestBitSetDomain_Intersect(t *testing.T) {
	tests := []struct {
		name    string
		domain1 []int
		domain2 []int
		want    []int
	}{
		{
			"overlapping sets",
			[]int{1, 2, 3, 4, 5},
			[]int{3, 4, 5, 6, 7},
			[]int{3, 4, 5},
		},
		{
			"disjoint sets",
			[]int{1, 2, 3},
			[]int{4, 5, 6},
			[]int{},
		},
		{
			"identical sets",
			[]int{1, 3, 5},
			[]int{1, 3, 5},
			[]int{1, 3, 5},
		},
		{
			"subset",
			[]int{1, 2, 3, 4, 5},
			[]int{2, 4},
			[]int{2, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1 := NewBitSetDomainFromValues(10, tt.domain1)
			d2 := NewBitSetDomainFromValues(10, tt.domain2)
			result := d1.Intersect(d2)

			if result.Count() != len(tt.want) {
				t.Errorf("Intersect() count = %d, want %d", result.Count(), len(tt.want))
			}

			for _, v := range tt.want {
				if !result.Has(v) {
					t.Errorf("result should contain %d", v)
				}
			}
		})
	}
}

func TestBitSetDomain_Union(t *testing.T) {
	tests := []struct {
		name    string
		domain1 []int
		domain2 []int
		want    []int
	}{
		{
			"overlapping sets",
			[]int{1, 2, 3},
			[]int{3, 4, 5},
			[]int{1, 2, 3, 4, 5},
		},
		{
			"disjoint sets",
			[]int{1, 3, 5},
			[]int{2, 4, 6},
			[]int{1, 2, 3, 4, 5, 6},
		},
		{
			"identical sets",
			[]int{2, 4, 6},
			[]int{2, 4, 6},
			[]int{2, 4, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1 := NewBitSetDomainFromValues(10, tt.domain1)
			d2 := NewBitSetDomainFromValues(10, tt.domain2)
			result := d1.Union(d2)

			if result.Count() != len(tt.want) {
				t.Errorf("Union() count = %d, want %d", result.Count(), len(tt.want))
			}

			for _, v := range tt.want {
				if !result.Has(v) {
					t.Errorf("result should contain %d", v)
				}
			}
		})
	}
}

func TestBitSetDomain_Complement(t *testing.T) {
	tests := []struct {
		name     string
		maxValue int
		values   []int
		want     []int
	}{
		{
			"even digits",
			9,
			[]int{2, 4, 6, 8},
			[]int{1, 3, 5, 7, 9},
		},
		{
			"singleton",
			5,
			[]int{3},
			[]int{1, 2, 4, 5},
		},
		{
			"empty",
			3,
			[]int{},
			[]int{1, 2, 3},
		},
		{
			"full",
			3,
			[]int{1, 2, 3},
			[]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain := NewBitSetDomainFromValues(tt.maxValue, tt.values)
			complement := domain.Complement()

			if complement.Count() != len(tt.want) {
				t.Errorf("Complement() count = %d, want %d", complement.Count(), len(tt.want))
			}

			for _, v := range tt.want {
				if !complement.Has(v) {
					t.Errorf("complement should contain %d", v)
				}
			}

			// Double complement should give original
			doubleComp := complement.Complement()
			if !domain.Equal(doubleComp) {
				t.Error("double complement should equal original")
			}
		})
	}
}

func TestBitSetDomain_Clone(t *testing.T) {
	original := NewBitSetDomainFromValues(10, []int{2, 5, 7})
	clone := original.Clone()

	// Should be equal
	if !original.Equal(clone) {
		t.Error("clone should equal original")
	}

	// Modifying clone should not affect original
	modified := clone.Remove(5)
	if !original.Has(5) {
		t.Error("original should still have 5")
	}
	if modified.Has(5) {
		t.Error("modified clone should not have 5")
	}
}

func TestBitSetDomain_Equal(t *testing.T) {
	d1 := NewBitSetDomainFromValues(10, []int{2, 4, 6})
	d2 := NewBitSetDomainFromValues(10, []int{2, 4, 6})
	d3 := NewBitSetDomainFromValues(10, []int{2, 4, 8})
	d4 := NewBitSetDomainFromValues(5, []int{2, 4})

	if !d1.Equal(d2) {
		t.Error("identical domains should be equal")
	}
	if d1.Equal(d3) {
		t.Error("different domains should not be equal")
	}
	if d1.Equal(d4) {
		t.Error("domains with different maxValue should not be equal")
	}
}

func TestBitSetDomain_String(t *testing.T) {
	tests := []struct {
		name   string
		domain Domain
		want   string
	}{
		{"empty", NewBitSetDomainFromValues(10, []int{}), "{}"},
		{"singleton", NewBitSetDomainFromValues(10, []int{5}), "{5}"},
		{"range", NewBitSetDomain(5), "{1..5}"},
		{"sparse", NewBitSetDomainFromValues(10, []int{2, 4, 6, 8}), "{2,4,6,8}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.domain.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Benchmark domain operations
func BenchmarkBitSetDomain_Has(b *testing.B) {
	domain := NewBitSetDomain(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.Has(50)
	}
}

func BenchmarkBitSetDomain_Remove(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		domain := NewBitSetDomain(100)
		b.StartTimer()
		domain.Remove(50)
	}
}

func BenchmarkBitSetDomain_Intersect(b *testing.B) {
	d1 := NewBitSetDomain(100)
	d2 := NewBitSetDomainFromValues(100, []int{25, 50, 75})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1.Intersect(d2)
	}
}

func BenchmarkBitSetDomain_IterateValues(b *testing.B) {
	domain := NewBitSetDomain(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain.IterateValues(func(v int) {})
	}
}

// Additional edge case tests for >90% coverage

func TestBitSetDomain_EdgeCases(t *testing.T) {
	t.Run("NewBitSetDomain with zero", func(t *testing.T) {
		d := NewBitSetDomain(0)
		if d.Count() != 0 {
			t.Errorf("NewBitSetDomain(0).Count() = %d, want 0", d.Count())
		}
	})

	t.Run("NewBitSetDomain with negative", func(t *testing.T) {
		d := NewBitSetDomain(-5)
		if d.Count() != 0 {
			t.Errorf("NewBitSetDomain(-5).Count() = %d, want 0", d.Count())
		}
	})

	t.Run("SingletonValue panic on empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("SingletonValue on empty domain should panic")
			}
		}()
		d := NewBitSetDomainFromValues(10, []int{})
		d.SingletonValue()
	})

	t.Run("Intersect with different maxValue", func(t *testing.T) {
		d1 := NewBitSetDomain(10)
		d2 := NewBitSetDomain(20)
		result := d1.Intersect(d2)
		// Different maxValue means incompatible, should return empty
		if result.Count() != 0 {
			t.Errorf("Intersect with different maxValue should return empty domain, got count %d", result.Count())
		}
	})

	t.Run("Union with different sized domains", func(t *testing.T) {
		d1 := NewBitSetDomainFromValues(100, []int{1, 2, 3})
		d2 := NewBitSetDomainFromValues(200, []int{98, 99, 100})
		result := d1.Union(d2)
		if result.Count() != 6 {
			t.Errorf("Union count = %d, want 6", result.Count())
		}
	})

	t.Run("ToSlice", func(t *testing.T) {
		d := NewBitSetDomainFromValues(10, []int{2, 5, 8})
		values := d.ToSlice()
		expected := []int{2, 5, 8}
		if len(values) != len(expected) {
			t.Fatalf("ToSlice() length = %d, want %d", len(values), len(expected))
		}
		for i, v := range expected {
			if values[i] != v {
				t.Errorf("ToSlice()[%d] = %d, want %d", i, values[i], v)
			}
		}
	})

	t.Run("Equal with different types", func(t *testing.T) {
		// Since we only have BitSetDomain, test with nil
		d1 := NewBitSetDomain(10)
		if d1.Equal(nil) {
			t.Error("Equal with nil should return false")
		}
	})

	t.Run("Equal with different maxValue", func(t *testing.T) {
		d1 := NewBitSetDomain(10)
		d2 := NewBitSetDomain(20)
		if d1.Equal(d2) {
			t.Error("Equal with different maxValue should return false")
		}
	})
}
