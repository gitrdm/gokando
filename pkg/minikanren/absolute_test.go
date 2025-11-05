package minikanren

import (
	"context"
	"testing"
)

// Test constraint creation
func TestNewAbsolute(t *testing.T) {
	model := NewModel()

	tests := []struct {
		name     string
		setupX   func() *FDVariable
		offset   int
		setupAbs func() *FDVariable
		errorMsg string
	}{
		{
			name:     "valid_constraint",
			setupX:   func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			offset:   10,
			setupAbs: func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) },
			errorMsg: "",
		},
		{
			name:     "zero_offset",
			setupX:   func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			offset:   0,
			setupAbs: func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) },
			errorMsg: "offset must be > 0",
		},
		{
			name:     "negative_offset",
			setupX:   func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			offset:   -5,
			setupAbs: func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) },
			errorMsg: "offset must be > 0",
		},
		{
			name:     "large_offset",
			setupX:   func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(1, 100))) },
			offset:   50,
			setupAbs: func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(51, rangeValues(1, 50))) },
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var xVar, absVar *FDVariable
			if test.setupX != nil {
				xVar = test.setupX()
			}
			if test.setupAbs != nil {
				absVar = test.setupAbs()
			}

			constraint, err := NewAbsolute(xVar, test.offset, absVar)

			if test.errorMsg != "" {
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
func TestNewAbsolute_NilVariables(t *testing.T) {
	model := NewModel()
	validVar := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))

	tests := []struct {
		name   string
		x      *FDVariable
		absVal *FDVariable
	}{
		{"nil_x", nil, validVar},
		{"nil_absValue", validVar, nil},
		{"both_nil", nil, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			constraint, err := NewAbsolute(test.x, 10, test.absVal)

			if err == nil {
				t.Error("Expected error for nil variable, got nil")
			} else if !containsString(err.Error(), "cannot be nil") {
				t.Errorf("Expected error about nil variables, got: %s", err.Error())
			}

			if constraint != nil {
				t.Error("Expected nil constraint for invalid input")
			}
		})
	}
}

// Test constraint interface compliance
func TestAbsolute_Interface(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	absValue := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))

	constraint, err := NewAbsolute(x, 10, absValue)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	// Test Variables method
	vars := constraint.Variables()
	if len(vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(vars))
	}

	if vars[0] != x || vars[1] != absValue {
		t.Error("Variables returned in wrong order or wrong variables")
	}

	// Test Type method
	if constraint.Type() != "Absolute" {
		t.Errorf("Expected type 'Absolute', got '%s'", constraint.Type())
	}

	// Test String method
	str := constraint.String()
	if str == "" {
		t.Error("String method returned empty string")
	}
	if !containsString(str, "Absolute") {
		t.Errorf("String method should contain 'Absolute', got: %s", str)
	}

	// Test Clone method
	clone := constraint.Clone()
	if clone == nil {
		t.Error("Clone returned nil")
	}
	if clone == constraint {
		t.Error("Clone returned same instance (should be different)")
	}
	if clone.Type() != constraint.Type() {
		t.Error("Clone has different type than original")
	}
}

// Test forward propagation: x → |x|
func TestAbsolute_ForwardPropagation(t *testing.T) {
	model := NewModel()

	// Test with offset=10: domain [5,15] represents actual values [-5,5]
	// Expected absolute values: |{-5,-4,...,4,5}| = {0,1,2,3,4,5}
	// BitSetDomain encoding: {0,1,2,3,4,5} → {1,1,2,3,4,5} → {1,2,3,4,5}
	x := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(5, 15)))
	absValue := model.NewVariable(NewBitSetDomainFromValues(10, rangeValues(1, 9)))

	constraint, err := NewAbsolute(x, 10, absValue)
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

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check that absValue domain was properly constrained
	finalAbsValue := solver.GetDomain(nil, absValue.ID())
	actualAbsValues := domainToSlice(finalAbsValue)

	// Should contain values {1,2,3,4,5} (representing |{-5,-4,...,4,5}|)
	expectedAbsValues := []int{1, 2, 3, 4, 5}
	if !isSubset(expectedAbsValues, actualAbsValues) {
		t.Errorf("Expected absValue domain to contain %v, got %v", expectedAbsValues, actualAbsValues)
	}
}

// Test backward propagation: |x| → x
func TestAbsolute_BackwardPropagation(t *testing.T) {
	model := NewModel()

	// Test with offset=10, absValue={3,4}, should get x values representing {-4,-3,3,4}
	// In offset encoding: {-4,-3,3,4} → {6,7,13,14}
	x := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	absValue := model.NewVariable(NewBitSetDomainFromValues(5, []int{3, 4}))

	constraint, err := NewAbsolute(x, 10, absValue)
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

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check that x domain was properly constrained
	finalX := solver.GetDomain(nil, x.ID())
	actualXValues := domainToSlice(finalX)

	// Should contain values {6,7,13,14} (representing {-4,-3,3,4})
	expectedXValues := []int{6, 7, 13, 14}

	// Check that all expected values are present in the actual domain
	for _, expected := range expectedXValues {
		if !containsInt(actualXValues, expected) {
			t.Errorf("Expected x domain to contain %d, got %v", expected, actualXValues)
		}
	}
}

// Test bidirectional propagation
func TestAbsolute_BidirectionalPropagation(t *testing.T) {
	model := NewModel()

	// Test with offset=20: x represents [-10,10], absValue represents [0,10]
	// Start with broad domains and let constraint narrow them
	x := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(10, 30)))       // [-10,10]
	absValue := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) // [0,9] (1 represents 0)

	constraint, err := NewAbsolute(x, 20, absValue)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 10) // Get multiple solutions
	if err != nil {
		t.Fatalf("Unexpected error during solving: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Verify constraint satisfaction in solutions
	for i, solution := range solutions {
		if len(solution) < 2 {
			t.Errorf("Solution %d has wrong length: %v", i, solution)
			continue
		}

		xVal := solution[x.ID()]
		absVal := solution[absValue.ID()]

		// Decode values
		actualX := xVal - 20
		expectedAbs := actualX
		if actualX < 0 {
			expectedAbs = -actualX
		}

		// Handle BitSetDomain encoding of 0
		if expectedAbs == 0 && absVal != 1 {
			t.Errorf("Solution %d: |%d| = %d, but got absValue %d", i, actualX, expectedAbs, absVal)
		} else if expectedAbs > 0 && absVal != expectedAbs {
			t.Errorf("Solution %d: |%d| = %d, but got absValue %d", i, actualX, expectedAbs, absVal)
		}
	}
}

// Test absolute value with zero
func TestAbsolute_Zero(t *testing.T) {
	model := NewModel()

	// Test |0| = 0, using offset=10: 0 → 10, |0|=0 → 1
	x := model.NewVariable(NewBitSetDomainFromValues(11, []int{10}))      // represents 0
	absValue := model.NewVariable(NewBitSetDomainFromValues(2, []int{1})) // represents 0

	constraint, err := NewAbsolute(x, 10, absValue)
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

	if len(solutions) != 1 {
		t.Errorf("Expected exactly 1 solution for |0|=0, got %d", len(solutions))
		return
	}

	solution := solutions[0]
	if solution[x.ID()] != 10 || solution[absValue.ID()] != 1 {
		t.Errorf("Expected solution [10,1] for |0|=0, got [%d,%d]", solution[x.ID()], solution[absValue.ID()])
	}
}

// Test self-reference constraint: |x| = x
func TestAbsolute_SelfReference(t *testing.T) {
	tests := []struct {
		name           string
		domain         []int
		offset         int
		expectedValues []int
		expectError    bool
		description    string
	}{
		{
			name:           "valid_self_reference",
			domain:         []int{10, 11, 12, 13, 14, 15}, // represents [0,1,2,3,4,5]
			offset:         10,
			expectedValues: []int{10, 11, 12, 13, 14, 15}, // All non-negative
			expectError:    false,
			description:    "|x| = x valid when x >= 0",
		},
		{
			name:           "partial_valid_self_reference",
			domain:         []int{8, 9, 10, 11, 12}, // represents [-2,-1,0,1,2]
			offset:         10,
			expectedValues: []int{10, 11, 12}, // Only non-negative [0,1,2]
			expectError:    false,
			description:    "|x| = x filters to non-negative values",
		},
		{
			name:           "no_valid_self_reference",
			domain:         []int{5, 6, 7, 8}, // represents [-5,-4,-3,-2]
			offset:         10,
			expectedValues: []int{}, // No non-negative values
			expectError:    true,
			description:    "|x| = x invalid when all x < 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := NewModel()
			x := model.NewVariable(NewBitSetDomainFromValues(maxValue(test.domain)+1, test.domain))

			constraint, err := NewAbsolute(x, test.offset, x) // Self-reference
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

// Test computeAbsolute function
func TestAbsolute_ComputeAbsolute(t *testing.T) {
	absolute := &Absolute{offset: 10}

	tests := []struct {
		input    int
		expected int
		desc     string
	}{
		{5, 5, "negative value -5 (encoded as 5)"},  // -5 → |−5| = 5
		{10, 1, "zero value 0 (encoded as 10)"},     // 0 → |0| = 0 → 1 (BitSetDomain)
		{15, 5, "positive value 5 (encoded as 15)"}, // 5 → |5| = 5
		{8, 2, "negative value -2 (encoded as 8)"},  // -2 → |−2| = 2
		{12, 2, "positive value 2 (encoded as 12)"}, // 2 → |2| = 2
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := absolute.computeAbsolute(test.input)
			if result != test.expected {
				t.Errorf("computeAbsolute(%d) = %d, expected %d (%s)", test.input, result, test.expected, test.desc)
			}
		})
	}
}

// Test constraint cloning
func TestAbsolute_Clone(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	absValue := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))

	original, err := NewAbsolute(x, 10, absValue)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	clone := original.Clone().(*Absolute)

	// Verify clone is independent but equivalent
	if clone == original {
		t.Error("Clone returned same instance")
	}

	if clone.x != original.x {
		t.Error("Clone has different x variable")
	}

	if clone.absValue != original.absValue {
		t.Error("Clone has different absValue variable")
	}

	if clone.offset != original.offset {
		t.Error("Clone has different offset")
	}

	if clone.Type() != original.Type() {
		t.Error("Clone has different type")
	}
}

// Test large offset values
func TestAbsolute_LargeOffset(t *testing.T) {
	model := NewModel()

	// Test with large offset=100, domain representing [-50,50]
	x := model.NewVariable(NewBitSetDomainFromValues(151, rangeValues(50, 150)))
	absValue := model.NewVariable(NewBitSetDomainFromValues(51, rangeValues(1, 50)))

	constraint, err := NewAbsolute(x, 100, absValue)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 5)
	if err != nil {
		t.Fatalf("Unexpected error during solving: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Expected at least one solution with large offset")
		return
	}

	// Verify a few solutions satisfy the constraint
	for i, solution := range solutions[:3] { // Check first 3 solutions
		xVal := solution[x.ID()]
		absVal := solution[absValue.ID()]

		actualX := xVal - 100
		expectedAbs := actualX
		if actualX < 0 {
			expectedAbs = -actualX
		}

		// Handle BitSetDomain encoding of 0
		if expectedAbs == 0 && absVal != 1 {
			t.Errorf("Solution %d: |%d| = %d, but got absValue %d", i, actualX, expectedAbs, absVal)
		} else if expectedAbs > 0 && absVal != expectedAbs {
			t.Errorf("Solution %d: |%d| = %d, but got absValue %d", i, actualX, expectedAbs, absVal)
		}
	}
}

// Helper functions for tests

func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func isSubset(subset, set []int) bool {
	setMap := make(map[int]bool)
	for _, v := range set {
		setMap[v] = true
	}
	for _, v := range subset {
		if !setMap[v] {
			return false
		}
	}
	return true
}
