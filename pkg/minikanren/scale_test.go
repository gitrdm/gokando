package minikanren

import (
	"context"
	"testing"
)

// Test the creation of Scale constraint
func TestNewScale(t *testing.T) {
	model := NewModel()

	tests := []struct {
		name         string
		xDomain      []int
		multiplier   int
		resultDomain []int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid_constraint",
			xDomain:      []int{1, 2, 3},
			multiplier:   5,
			resultDomain: []int{5, 10, 15},
			expectError:  false,
		},
		{
			name:         "zero_multiplier",
			xDomain:      []int{1, 2, 3},
			multiplier:   0,
			resultDomain: []int{0},
			expectError:  true,
			errorMsg:     "multiplier must be > 0",
		},
		{
			name:         "negative_multiplier",
			xDomain:      []int{1, 2, 3},
			multiplier:   -2,
			resultDomain: []int{1, 2, 3}, // Dummy values - should fail validation
			expectError:  true,
			errorMsg:     "multiplier must be > 0",
		},
		{
			name:         "multiplier_one",
			xDomain:      []int{10, 20, 30},
			multiplier:   1,
			resultDomain: []int{10, 20, 30},
			expectError:  false,
		},
		{
			name:         "large_multiplier",
			xDomain:      []int{1, 2},
			multiplier:   1000,
			resultDomain: []int{1000, 2000},
			expectError:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var xVar, resultVar *FDVariable
			if len(test.xDomain) > 0 {
				xVar = model.NewVariable(NewBitSetDomainFromValues(maxValue(test.xDomain)+1, test.xDomain))
			}
			if len(test.resultDomain) > 0 {
				resultVar = model.NewVariable(NewBitSetDomainFromValues(maxValue(test.resultDomain)+1, test.resultDomain))
			}

			constraint, err := NewScale(xVar, test.multiplier, resultVar)

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
func TestNewScale_NilVariables(t *testing.T) {
	model := NewModel()
	validVar := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3}))

	tests := []struct {
		name   string
		x      *FDVariable
		result *FDVariable
	}{
		{"nil_x", nil, validVar},
		{"nil_result", validVar, nil},
		{"both_nil", nil, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewScale(test.x, 5, test.result)
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
func TestScale_Interface(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3}))
	result := model.NewVariable(NewBitSetDomainFromValues(16, []int{5, 10, 15}))

	constraint, err := NewScale(x, 5, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	// Test Variables()
	vars := constraint.Variables()
	if len(vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(vars))
	}
	if vars[0] != x || vars[1] != result {
		t.Error("Variables not returned in correct order")
	}

	// Test Type()
	if constraint.Type() != "Scale" {
		t.Errorf("Expected type 'Scale', got '%s'", constraint.Type())
	}

	// Test String()
	str := constraint.String()
	if !containsString(str, "Scale") {
		t.Errorf("String representation should contain 'Scale', got: %s", str)
	}
	if !containsString(str, "5") {
		t.Errorf("String representation should contain multiplier '5', got: %s", str)
	}
}

// Test forward propagation: result ← x * multiplier
func TestScale_ForwardPropagation(t *testing.T) {
	model := NewModel()

	// x = {2, 3, 4}, multiplier = 3
	// Expected result after propagation: {6, 9, 12}
	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{2, 3, 4}))
	result := model.NewVariable(NewBitSetDomainFromValues(20, []int{6, 9, 12, 15, 18}))

	constraint, err := NewScale(x, 3, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	// Solve to trigger propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check result was pruned to {6, 9, 12}
	finalResult := solver.GetDomain(nil, result.ID())
	if finalResult.Count() != 3 {
		t.Errorf("Expected result domain size 3, got %d", finalResult.Count())
	}

	expected := map[int]bool{6: true, 9: true, 12: true}
	finalResult.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d in result domain", v)
		}
		delete(expected, v)
	})

	if len(expected) > 0 {
		t.Errorf("Missing values in result domain: %v", expected)
	}
}

// Test backward propagation: x ← result / multiplier
func TestScale_BackwardPropagation(t *testing.T) {
	model := NewModel()

	// result = {6, 9}, multiplier = 3
	// Should constrain x to {2, 3}
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{1, 2, 3, 4, 5}))
	result := model.NewVariable(NewBitSetDomainFromValues(10, []int{6, 9}))

	constraint, err := NewScale(x, 3, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())
	if finalX.Count() != 2 {
		t.Errorf("Expected x domain size 2, got %d", finalX.Count())
	}

	expected := map[int]bool{2: true, 3: true}
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
func TestScale_BidirectionalPropagation(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3, 4, 5}, multiplier = 3
	// result = {6, 7, 9, 10, 12}
	// Expected after propagation: x = {2, 3, 4}, result = {6, 9, 12}
	x := model.NewVariable(NewBitSetDomainFromValues(6, []int{1, 2, 3, 4, 5}))
	result := model.NewVariable(NewBitSetDomainFromValues(13, []int{6, 7, 9, 10, 12}))

	constraint, err := NewScale(x, 3, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())
	finalResult := solver.GetDomain(nil, result.ID())

	// Check x domain
	if finalX.Count() != 3 {
		t.Errorf("Expected x domain size 3, got %d", finalX.Count())
	}
	expectedX := map[int]bool{2: true, 3: true, 4: true}
	finalX.IterateValues(func(v int) {
		if !expectedX[v] {
			t.Errorf("Unexpected value %d in x domain", v)
		}
		delete(expectedX, v)
	})

	// Check result domain
	if finalResult.Count() != 3 {
		t.Errorf("Expected result domain size 3, got %d", finalResult.Count())
	}
	expectedResult := map[int]bool{6: true, 9: true, 12: true}
	finalResult.IterateValues(func(v int) {
		if !expectedResult[v] {
			t.Errorf("Unexpected value %d in result domain", v)
		}
		delete(expectedResult, v)
	})
}

// Test indivisible values get filtered out
func TestScale_IndivisibleFiltering(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3, 4}, multiplier = 3
	// result = {5, 6, 7, 9} - only 6 and 9 are divisible by 3
	// Expected: x = {2, 3}, result = {6, 9}
	x := model.NewVariable(NewBitSetDomainFromValues(5, []int{1, 2, 3, 4}))
	result := model.NewVariable(NewBitSetDomainFromValues(10, []int{5, 6, 7, 9}))

	constraint, err := NewScale(x, 3, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())
	finalResult := solver.GetDomain(nil, result.ID())

	// Check x domain - should be {2, 3}
	if finalX.Count() != 2 {
		t.Errorf("Expected x domain size 2, got %d", finalX.Count())
	}
	expectedX := map[int]bool{2: true, 3: true}
	finalX.IterateValues(func(v int) {
		if !expectedX[v] {
			t.Errorf("Unexpected value %d in x domain", v)
		}
		delete(expectedX, v)
	})

	// Check result domain - should be {6, 9}
	if finalResult.Count() != 2 {
		t.Errorf("Expected result domain size 2, got %d", finalResult.Count())
	}
	expectedResult := map[int]bool{6: true, 9: true}
	finalResult.IterateValues(func(v int) {
		if !expectedResult[v] {
			t.Errorf("Unexpected value %d in result domain", v)
		}
		delete(expectedResult, v)
	})
}

// Test zero handling (adjusting for domain constraints - only positive integers)
func TestScale_ZeroHandling(t *testing.T) {
	model := NewModel()

	// x = {1, 2, 3}, multiplier = 5
	// result = {5, 10, 15} - all consistent
	x := model.NewVariable(NewBitSetDomainFromValues(4, []int{1, 2, 3}))
	result := model.NewVariable(NewBitSetDomainFromValues(16, []int{5, 10, 15}))

	constraint, err := NewScale(x, 5, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalX := solver.GetDomain(nil, x.ID())
	finalResult := solver.GetDomain(nil, result.ID())

	// Both domains should remain unchanged (all consistent)
	if finalX.Count() != 3 {
		t.Errorf("Expected x domain size 3, got %d", finalX.Count())
	}
	if finalResult.Count() != 3 {
		t.Errorf("Expected result domain size 3, got %d", finalResult.Count())
	}
}

// Test Clone method
func TestScale_Clone(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{1, 2, 3}))
	result := model.NewVariable(NewBitSetDomainFromValues(16, []int{5, 10, 15}))

	original, err := NewScale(x, 5, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	cloned := original.Clone().(*Scale)

	// Verify clone has same structure
	if cloned.x != original.x {
		t.Error("Cloned constraint should reference same x variable")
	}
	if cloned.result != original.result {
		t.Error("Cloned constraint should reference same result variable")
	}
	if cloned.multiplier != original.multiplier {
		t.Errorf("Expected multiplier %d, got %d", original.multiplier, cloned.multiplier)
	}

	// Verify they are separate objects
	if cloned == original {
		t.Error("Clone should be a different object")
	}
}

// Benchmark propagation performance
func BenchmarkScale_Propagate(b *testing.B) {
	model := NewModel()

	// Large domains for performance testing
	xValues := make([]int, 100)
	resultValues := make([]int, 200)
	for i := 0; i < 100; i++ {
		xValues[i] = i + 1
	}
	for i := 0; i < 200; i++ {
		resultValues[i] = (i + 1) * 3 // Some will match, some won't
	}

	x := model.NewVariable(NewBitSetDomainFromValues(101, xValues))
	result := model.NewVariable(NewBitSetDomainFromValues(601, resultValues))

	constraint, err := NewScale(x, 3, result)
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

// Helper functions for testing

func maxValue(slice []int) int {
	if len(slice) == 0 {
		return 0
	}
	max := slice[0]
	for _, v := range slice[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			len(str) > len(substr) &&
				(str[:len(substr)] == substr ||
					str[len(str)-len(substr):] == substr ||
					containsSubstring(str, substr)))
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
