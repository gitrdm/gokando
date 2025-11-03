package minikanren

import (
	"context"
	"testing"
)

// TestScaledDivision_ForwardPropagation tests that dividend values correctly
// prune the quotient domain through forward propagation.
func TestScaledDivision_ForwardPropagation(t *testing.T) {
	model := NewModel()

	// dividend = {50, 60, 70}, divisor = 10
	// Expected quotient after propagation: {5, 6, 7}
	dividend := model.NewVariable(NewBitSetDomainFromValues(71, []int{50, 60, 70}))
	quotient := model.NewVariable(NewBitSetDomainFromValues(20, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	// Solve to trigger propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check quotient was pruned to {5, 6, 7}
	finalQuotient := solver.GetDomain(nil, quotient.ID())
	if finalQuotient.Count() != 3 {
		t.Errorf("Expected quotient domain size 3, got %d", finalQuotient.Count())
	}

	expected := map[int]bool{5: true, 6: true, 7: true}
	finalQuotient.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d in quotient domain", v)
		}
		delete(expected, v)
	})

	if len(expected) > 0 {
		t.Errorf("Missing values in quotient domain: %v", expected)
	}
}

// TestScaledDivision_BackwardPropagation tests that quotient values correctly
// prune the dividend domain through backward propagation.
func TestScaledDivision_BackwardPropagation(t *testing.T) {
	model := NewModel()

	// quotient = {5, 6}, divisor = 10
	// Expected dividend after propagation: {50-59, 60-69} ∩ original domain
	dividend := model.NewVariable(NewBitSetDomainFromValues(100, makeRange(45, 75)))
	quotient := model.NewVariable(NewBitSetDomainFromValues(10, []int{5, 6}))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check dividend was pruned to [50-69]
	finalDividend := solver.GetDomain(nil, dividend.ID())

	// Should have values 50-69 (20 values)
	if finalDividend.Count() != 20 {
		t.Errorf("Expected dividend domain size 20, got %d", finalDividend.Count())
	}

	if finalDividend.Min() != 50 {
		t.Errorf("Expected dividend min 50, got %d", finalDividend.Min())
	}
	if finalDividend.Max() != 69 {
		t.Errorf("Expected dividend max 69, got %d", finalDividend.Max())
	}
}

// TestScaledDivision_Bidirectional tests that both forward and backward
// propagation work together to reach a fixed point.
func TestScaledDivision_Bidirectional(t *testing.T) {
	model := NewModel()

	// dividend = {50, 51, 52, ..., 79}, quotient = {3, 4, 5, 6, 7, 8}
	// divisor = 10
	// Forward: quotient ⊇ {5, 6, 7}
	// Backward: dividend ⊇ {30-89}
	// Fixed point: dividend = {50-79}, quotient = {5, 6, 7}
	dividend := model.NewVariable(NewBitSetDomainFromValues(80, makeRange(50, 79)))
	quotient := model.NewVariable(NewBitSetDomainFromValues(9, makeRange(3, 8)))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalDividend := solver.GetDomain(nil, dividend.ID())
	finalQuotient := solver.GetDomain(nil, quotient.ID())

	// Dividend should remain unchanged
	if finalDividend.Count() != 30 {
		t.Errorf("Expected dividend count 30, got %d", finalDividend.Count())
	}

	// Quotient should be pruned to {5, 6, 7}
	if finalQuotient.Count() != 3 {
		t.Errorf("Expected quotient count 3, got %d", finalQuotient.Count())
	}
	if !finalQuotient.Has(5) || !finalQuotient.Has(6) || !finalQuotient.Has(7) {
		t.Error("Quotient should contain {5, 6, 7}")
	}
}

// TestScaledDivision_IntegerTruncation tests that integer division truncates
// correctly (floor division for positive numbers).
func TestScaledDivision_IntegerTruncation(t *testing.T) {
	model := NewModel()

	// dividend = {15, 16, 17, 18, 19}, divisor = 10
	// Expected quotient = {1} (all divide to 1)
	dividend := model.NewVariable(NewBitSetDomainFromValues(20, makeRange(15, 19)))
	quotient := model.NewVariable(NewBitSetDomainFromValues(5, makeRange(1, 3)))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalQuotient := solver.GetDomain(nil, quotient.ID())
	if !finalQuotient.IsSingleton() {
		t.Errorf("Expected singleton quotient, got count %d", finalQuotient.Count())
	}
	if finalQuotient.SingletonValue() != 1 {
		t.Errorf("Expected quotient = 1, got %d", finalQuotient.SingletonValue())
	}
}

// TestScaledDivision_EmptyDomainFailure tests that constraint detects
// inconsistency when domains become empty.
func TestScaledDivision_EmptyDomainFailure(t *testing.T) {
	model := NewModel()

	// dividend = {100}, quotient = {1}, divisor = 10
	// 100 / 10 = 10, not 1 → inconsistent
	dividend := model.NewVariable(NewBitSetDomainFromValues(101, []int{100}))
	quotient := model.NewVariable(NewBitSetDomainFromValues(2, []int{1}))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solutions, err := solver.Solve(ctx, 1)

	// Should fail - no solutions
	if err == nil && len(solutions) > 0 {
		t.Error("Expected failure due to inconsistent domains, but got solutions")
	}
}

// TestScaledDivision_SingletonPropagation tests that singleton values
// propagate correctly in both directions.
func TestScaledDivision_SingletonPropagation(t *testing.T) {
	model := NewModel()

	// dividend = {50}, divisor = 10
	// Should force quotient = {5}
	dividend := model.NewVariable(NewBitSetDomainFromValues(51, []int{50}))
	quotient := model.NewVariable(NewBitSetDomainFromValues(10, makeRange(1, 10)))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalQuotient := solver.GetDomain(nil, quotient.ID())
	if !finalQuotient.IsSingleton() {
		t.Errorf("Expected singleton quotient, got count %d", finalQuotient.Count())
	}
	if finalQuotient.SingletonValue() != 5 {
		t.Errorf("Expected quotient = 5, got %d", finalQuotient.SingletonValue())
	}
}

// TestScaledDivision_BackwardSingleton tests backward propagation from
// a singleton quotient.
func TestScaledDivision_BackwardSingleton(t *testing.T) {
	model := NewModel()

	// quotient = {7}, divisor = 10
	// Should constrain dividend to [70, 79]
	dividend := model.NewVariable(NewBitSetDomainFromValues(100, makeRange(65, 85)))
	quotient := model.NewVariable(NewBitSetDomainFromValues(8, []int{7}))

	constraint, err := NewScaledDivision(dividend, 10, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalDividend := solver.GetDomain(nil, dividend.ID())
	if finalDividend.Min() != 70 {
		t.Errorf("Expected min 70, got %d", finalDividend.Min())
	}
	if finalDividend.Max() != 79 {
		t.Errorf("Expected max 79, got %d", finalDividend.Max())
	}
	if finalDividend.Count() != 10 {
		t.Errorf("Expected 10 values, got %d", finalDividend.Count())
	}
}

// TestScaledDivision_LargeDivisor tests division with large divisors
// (smaller quotients).
func TestScaledDivision_LargeDivisor(t *testing.T) {
	model := NewModel()

	// dividend = {100, 200, 300}, divisor = 100
	// Expected quotient = {1, 2, 3}
	dividend := model.NewVariable(NewBitSetDomainFromValues(301, []int{100, 200, 300}))
	quotient := model.NewVariable(NewBitSetDomainFromValues(10, makeRange(1, 10)))

	constraint, err := NewScaledDivision(dividend, 100, quotient)
	if err != nil {
		t.Fatalf("Failed to create constraint: %v", err)
	}

	model.AddConstraint(constraint)

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	finalQuotient := solver.GetDomain(nil, quotient.ID())
	expected := map[int]bool{1: true, 2: true, 3: true}

	if finalQuotient.Count() != 3 {
		t.Errorf("Expected 3 values, got %d", finalQuotient.Count())
	}

	finalQuotient.IterateValues(func(v int) {
		if !expected[v] {
			t.Errorf("Unexpected value %d", v)
		}
		delete(expected, v)
	})

	if len(expected) > 0 {
		t.Errorf("Missing values: %v", expected)
	}
}

// TestNewScaledDivision_ErrorCases tests constructor validation.
func TestNewScaledDivision_ErrorCases(t *testing.T) {
	model := NewModel()
	validVar := model.NewVariable(NewBitSetDomain(10))

	tests := []struct {
		name      string
		dividend  *FDVariable
		divisor   int
		quotient  *FDVariable
		expectErr bool
	}{
		{"nil dividend", nil, 10, validVar, true},
		{"nil quotient", validVar, 10, nil, true},
		{"zero divisor", validVar, 0, validVar, true},
		{"negative divisor", validVar, -5, validVar, true},
		{"valid", validVar, 10, validVar, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScaledDivision(tt.dividend, tt.divisor, tt.quotient)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error=%v, got %v", tt.expectErr, err)
			}
		})
	}
}

// TestScaledDivision_Type tests the Type() method returns correct identifier.
func TestScaledDivision_Type(t *testing.T) {
	model := NewModel()
	dividend := model.NewVariable(NewBitSetDomain(10))
	quotient := model.NewVariable(NewBitSetDomain(10))

	constraint, _ := NewScaledDivision(dividend, 10, quotient)
	if constraint.Type() != "ScaledDivision" {
		t.Errorf("Expected type 'ScaledDivision', got '%s'", constraint.Type())
	}
}

// TestScaledDivision_Variables tests that Variables() returns correct list.
func TestScaledDivision_Variables(t *testing.T) {
	model := NewModel()
	dividend := model.NewVariable(NewBitSetDomain(10))
	quotient := model.NewVariable(NewBitSetDomain(10))

	constraint, _ := NewScaledDivision(dividend, 10, quotient)
	vars := constraint.Variables()

	if len(vars) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(vars))
	}
	if vars[0].ID() != dividend.ID() {
		t.Error("First variable should be dividend")
	}
	if vars[1].ID() != quotient.ID() {
		t.Error("Second variable should be quotient")
	}
}

// Helper function to create a range of integers [start, end] inclusive.
func makeRange(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}
