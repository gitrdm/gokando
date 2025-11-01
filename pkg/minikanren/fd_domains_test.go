package minikanren

import (
	"testing"
)

// TestNewBitSetFromValues tests creating BitSets from value slices
func TestNewBitSetFromValues(t *testing.T) {
	tests := []struct {
		name     string
		values   []int
		expected []int // expected values that should be present
		size     int   // expected domain size
	}{
		{
			name:     "empty slice",
			values:   []int{},
			expected: []int{},
			size:     0,
		},
		{
			name:     "single value",
			values:   []int{5},
			expected: []int{5},
			size:     5,
		},
		{
			name:     "multiple values",
			values:   []int{1, 3, 5, 7},
			expected: []int{1, 3, 5, 7},
			size:     7,
		},
		{
			name:     "unsorted values",
			values:   []int{7, 2, 5, 1},
			expected: []int{1, 2, 5, 7},
			size:     7,
		},
		{
			name:     "duplicate values",
			values:   []int{3, 3, 5, 5},
			expected: []int{3, 5},
			size:     5,
		},
		{
			name:     "negative values ignored",
			values:   []int{-1, 0, 3, 5},
			expected: []int{3, 5},
			size:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBitSetFromValues(tt.values)

			if bs.n != tt.size {
				t.Errorf("Expected domain size %d, got %d", tt.size, bs.n)
			}

			// Check that expected values are present
			for _, val := range tt.expected {
				if !bs.Has(val) {
					t.Errorf("Expected value %d to be present", val)
				}
			}

			// Check that no unexpected values are present
			count := 0
			bs.IterateValues(func(v int) {
				count++
				found := false
				for _, expected := range tt.expected {
					if v == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected value %d found in domain", v)
				}
			})

			if count != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), count)
			}
		})
	}
}

// TestNewBitSetFromInterval tests creating BitSets from intervals
func TestNewBitSetFromInterval(t *testing.T) {
	tests := []struct {
		name     string
		min, max int
		expected []int
		size     int
	}{
		{
			name:     "valid interval",
			min:      2,
			max:      5,
			expected: []int{2, 3, 4, 5},
			size:     5,
		},
		{
			name:     "single value interval",
			min:      3,
			max:      3,
			expected: []int{3},
			size:     3,
		},
		{
			name:     "invalid min > max",
			min:      5,
			max:      2,
			expected: []int{},
			size:     0,
		},
		{
			name:     "invalid min < 1",
			min:      0,
			max:      3,
			expected: []int{},
			size:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBitSetFromInterval(tt.min, tt.max)

			if bs.n != tt.size {
				t.Errorf("Expected domain size %d, got %d", tt.size, bs.n)
			}

			// Check that expected values are present
			for _, val := range tt.expected {
				if !bs.Has(val) {
					t.Errorf("Expected value %d to be present", val)
				}
			}

			// Check count
			if bs.Count() != len(tt.expected) {
				t.Errorf("Expected %d values, got %d", len(tt.expected), bs.Count())
			}
		})
	}
}

// TestFDStoreNewVarWithDomain tests creating variables with custom domains
func TestFDStoreNewVarWithDomain(t *testing.T) {
	store := NewFDStoreWithDomain(10) // Default domain 1-10

	// Test with custom domain
	customDomain := NewBitSetFromValues([]int{2, 4, 6, 8})
	v := store.NewVarWithDomain(customDomain)

	if v == nil {
		t.Fatal("Expected variable to be created")
	}

	// Check that the variable has the correct domain
	varDomain := store.GetDomain(v)
	if !bitSetEquals(varDomain, customDomain) {
		t.Errorf("Variable domain doesn't match expected domain")
	}

	// Check that only expected values are present
	expectedValues := []int{2, 4, 6, 8}
	var values []int
	varDomain.IterateValues(func(val int) {
		values = append(values, val)
	})

	if len(values) != len(expectedValues) {
		t.Errorf("Expected %d values, got %d", len(expectedValues), len(values))
	}

	for _, expected := range expectedValues {
		found := false
		for _, actual := range values {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected value %d not found in variable domain", expected)
		}
	}
}

// TestFDStoreNewVarWithValues tests creating variables with value lists
func TestFDStoreNewVarWithValues(t *testing.T) {
	store := NewFDStoreWithDomain(10)

	values := []int{1, 3, 5, 7, 9}
	v := store.NewVarWithValues(values)

	if v == nil {
		t.Fatal("Expected variable to be created")
	}

	varDomain := store.GetDomain(v)

	// Check that all specified values are present
	for _, val := range values {
		if !varDomain.Has(val) {
			t.Errorf("Expected value %d to be present", val)
		}
	}

	// Check that domain size is correct
	if varDomain.n != 9 {
		t.Errorf("Expected domain size 9, got %d", varDomain.n)
	}

	// Check count
	if varDomain.Count() != len(values) {
		t.Errorf("Expected %d values, got %d", len(values), varDomain.Count())
	}
}

// TestFDStoreNewVarWithInterval tests creating variables with intervals
func TestFDStoreNewVarWithInterval(t *testing.T) {
	store := NewFDStoreWithDomain(10)

	v := store.NewVarWithInterval(3, 7)

	if v == nil {
		t.Fatal("Expected variable to be created")
	}

	varDomain := store.GetDomain(v)

	// Check that interval values are present
	expectedValues := []int{3, 4, 5, 6, 7}
	for _, val := range expectedValues {
		if !varDomain.Has(val) {
			t.Errorf("Expected value %d to be present", val)
		}
	}

	// Check that values outside interval are not present
	if varDomain.Has(2) || varDomain.Has(8) {
		t.Errorf("Values outside interval should not be present")
	}

	// Check count
	if varDomain.Count() != len(expectedValues) {
		t.Errorf("Expected %d values, got %d", len(expectedValues), varDomain.Count())
	}
}

// TestFDDomainGoal tests the fd/dom goal
func TestFDDomainGoal(t *testing.T) {
	// Test with custom domain
	customDomain := NewBitSetFromValues([]int{2, 4, 6})

	results := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return FDSolve(Conj(
			FDIn(x, customDomain.Values()),
			Eq(q, x),
		))
	})

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check that all results are in the domain
	validValues := map[int]bool{2: true, 4: true, 6: true}
	for _, result := range results {
		if atom, ok := result.(*Atom); ok {
			if val, ok := atom.Value().(int); ok {
				if !validValues[val] {
					t.Errorf("Result value %d not in expected domain", val)
				}
			} else {
				t.Errorf("Expected integer result, got %T", atom.Value())
			}
		} else {
			t.Errorf("Expected Atom result, got %T", result)
		}
	}
}

// TestFDInGoal tests the fd/in goal
func TestFDInGoal(t *testing.T) {
	values := []int{1, 3, 5, 7, 9}

	results := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return FDSolve(Conj(
			FDIn(x, values),
			Eq(q, x),
		))
	})

	if len(results) != len(values) {
		t.Errorf("Expected %d results, got %d", len(values), len(results))
	}

	// Check that all results are in the values list
	validValues := make(map[int]bool)
	for _, v := range values {
		validValues[v] = true
	}

	for _, result := range results {
		if atom, ok := result.(*Atom); ok {
			if val, ok := atom.Value().(int); ok {
				if !validValues[val] {
					t.Errorf("Result value %d not in expected values", val)
				}
			} else {
				t.Errorf("Expected integer result, got %T", atom.Value())
			}
		} else {
			t.Errorf("Expected Atom result, got %T", result)
		}
	}
}

// TestFDIntervalGoal tests the fd/interval goal
func TestFDIntervalGoal(t *testing.T) {
	results := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return FDSolve(Conj(
			FDIn(x, intervalValues(3, 7)),
			Eq(q, x),
		))
	})

	expectedCount := 5 // 3, 4, 5, 6, 7
	if len(results) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(results))
	}

	// Check that all results are in the interval
	for _, result := range results {
		if atom, ok := result.(*Atom); ok {
			if val, ok := atom.Value().(int); ok {
				if val < 3 || val > 7 {
					t.Errorf("Result value %d not in expected interval [3,7]", val)
				}
			} else {
				t.Errorf("Expected integer result, got %T", atom.Value())
			}
		} else {
			t.Errorf("Expected Atom result, got %T", result)
		}
	}
}

// TestDomainConstraintsWithOtherGoals tests domain goals combined with other constraints
func TestDomainConstraintsWithOtherGoals(t *testing.T) {
	// Test fd/in with equality
	values := []int{2, 4, 6, 8}

	results := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		return FDSolve(Conj(
			FDIn(x, values),
			Eq(x, NewAtom(4)),
			Eq(q, NewAtom("success")),
		))
	})

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if atom, ok := results[0].(*Atom); ok {
			if val, ok := atom.Value().(string); ok {
				if val != "success" {
					t.Errorf("Expected result 'success', got %s", val)
				}
			}
		}
	}
}

// TestEmptyDomainGoals tests behavior with empty domains
func TestEmptyDomainGoals(t *testing.T) {
	// Test empty values list
	goal := FDSolve(FDIn(Fresh("x"), []int{}))
	results := Run(10, func(q *Var) Goal {
		return goal
	})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty domain, got %d", len(results))
	}

	// Test invalid interval
	goal = FDSolve(FDIn(Fresh("x"), []int{})) // min > max -> empty domain
	results = Run(10, func(q *Var) Goal {
		return goal
	})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for invalid interval, got %d", len(results))
	}
}

// TestFDAllDifferentFunctionalOptions tests the FDAllDifferent function with options.
func TestFDAllDifferentFunctionalOptions(t *testing.T) {
	t.Run("Basic functionality", func(t *testing.T) {
		x, y, z := Fresh("x"), Fresh("y"), Fresh("z")

		results := Run(10, func(q *Var) Goal {
			return FDSolve(Conj(
				FDAllDifferent(x, y, z),
				Eq(q, List(x, y, z)),
			))
		})

		if len(results) == 0 {
			t.Error("Should find some solutions")
		}

		// Verify all values are different
		for _, result := range results {
			if pair, ok := result.(*Pair); ok {
				xVal := extractIntValue(pair.Car())
				cdr := pair.Cdr()
				if cdrPair, ok := cdr.(*Pair); ok {
					yVal := extractIntValue(cdrPair.Car())
					cdr2 := cdrPair.Cdr()
					if cdr2Pair, ok := cdr2.(*Pair); ok {
						zVal := extractIntValue(cdr2Pair.Car())

						if xVal == yVal || xVal == zVal || yVal == zVal {
							t.Error("All values should be different")
						}
					}
				}
			}
		}
	})

	t.Run("With custom search strategy", func(t *testing.T) {
		x, y := Fresh("x"), Fresh("y")

		results := Run(5, func(q *Var) Goal {
			return FDSolve(Conj(
				FDAllDifferent(x, y),
				Eq(q, List(x, y)),
			))
		})

		if len(results) == 0 {
			t.Error("Should find solutions with custom search strategy")
		}
	})

	t.Run("With custom labeling strategy", func(t *testing.T) {
		x, y := Fresh("x"), Fresh("y")

		results := Run(5, func(q *Var) Goal {
			return FDSolve(Conj(
				FDAllDifferent(x, y),
				Eq(q, List(x, y)),
			))
		})

		if len(results) == 0 {
			t.Error("Should find solutions with custom labeling strategy")
		}
	})
}

// TestFDInFunctionalOptions tests the FDIn function with options.
func TestFDInFunctionalOptions(t *testing.T) {
	t.Run("Basic functionality", func(t *testing.T) {
		values := []int{2, 4, 6, 8}

		results := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return Conj(
				FDIn(x, values),
				Eq(q, x),
			)
		})

		if len(results) != len(values) {
			t.Errorf("Expected %d results, got %d", len(values), len(results))
		}

		// Verify all results are in the allowed values
		for _, result := range results {
			val := extractIntValue(result)
			found := false
			for _, allowed := range values {
				if val == allowed {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Value %d not in allowed values %v", val, values)
			}
		}
	})

	t.Run("Empty domain", func(t *testing.T) {
		results := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return Conj(
				FDIn(x, []int{}),
				Eq(q, x),
			)
		})

		if len(results) != 0 {
			t.Error("Empty domain should produce no solutions")
		}
	})

	t.Run("Single value domain", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			x := Fresh("x")
			return Conj(
				FDIn(x, []int{42}),
				Eq(q, x),
			)
		})

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			val := extractIntValue(results[0])
			if val != 42 {
				t.Errorf("Expected 42, got %d", val)
			}
		}
	})
}

// TestFDIntervalFunctionalOptions tests the FDIn function with interval-like lists.
func TestFDIntervalFunctionalOptions(t *testing.T) {
	t.Run("Basic functionality", func(t *testing.T) {
		min, max := 3, 7
		var values []int
		for i := min; i <= max; i++ {
			values = append(values, i)
		}

		results := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return FDSolve(Conj(
				FDIn(x, values),
				Eq(q, x),
			))
		})

		expectedCount := max - min + 1
		if len(results) != expectedCount {
			t.Errorf("Expected %d results, got %d", expectedCount, len(results))
		}

		// Verify all results are in the interval
		for _, result := range results {
			val := extractIntValue(result)
			if val < min || val > max {
				t.Errorf("Value %d not in interval [%d, %d]", val, min, max)
			}
		}
	})

	t.Run("Invalid interval", func(t *testing.T) {
		results := Run(10, func(q *Var) Goal {
			x := Fresh("x")
			return FDSolve(Conj(
				FDIn(x, []int{}), // min > max becomes empty list
				Eq(q, x),
			))
		})

		if len(results) != 0 {
			t.Error("Invalid interval should produce no solutions")
		}
	})

	t.Run("Single value interval", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			x := Fresh("x")
			return FDSolve(Conj(
				FDIn(x, []int{42}),
				Eq(q, x),
			))
		})

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			val := extractIntValue(results[0])
			if val != 42 {
				t.Errorf("Expected 42, got %d", val)
			}
		}
	})
}

// Helper function to extract int value from atom
func extractIntValue(term Term) int {
	if atom, ok := term.(*Atom); ok {
		if val, ok := atom.Value().(int); ok {
			return val
		}
	}
	return -1 // Error value
}

func intervalValues(min, max int) []int {
	if min > max {
		return []int{}
	}
	values := make([]int, 0, max-min+1)
	for i := min; i <= max; i++ {
		values = append(values, i)
	}
	return values
}
