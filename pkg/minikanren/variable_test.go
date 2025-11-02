package minikanren

import (
	"testing"
)

func TestNewFDVariable(t *testing.T) {
	domain := NewBitSetDomain(10)
	v := NewFDVariable(0, domain)

	if v == nil {
		t.Fatal("NewFDVariable() returned nil")
	}
	if v.ID() != 0 {
		t.Errorf("ID() = %d, want 0", v.ID())
	}
	if v.Domain() != domain {
		t.Error("Domain() should return same domain instance")
	}
}

func TestNewFDVariableWithName(t *testing.T) {
	domain := NewBitSetDomain(10)
	v := NewFDVariableWithName(0, domain, "x")

	if v == nil {
		t.Fatal("NewFDVariableWithName() returned nil")
	}
	if v.ID() != 0 {
		t.Errorf("ID() = %d, want 0", v.ID())
	}
	if v.String() == "" || !contains(v.String(), "x") {
		t.Errorf("String() should contain name 'x', got %q", v.String())
	}
}

func TestFDVariable_ID(t *testing.T) {
	tests := []struct {
		name   string
		id     int
		wantID int
	}{
		{"zero id", 0, 0},
		{"positive id", 42, 42},
		{"large id", 999999, 999999},
	}

	domain := NewBitSetDomain(5)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewFDVariable(tt.id, domain)
			if got := v.ID(); got != tt.wantID {
				t.Errorf("ID() = %d, want %d", got, tt.wantID)
			}
		})
	}
}

func TestFDVariable_Domain(t *testing.T) {
	tests := []struct {
		name   string
		domain Domain
	}{
		{
			"small domain",
			NewBitSetDomain(5),
		},
		{
			"large domain",
			NewBitSetDomain(1000),
		},
		{
			"sparse domain",
			NewBitSetDomainFromValues(100, []int{1, 50, 99}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewFDVariable(0, tt.domain)
			got := v.Domain()

			if got == nil {
				t.Fatal("Domain() returned nil")
			}
			if got != tt.domain {
				t.Error("Domain() should return same instance")
			}
			if got.Count() != tt.domain.Count() {
				t.Errorf("Domain().Count() = %d, want %d", got.Count(), tt.domain.Count())
			}
		})
	}
}

func TestFDVariable_String_WithName(t *testing.T) {
	tests := []struct {
		name    string
		varName string
	}{
		{"single char", "x"},
		{"multi char", "variable"},
		{"with numbers", "x1"},
		{"with underscores", "my_var"},
	}

	domain := NewBitSetDomain(5)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewFDVariableWithName(0, domain, tt.varName)
			str := v.String()
			if !contains(str, tt.varName) {
				t.Errorf("String() = %q, should contain %q", str, tt.varName)
			}
		})
	}
}

func TestFDVariable_IsBound(t *testing.T) {
	tests := []struct {
		name      string
		domain    Domain
		wantBound bool
	}{
		{
			"unbound - multiple values",
			NewBitSetDomain(10),
			false,
		},
		{
			"bound - single value",
			NewBitSetDomainFromValues(10, []int{5}),
			true,
		},
		{
			"unbound - two values",
			NewBitSetDomainFromValues(10, []int{3, 7}),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewFDVariable(0, tt.domain)
			if got := v.IsBound(); got != tt.wantBound {
				t.Errorf("IsBound() = %v, want %v", got, tt.wantBound)
			}
		})
	}
}

func TestFDVariable_Value(t *testing.T) {
	tests := []struct {
		name        string
		domain      Domain
		wantValue   int
		shouldPanic bool
	}{
		{
			"unbound variable panics",
			NewBitSetDomain(10),
			0,
			true,
		},
		{
			"bound to single value",
			NewBitSetDomainFromValues(10, []int{5}),
			5,
			false,
		},
		{
			"multiple values panics",
			NewBitSetDomainFromValues(10, []int{3, 7}),
			0,
			true,
		},
		{
			"bound to 1",
			NewBitSetDomainFromValues(10, []int{1}),
			1,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewFDVariable(0, tt.domain)

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Value() should panic for unbound variable")
					}
				}()
				v.Value()
			} else {
				gotValue := v.Value()
				if gotValue != tt.wantValue {
					t.Errorf("Value() = %d, want %d", gotValue, tt.wantValue)
				}
			}
		})
	}
}

func TestFDVariable_String(t *testing.T) {
	tests := []struct {
		name         string
		id           int
		domain       Domain
		varName      string
		wantContains []string
	}{
		{
			"unbound with name",
			0,
			NewBitSetDomain(5),
			"x",
			[]string{"x", "{1..5}"},
		},
		{
			"bound with name",
			1,
			NewBitSetDomainFromValues(10, []int{5}),
			"y",
			[]string{"y", "5"},
		},
		{
			"without name",
			2,
			NewBitSetDomain(3),
			"",
			[]string{"v2", "{1..3}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v *FDVariable
			if tt.varName == "" {
				v = NewFDVariable(tt.id, tt.domain)
			} else {
				v = NewFDVariableWithName(tt.id, tt.domain, tt.varName)
			}
			str := v.String()

			for _, want := range tt.wantContains {
				if !contains(str, want) {
					t.Errorf("String() = %q, should contain %q", str, want)
				}
			}
		})
	}
}

func TestFDVariable_Immutability(t *testing.T) {
	// FDVariable should be safe to share across goroutines
	// This test verifies that the domain reference is immutable

	domain := NewBitSetDomain(10)
	v := NewFDVariable(0, domain)

	// Getting domain multiple times should return same reference
	d1 := v.Domain()
	d2 := v.Domain()

	if d1 != d2 {
		t.Error("Domain() should return consistent reference")
	}

	// Modifying the returned domain should not affect the variable
	// (domains are immutable, so this creates a new domain)
	removed := d1.Remove(5)

	if removed == v.Domain() {
		t.Error("domain operations should not mutate original")
	}
	if v.Domain().Count() != domain.Count() {
		t.Error("variable domain should be unchanged")
	}
}

// Benchmark variable operations
func BenchmarkFDVariable_ID(b *testing.B) {
	v := NewFDVariableWithName(42, NewBitSetDomain(100), "x")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ID()
	}
}

func BenchmarkFDVariable_Domain(b *testing.B) {
	v := NewFDVariable(0, NewBitSetDomain(100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Domain()
	}
}

func BenchmarkFDVariable_IsBound(b *testing.B) {
	v := NewFDVariable(0, NewBitSetDomain(100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.IsBound()
	}
}

func BenchmarkFDVariable_Value(b *testing.B) {
	v := NewFDVariable(0, NewBitSetDomainFromValues(100, []int{50}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Value()
	}
}

func BenchmarkFDVariable_String(b *testing.B) {
	v := NewFDVariableWithName(0, NewBitSetDomain(10), "x")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.String()
	}
}

func TestFDVariable_TryValue(t *testing.T) {
	// Unbound -> error
	v1 := NewFDVariable(0, NewBitSetDomain(10))
	if _, err := v1.TryValue(); err == nil {
		t.Errorf("TryValue() should return error for unbound variable")
	}

	// Bound -> value, nil
	v2 := NewFDVariable(0, NewBitSetDomainFromValues(10, []int{7}))
	val, err := v2.TryValue()
	if err != nil {
		t.Fatalf("TryValue() unexpected error: %v", err)
	}
	if val != 7 {
		t.Errorf("TryValue() = %d, want 7", val)
	}
}

// Edge case tests for >90% coverage

func TestFDVariable_SetDomain(t *testing.T) {
	v := NewFDVariable(0, NewBitSetDomain(10))
	originalDomain := v.Domain()

	// SetDomain should update the domain
	newDomain := NewBitSetDomainFromValues(10, []int{5})
	v.SetDomain(newDomain)

	if v.Domain() == originalDomain {
		t.Error("SetDomain should change the domain")
	}
	if v.Domain() != newDomain {
		t.Error("SetDomain should set the new domain")
	}
	if v.Domain().Count() != 1 {
		t.Errorf("SetDomain domain count = %d, want 1", v.Domain().Count())
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
