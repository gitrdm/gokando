package minikanren

import (
	"context"
	"testing"
)

// Test the creation of Modulo constraint
func TestNewModulo(t *testing.T) {
	model := NewModel()

	tests := []struct {
		name            string
		xDomain         []int
		modulus         int
		remainderDomain []int
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "valid_constraint",
			xDomain:         []int{1, 2, 3, 4, 5},
			modulus:         3,
			remainderDomain: []int{1, 2, 3},
			expectError:     false,
		},
		{
			name:            "zero_modulus",
			xDomain:         []int{1, 2, 3},
			modulus:         0,
			remainderDomain: []int{1},
			expectError:     true,
			errorMsg:        "modulus must be > 0",
		},
		{
			name:            "negative_modulus",
			xDomain:         []int{1, 2, 3},
			modulus:         -2,
			remainderDomain: []int{1, 2},
			expectError:     true,
			errorMsg:        "modulus must be > 0",
		},
		{
			name:            "modulus_one",
			xDomain:         []int{10, 20, 30},
			modulus:         1,
			remainderDomain: []int{1},
			expectError:     false,
		},
		{
			name:            "large_modulus",
			xDomain:         []int{1, 2},
			modulus:         100,
			remainderDomain: []int{1, 2},
			expectError:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var xVar, remainderVar *FDVariable
			if len(test.xDomain) > 0 {
				xVar = model.NewVariable(NewBitSetDomainFromValues(maxValue(test.xDomain)+1, test.xDomain))
			}
			if len(test.remainderDomain) > 0 {
				remainderVar = model.NewVariable(NewBitSetDomainFromValues(maxValue(test.remainderDomain)+1, test.remainderDomain))
			}

			constraint, err := NewModulo(xVar, test.modulus, remainderVar)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", test.errorMsg)
				} else if !containsString(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if constraint == nil {
					t.Error("Expected non-nil constraint")
				}
			}
		})
	}
}

// Test nil variable handling
func TestNewModulo_NilVariables(t *testing.T) {
	model := NewModel()
	validVar := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3}))

	tests := []struct {
		name      string
		x         *FDVariable
		remainder *FDVariable
	}{
		{"nil_x", nil, validVar},
		{"nil_remainder", validVar, nil},
		{"both_nil", nil, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewModulo(test.x, 5, test.remainder)
			if err == nil {
				t.Error("Expected error for nil variable(s)")
			}
			if !containsString(err.Error(), "variables cannot be nil") {
				t.Errorf("Expected 'variables cannot be nil' error, got: %v", err)
			}
		})
	}
}

// Test basic constraint interface methods
func TestModulo_Interface(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3, 4, 5}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(4, []int{1, 2, 3}))

	constraint, err := NewModulo(x, 3, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	// Test Variables()
	vars := constraint.Variables()
	if len(vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(vars))
	}
	if vars[0] != x || vars[1] != remainder {
		t.Error("Variables not returned in correct order")
	}

	// Test Type()
	if constraint.Type() != "Modulo" {
		t.Errorf("Expected type 'Modulo', got '%s'", constraint.Type())
	}

	// Test String()
	str := constraint.String()
	if !containsString(str, "Modulo") {
		t.Errorf("String representation should contain 'Modulo', got: %s", str)
	}
	if !containsString(str, "3") {
		t.Errorf("String representation should contain modulus '3', got: %s", str)
	}
}

// Test forward propagation: remainder ← x mod modulus
func TestModulo_ForwardPropagation(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3, 4, 5, 6, 7}, modulus = 3
	// Expected remainder after propagation: {1, 2, 3} (since 0 maps to 3)
	x := model.NewVariable(NewBitSetDomainFromValues(8, []int{1, 2, 3, 4, 5, 6, 7}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(6, []int{1, 2, 3, 4, 5}))

	constraint, err := NewModulo(x, 3, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	// Solve to trigger propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check remainder was pruned to {1, 2, 3}
	finalRemainder := solver.GetDomain(nil, remainder.ID())
	if finalRemainder.Count() != 3 {
		t.Errorf("Expected remainder domain size 3, got %d", finalRemainder.Count())
	}

	expected := map[int]bool{1: true, 2: true, 3: true}
	finalRemainder.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d in remainder domain", v)
		}
		delete(expected, v)
	})

	if len(expected) > 0 {
		t.Errorf("Missing values in remainder domain: %v", expected)
	}
}

// Test backward propagation: x ← values that produce valid remainders
func TestModulo_BackwardPropagation(t *testing.T) {
	model := NewModel()

	// remainder = {1, 3}, modulus = 5
	// Should constrain x to values where x mod 5 ∈ {1, 3}
	// i.e., x ∈ {1, 3, 6, 8, 11, 13, ...}
	x := model.NewVariable(NewBitSetDomainFromValues(21, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(4, []int{1, 3}))

	constraint, err := NewModulo(x, 5, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())

	// Expected x values: {1, 3, 6, 8, 11, 13, 16, 18}
	// Note: 5 mod 5 = 0 which maps to 5, 10 mod 5 = 0 which maps to 5, etc.
	expected := map[int]bool{1: true, 3: true, 6: true, 8: true, 11: true, 13: true, 16: true, 18: true}

	if finalX.Count() != len(expected) {
		t.Errorf("Expected x domain size %d, got %d", len(expected), finalX.Count())
	}

	finalX.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d in x domain", v)
		}
		delete(expected, v)
	})

	if len(expected) > 0 {
		t.Errorf("Missing values in x domain: %v", expected)
	}
}

// Test bidirectional propagation
func TestModulo_BidirectionalPropagation(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, modulus = 4
	// remainder = {1, 2, 4, 5}
	// Expected after propagation: x = {1, 2, 4, 5, 6, 8, 9, 10}, remainder = {1, 2, 4}
	// Note: modulus=4, so remainders are {1,2,3,4} where 4 represents 0
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(6, []int{1, 2, 4, 5}))

	constraint, err := NewModulo(x, 4, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())
	finalRemainder := solver.GetDomain(nil, remainder.ID())

	// Check x domain: values where x mod 4 ∈ {1, 2, 0}
	// x mod 4 = 1: x ∈ {1, 5, 9}
	// x mod 4 = 2: x ∈ {2, 6, 10}
	// x mod 4 = 0: x ∈ {4, 8} (0 maps to 4)
	expectedX := map[int]bool{1: true, 2: true, 4: true, 5: true, 6: true, 8: true, 9: true, 10: true}

	if finalX.Count() != len(expectedX) {
		t.Errorf("Expected x domain size %d, got %d", len(expectedX), finalX.Count())
	}
	finalX.IterateValues(func(v int) {
		if !expectedX[v] {
			t.Errorf("Unexpected value %d in x domain", v)
		}
		delete(expectedX, v)
	})

	// Check remainder domain: only {1, 2, 4} are possible (5 is not a valid remainder mod 4)
	expectedRemainder := map[int]bool{1: true, 2: true, 4: true}

	if finalRemainder.Count() != len(expectedRemainder) {
		t.Errorf("Expected remainder domain size %d, got %d", len(expectedRemainder), finalRemainder.Count())
	}
	finalRemainder.IterateValues(func(v int) {
		if !expectedRemainder[v] {
			t.Errorf("Unexpected value %d in remainder domain", v)
		}
		delete(expectedRemainder, v)
	})
}

// Test modulo by 1 (everything has remainder 0, represented as 1)
func TestModulo_ModulusOne(t *testing.T) {
	model := NewModel()

	// x = {5, 10, 15}, modulus = 1
	// remainder should be forced to {1} (representing 0)
	x := model.NewVariable(NewBitSetDomainFromValues(16, []int{5, 10, 15}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(3, []int{1, 2}))

	constraint, err := NewModulo(x, 1, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalRemainder := solver.GetDomain(nil, remainder.ID())

	// All values mod 1 = 0, which is represented as 1
	if !finalRemainder.IsSingleton() || finalRemainder.SingletonValue() != 1 {
		t.Errorf("Expected remainder to be singleton {1}, got count=%d", finalRemainder.Count())
	}
}

// Test self-reference constraint: X mod modulus = X
func TestModulo_SelfReference(t *testing.T) {
	tests := []struct {
		name           string
		domain         []int
		modulus        int
		expectedValues []int
		expectError    bool
		description    string
	}{
		{
			name:           "valid_self_reference",
			domain:         []int{1, 2, 3, 4, 5},
			modulus:        7,
			expectedValues: []int{1, 2, 3, 4, 5}, // All values < 7
			expectError:    false,
			description:    "X mod 7 = X valid when X < 7",
		},
		{
			name:           "partial_valid_self_reference",
			domain:         []int{3, 4, 5, 6, 7, 8},
			modulus:        6,
			expectedValues: []int{3, 4, 5}, // Only values < 6
			expectError:    false,
			description:    "X mod 6 = X filters to values < 6",
		},
		{
			name:           "no_valid_self_reference",
			domain:         []int{5, 6, 7, 8},
			modulus:        4,
			expectedValues: []int{}, // No values < 4
			expectError:    true,
			description:    "X mod 4 = X invalid when all X >= 4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := NewModel()
			x := model.NewVariable(NewBitSetDomainFromValues(maxValue(test.domain)+1, test.domain))

			constraint, err := NewModulo(x, test.modulus, x) // Self-reference
			if err != nil {
				t.Fatalf("Failed to create constraint: %v", err)
			}

			model.AddConstraint(constraint)

			solver := NewSolver(model)
			ctx := context.Background()
			solutions, err := solver.Solve(ctx, 1)
			if err != nil {
				t.Fatalf("Unexpected error during solving: %v", err)
			}

			if test.expectError {
				// Should have no solutions when constraint is unsatisfiable
				if len(solutions) != 0 {
					t.Errorf("Expected no solutions for %s, got %d solutions", test.description, len(solutions))
				}
			} else {
				// Should have solutions - check the domain values match expected
				finalX := solver.GetDomain(nil, x.ID())
				actualValues := domainToSlice(finalX)
				if !equalSlices(actualValues, test.expectedValues) {
					t.Errorf("Expected domain %v for %s, got %v", test.expectedValues, test.description, actualValues)
				}
			}
		})
	}
}

// Test computeModulo function handles BitSetDomain constraint (0 → modulus)
func TestModulo_ComputeModulo(t *testing.T) {
	constraint := &Modulo{modulus: 5}

	tests := []struct {
		input    int
		expected int
	}{
		{1, 1},  // 1 mod 5 = 1
		{2, 2},  // 2 mod 5 = 2
		{4, 4},  // 4 mod 5 = 4
		{5, 5},  // 5 mod 5 = 0 → 5 (BitSetDomain constraint)
		{6, 1},  // 6 mod 5 = 1
		{10, 5}, // 10 mod 5 = 0 → 5
		{13, 3}, // 13 mod 5 = 3
	}

	for _, test := range tests {
		result := constraint.computeModulo(test.input)
		if result != test.expected {
			t.Errorf("computeModulo(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

// Test Clone method
func TestModulo_Clone(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3, 4, 5}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(4, []int{1, 2, 3}))

	original, err := NewModulo(x, 3, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	cloned := original.Clone().(*Modulo)

	// Verify clone has same structure
	if cloned.x != original.x {
		t.Error("Cloned constraint should reference same x variable")
	}
	if cloned.remainder != original.remainder {
		t.Error("Cloned constraint should reference same remainder variable")
	}
	if cloned.modulus != original.modulus {
		t.Errorf("Expected modulus %d, got %d", original.modulus, cloned.modulus)
	}

	// Verify they are separate objects
	if cloned == original {
		t.Error("Clone should be a different object")
	}
}

// Test edge case: large modulus with small domain
func TestModulo_LargeModulus(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3}, modulus = 100
	// remainder should be constrained to {1, 2, 3}
	x := model.NewVariable(NewBitSetDomainFromValues(4, []int{1, 2, 3}))
	remainder := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(1, 100)))

	constraint, err := NewModulo(x, 100, remainder)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalRemainder := solver.GetDomain(nil, remainder.ID())

	// Since all x values are < 100, remainder = x
	expected := map[int]bool{1: true, 2: true, 3: true}

	if finalRemainder.Count() != len(expected) {
		t.Errorf("Expected remainder domain size %d, got %d", len(expected), finalRemainder.Count())
	}

	finalRemainder.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d in remainder domain", v)
		}
		delete(expected, v)
	})
}

// Benchmark propagation performance
func BenchmarkModulo_Propagate(b *testing.B) {
	model := NewModel()

	// Large domains for performance testing
	xValues := make([]int, 100)
	remainderValues := make([]int, 7)
	for i := 0; i < 100; i++ {
		xValues[i] = i + 1
	}
	for i := 0; i < 7; i++ {
		remainderValues[i] = i + 1
	}

	x := model.NewVariable(NewBitSetDomainFromValues(101, xValues))
	remainder := model.NewVariable(NewBitSetDomainFromValues(8, remainderValues))

	constraint, err := NewModulo(x, 7, remainder)
	if err != nil {
		b.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)
	solver := NewSolver(model)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset solver state for each iteration
		solver = NewSolver(model)
		solver.Solve(ctx, 1)
	}
}

// Helper functions for testing (reused from scale_test.go)

func domainToSlice(domain Domain) []int {
	var values []int
	min, max := domain.Min(), domain.Max()
	for v := min; v <= max; v++ {
		if domain.Has(v) {
			values = append(values, v)
		}
	}
	return values
}

func equalSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
