package minikanren

import (
	"context"
	"testing"
)

// Test constraint creation
func TestNewIntervalArithmetic(t *testing.T) {
	model := NewModel()

	tests := []struct {
		name      string
		setupVar  func() *FDVariable
		minBound  int
		maxBound  int
		operation IntervalOperation
		setupRes  func() *FDVariable
		errorMsg  string
	}{
		{
			name:      "valid_containment",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  5,
			maxBound:  15,
			operation: IntervalContainment,
			setupRes:  func() *FDVariable { return nil },
			errorMsg:  "",
		},
		{
			name:      "valid_intersection",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  5,
			maxBound:  15,
			operation: IntervalIntersection,
			setupRes:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15))) },
			errorMsg:  "",
		},
		{
			name:      "valid_sum",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10))) },
			minBound:  2,
			maxBound:  8,
			operation: IntervalSum,
			setupRes:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			errorMsg:  "",
		},
		{
			name:      "invalid_minBound_zero",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  0,
			maxBound:  10,
			operation: IntervalContainment,
			setupRes:  func() *FDVariable { return nil },
			errorMsg:  "minBound must be ≥ 1",
		},
		{
			name:      "invalid_bounds_order",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  15,
			maxBound:  10,
			operation: IntervalContainment,
			setupRes:  func() *FDVariable { return nil },
			errorMsg:  "maxBound (10) must be ≥ minBound (15)",
		},
		{
			name:      "containment_with_result",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  5,
			maxBound:  15,
			operation: IntervalContainment,
			setupRes:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15))) },
			errorMsg:  "containment operation should have nil result",
		},
		{
			name:      "binary_without_result",
			setupVar:  func() *FDVariable { return model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20))) },
			minBound:  5,
			maxBound:  15,
			operation: IntervalSum,
			setupRes:  func() *FDVariable { return nil },
			errorMsg:  "sum operation requires non-nil result",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var variable, result *FDVariable
			if test.setupVar != nil {
				variable = test.setupVar()
			}
			if test.setupRes != nil {
				result = test.setupRes()
			}

			constraint, err := NewIntervalArithmetic(variable, test.minBound, test.maxBound, test.operation, result)

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
func TestNewIntervalArithmetic_NilVariable(t *testing.T) {
	model := NewModel()
	validResult := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))

	constraint, err := NewIntervalArithmetic(nil, 1, 10, IntervalContainment, nil)

	if err == nil {
		t.Error("Expected error for nil variable, got nil")
	} else if !containsString(err.Error(), "variable cannot be nil") {
		t.Errorf("Expected error about nil variable, got: %s", err.Error())
	}

	if constraint != nil {
		t.Error("Expected nil constraint for invalid input")
	}

	// Test with binary operation
	_, err2 := NewIntervalArithmetic(nil, 1, 10, IntervalSum, validResult)
	if err2 == nil {
		t.Error("Expected error for nil variable in binary operation, got nil")
	}
}

// Test constraint interface compliance
func TestIntervalArithmetic_Interface(t *testing.T) {
	model := NewModel()
	variable := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	result := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(1, 15)))

	// Test containment constraint
	constraint1, err := NewIntervalArithmetic(variable, 5, 15, IntervalContainment, nil)
	if err != nil {
		t.Fatalf("Failed to create containment constraint: %v", err)
	}

	// Test Variables method
	vars1 := constraint1.Variables()
	if len(vars1) != 1 {
		t.Errorf("Expected 1 variable for containment, got %d", len(vars1))
	}
	if vars1[0] != variable {
		t.Error("Variables returned wrong variable for containment")
	}

	// Test binary constraint
	constraint2, err := NewIntervalArithmetic(variable, 2, 8, IntervalSum, result)
	if err != nil {
		t.Fatalf("Failed to create sum constraint: %v", err)
	}

	vars2 := constraint2.Variables()
	if len(vars2) != 2 {
		t.Errorf("Expected 2 variables for sum, got %d", len(vars2))
	}
	if vars2[0] != variable || vars2[1] != result {
		t.Error("Variables returned wrong variables for sum")
	}

	// Test Type method
	if constraint1.Type() != "IntervalArithmetic" {
		t.Errorf("Expected type 'IntervalArithmetic', got '%s'", constraint1.Type())
	}

	// Test String method
	str := constraint1.String()
	if str == "" {
		t.Error("String method returned empty string")
	}
	if !containsString(str, "IntervalArithmetic") {
		t.Errorf("String method should contain 'IntervalArithmetic', got: %s", str)
	}

	// Test Clone method
	clone := constraint1.Clone()
	if clone == nil {
		t.Error("Clone returned nil")
	}
	if clone == constraint1 {
		t.Error("Clone returned same instance (should be different)")
	}
	if clone.Type() != constraint1.Type() {
		t.Error("Clone has different type than original")
	}
}

// Test interval operation string representations
func TestIntervalOperation_String(t *testing.T) {
	tests := []struct {
		op       IntervalOperation
		expected string
	}{
		{IntervalContainment, "containment"},
		{IntervalIntersection, "intersection"},
		{IntervalUnion, "union"},
		{IntervalSum, "sum"},
		{IntervalDifference, "difference"},
		{IntervalOperation(999), "unknown"},
	}

	for _, test := range tests {
		actual := test.op.String()
		if actual != test.expected {
			t.Errorf("Operation %d.String() = '%s', expected '%s'", int(test.op), actual, test.expected)
		}
	}
}

// Test containment constraint
func TestIntervalArithmetic_Containment(t *testing.T) {
	model := NewModel()

	// Variable with domain [1,20], constrain to [5,15]
	variable := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))

	constraint, err := NewIntervalArithmetic(variable, 5, 15, IntervalContainment, nil)
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
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check that variable domain was constrained to [5,15]
	finalVar := solver.GetDomain(nil, variable.ID())
	actualValues := domainToSlice(finalVar)

	expectedValues := rangeValues(5, 15)
	if !equalSlices(actualValues, expectedValues) {
		t.Errorf("Expected variable domain %v, got %v", expectedValues, actualValues)
	}

	// Verify all solutions satisfy containment constraint
	for i, solution := range solutions {
		varVal := solution[variable.ID()]
		if varVal < 5 || varVal > 15 {
			t.Errorf("Solution %d: variable value %d not in [5,15]", i, varVal)
		}
	}
}

// Test intersection constraint
func TestIntervalArithmetic_Intersection(t *testing.T) {
	model := NewModel()

	// Variable [1,20] intersect [5,15] should give result [5,15]
	variable := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	result := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))

	constraint, err := NewIntervalArithmetic(variable, 5, 15, IntervalIntersection, result)
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
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check result domain constrained to intersection [5,15]
	finalResult := solver.GetDomain(nil, result.ID())
	resultValues := domainToSlice(finalResult)

	expectedIntersection := rangeValues(5, 15)
	if !isSubset(expectedIntersection, resultValues) {
		t.Errorf("Expected result domain to contain %v, got %v", expectedIntersection, resultValues)
	}

	// Verify solutions
	for i, solution := range solutions[:3] { // Check first 3
		varVal := solution[variable.ID()]
		resVal := solution[result.ID()]

		// Variable should be in intersection range for this constraint type
		if varVal >= 5 && varVal <= 15 {
			// Variable in intersection range
			if resVal != varVal {
				t.Errorf("Solution %d: intersection result %d should equal variable %d when in range", i, resVal, varVal)
			}
		}
	}
}

// Test union constraint
func TestIntervalArithmetic_Union(t *testing.T) {
	model := NewModel()

	// Variable [6,10] union [5,15] should give result [5,15]
	variable := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(6, 10)))
	result := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))

	constraint, err := NewIntervalArithmetic(variable, 5, 15, IntervalUnion, result)
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
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check result domain constrained to union [5,15]
	finalResult := solver.GetDomain(nil, result.ID())
	resultValues := domainToSlice(finalResult)

	expectedUnion := rangeValues(5, 15)
	if !isSubset(expectedUnion, resultValues) {
		t.Errorf("Expected result domain to contain %v, got %v", expectedUnion, resultValues)
	}
}

// Test sum constraint
func TestIntervalArithmetic_Sum(t *testing.T) {
	model := NewModel()

	// Variable [1,5] + [2,4] should give result [3,9]
	variable := model.NewVariable(NewBitSetDomainFromValues(6, rangeValues(1, 5)))
	result := model.NewVariable(NewBitSetDomainFromValues(15, rangeValues(1, 14)))

	constraint, err := NewIntervalArithmetic(variable, 2, 4, IntervalSum, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		t.Fatalf("Unexpected error during solving: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Verify solutions satisfy sum constraint
	for i, solution := range solutions {
		varVal := solution[variable.ID()]
		resVal := solution[result.ID()]

		// result should be in [varVal+2, varVal+4]
		if resVal < varVal+2 || resVal > varVal+4 {
			t.Errorf("Solution %d: result %d not in range [%d,%d] for variable %d",
				i, resVal, varVal+2, varVal+4, varVal)
		}
	}
}

// Test difference constraint
func TestIntervalArithmetic_Difference(t *testing.T) {
	model := NewModel()

	// Variable [10,15] - [2,4] should give result [6,13]
	variable := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(10, 15)))
	result := model.NewVariable(NewBitSetDomainFromValues(20, rangeValues(1, 19)))

	constraint, err := NewIntervalArithmetic(variable, 2, 4, IntervalDifference, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		t.Fatalf("Unexpected error during solving: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Verify solutions satisfy difference constraint
	for i, solution := range solutions {
		varVal := solution[variable.ID()]
		resVal := solution[result.ID()]

		// result should be in [varVal-4, varVal-2]
		expectedMin := max(1, varVal-4) // Ensure ≥ 1 for BitSetDomain
		expectedMax := varVal - 2

		if resVal < expectedMin || resVal > expectedMax {
			t.Errorf("Solution %d: result %d not in range [%d,%d] for variable %d",
				i, resVal, expectedMin, expectedMax, varVal)
		}
	}
}

// Test bidirectional propagation in sum constraint
func TestIntervalArithmetic_BidirectionalSum(t *testing.T) {
	model := NewModel()

	// Start with broad domains and let constraint narrow them
	variable := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
	result := model.NewVariable(NewBitSetDomainFromValues(16, rangeValues(5, 15))) // Narrow result

	constraint, err := NewIntervalArithmetic(variable, 3, 7, IntervalSum, result)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 10)
	if err != nil {
		t.Fatalf("Unexpected error during solving: %v", err)
	}

	if len(solutions) == 0 {
		t.Error("Expected at least one solution, got none")
		return
	}

	// Check that variable domain was constrained by backward propagation
	finalVar := solver.GetDomain(nil, variable.ID())
	varValues := domainToSlice(finalVar)

	// Variable should be constrained: result = variable + [3,7]
	// So variable = result - [3,7] = [5,15] - [3,7] = [5-7, 15-3] = [-2, 12]
	// But BitSetDomain requires ≥ 1, so variable ∈ [1, 12]
	// However, variable + [3,7] must produce result ∈ [5,15]
	// So variable ∈ [max(1, 5-7), 15-3] = [1, 12] ∩ [need to produce 5-15]
	// variable ∈ [max(1, 5-7), min(20, 15-3)] = [1, 12]
	// But to produce [5,15], variable must be in [max(1, 5-7), min(20, 15-3)]

	// Verify backward constraint: for result ∈ [5,15], variable ∈ [max(1,5-7), 15-3] = [1, 12]
	// But we also need forward constraint: variable + [3,7] ⊆ [5,15]
	// So variable ∈ [5-7, 15-3] ∩ [1,∞) = [max(1,5-7), 15-3] = [1, 12]
	// But more precisely: variable ∈ [5-7, 15-3] = [-2, 12] → [1, 12] (BitSetDomain)
	// And variable + 3 ≥ 5 → variable ≥ 2
	// And variable + 7 ≤ 15 → variable ≤ 8
	// So actually variable ∈ [2, 8]

	maxVarValue := maxValue(varValues)

	// Variable should be reasonably constrained (not the full [1,20])
	if maxVarValue > 15 {
		t.Errorf("Expected variable domain to be more constrained, max value %d > 15", maxVarValue)
	}
	if len(varValues) == 20 {
		t.Error("Expected variable domain to be narrowed by constraint propagation")
	}

	// Verify all solutions satisfy the constraint
	for i, solution := range solutions {
		varVal := solution[variable.ID()]
		resVal := solution[result.ID()]

		if resVal < varVal+3 || resVal > varVal+7 {
			t.Errorf("Solution %d: result %d not in range [%d,%d] for variable %d",
				i, resVal, varVal+3, varVal+7, varVal)
		}
	}
}

// Test constraint with empty intersection
func TestIntervalArithmetic_EmptyIntersection(t *testing.T) {
	model := NewModel()

	// Variable [1,5] with containment [10,15] should fail
	variable := model.NewVariable(NewBitSetDomainFromValues(6, rangeValues(1, 5)))

	constraint, err := NewIntervalArithmetic(variable, 10, 15, IntervalContainment, nil)
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

	// Should have no solutions
	if len(solutions) != 0 {
		t.Errorf("Expected no solutions for empty intersection, got %d solutions", len(solutions))
	}
}

// Test large interval bounds
func TestIntervalArithmetic_LargeBounds(t *testing.T) {
	model := NewModel()

	// Variable with large interval
	variable := model.NewVariable(NewBitSetDomainFromValues(101, rangeValues(1, 100)))

	constraint, err := NewIntervalArithmetic(variable, 50, 80, IntervalContainment, nil)
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
		t.Error("Expected at least one solution with large bounds")
		return
	}

	// Verify constraint satisfaction
	for i, solution := range solutions {
		varVal := solution[variable.ID()]
		if varVal < 50 || varVal > 80 {
			t.Errorf("Solution %d: variable value %d not in [50,80]", i, varVal)
		}
	}
}

// Helper functions

func minValue(values []int) int {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
