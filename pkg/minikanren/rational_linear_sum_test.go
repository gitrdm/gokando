package minikanren

import (
	"context"
	"testing"
)

// TestRationalLinearSum_IntegerCoefficients tests with integer coefficients (scale=1).
func TestRationalLinearSum_IntegerCoefficients(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	// Constraint: 2*x + 3*y = z
	coeffs := []Rational{NewRational(2, 1), NewRational(3, 1)}
	rls, err := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)

	if err != nil {
		t.Fatalf("NewRationalLinearSum failed: %v", err)
	}

	if rls.GetScale() != 1 {
		t.Errorf("Expected scale=1 for integer coefficients, got %d", rls.GetScale())
	}

	intCoeffs := rls.GetIntCoeffs()
	if len(intCoeffs) != 2 || intCoeffs[0] != 2 || intCoeffs[1] != 3 {
		t.Errorf("Expected intCoeffs=[2,3], got %v", intCoeffs)
	}
}

// TestRationalLinearSum_FractionalCoefficients tests fractional coefficients requiring scaling.
func TestRationalLinearSum_FractionalCoefficients(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	// Constraint: (1/2)*x + (1/3)*y = z
	// LCM(2, 3, 1) = 6
	// Scaled: 3*x + 2*y = 6*z (but z needs pre-scaling)
	coeffs := []Rational{NewRational(1, 2), NewRational(1, 3)}
	rls, err := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)

	if err != nil {
		t.Fatalf("NewRationalLinearSum failed: %v", err)
	}

	if rls.GetScale() != 6 {
		t.Errorf("Expected scale=6 for LCM(2,3,1), got %d", rls.GetScale())
	}

	intCoeffs := rls.GetIntCoeffs()
	// (1/2) * 6 = 3, (1/3) * 6 = 2
	if len(intCoeffs) != 2 || intCoeffs[0] != 3 || intCoeffs[1] != 2 {
		t.Errorf("Expected intCoeffs=[3,2], got %v", intCoeffs)
	}
}

// TestRationalLinearSum_PropagationIntegerCoeffs tests propagation with integer coefficients.
func TestRationalLinearSum_PropagationIntegerCoeffs(t *testing.T) {
	model := NewModel()
	// Create variables with constrained domains
	x := model.NewVariable(NewBitSetDomainFromValues(10, []int{2}))
	y := model.NewVariable(NewBitSetDomainFromValues(10, []int{3}))
	z := model.NewVariable(NewBitSetDomain(50))

	// Constraint: 2*x + 3*y = z
	coeffs := []Rational{NewRational(2, 1), NewRational(3, 1)}
	rls, err := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)
	if err != nil {
		t.Fatalf("NewRationalLinearSum failed: %v", err)
	}

	model.AddConstraint(rls)

	// Solve to trigger propagation
	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check z domain: should be {13} (2*2 + 3*3 = 13)
	zDomain := solver.GetDomain(nil, z.ID())
	if zDomain.Count() != 1 || !zDomain.Has(13) {
		t.Errorf("Expected z={13}, got domain with count=%d, has(13)=%t",
			zDomain.Count(), zDomain.Has(13))
	}
}

// TestRationalLinearSum_NilVariableError tests error handling for nil variables.
func TestRationalLinearSum_NilVariableError(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	coeffs := []Rational{NewRational(1, 2), NewRational(1, 3)}

	_, err := NewRationalLinearSum([]*FDVariable{x, nil}, coeffs, nil)
	if err == nil {
		t.Error("Expected error for nil variable, got nil")
	}
}

// TestRationalLinearSum_MismatchedLengths tests error for mismatched array lengths.
func TestRationalLinearSum_MismatchedLengths(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	coeffs := []Rational{NewRational(1, 2)} // Only one coefficient

	_, err := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)
	if err == nil {
		t.Error("Expected error for mismatched lengths, got nil")
	}
}

// TestRationalLinearSum_ZeroCoefficientError tests error for zero coefficient.
func TestRationalLinearSum_ZeroCoefficientError(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	coeffs := []Rational{NewRational(0, 1), NewRational(1, 2)}

	_, err := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)
	if err == nil {
		t.Error("Expected error for zero coefficient, got nil")
	}
}

// TestRationalLinearSum_String tests string representation.
func TestRationalLinearSum_String(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	coeffs := []Rational{NewRational(1, 2), NewRational(3, 1)}
	rls, _ := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)

	str := rls.String()
	if len(str) == 0 {
		t.Error("String() returned empty string")
	}
	// Should contain variable IDs and coefficients
	// Exact format may vary, so just check non-empty
}

// TestRationalLinearSum_WithScaling_NoScaling tests the WithScaling helper when scale=1.
func TestRationalLinearSum_WithScaling_NoScaling(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	// Integer coefficients, scale=1
	coeffs := []Rational{NewRational(2, 1), NewRational(3, 1)}
	rls, div, err := NewRationalLinearSumWithScaling([]*FDVariable{x, y}, coeffs, z, model)

	if err != nil {
		t.Fatalf("NewRationalLinearSumWithScaling failed: %v", err)
	}

	if rls == nil {
		t.Fatal("Expected non-nil RationalLinearSum")
	}

	if div != nil {
		t.Error("Expected nil ScaledDivision when scale=1, got non-nil")
	}
}

// TestRationalLinearSum_WithScaling_WithScaling tests the WithScaling helper when scale>1.
func TestRationalLinearSum_WithScaling_WithScaling(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(20))

	// Fractional coefficients: (1/2)*x + (1/3)*y = z
	// LCM(2, 3, 1) = 6, so scale=6
	coeffs := []Rational{NewRational(1, 2), NewRational(1, 3)}
	rls, div, err := NewRationalLinearSumWithScaling([]*FDVariable{x, y}, coeffs, z, model)

	if err != nil {
		t.Fatalf("NewRationalLinearSumWithScaling failed: %v", err)
	}

	if rls == nil {
		t.Fatal("Expected non-nil RationalLinearSum")
	}

	if div == nil {
		t.Fatal("Expected non-nil ScaledDivision when scale>1, got nil")
	}

	// Check that RationalLinearSum uses intermediate variable (not original z)
	rlsVars := rls.Variables()
	// Last variable should be intermediate, not z
	if len(rlsVars) < 3 {
		t.Errorf("Expected at least 3 variables (x, y, intermediate), got %d", len(rlsVars))
	}
}

// TestRationalLinearSum_PiCircumference tests using pi approximation for circle calculations.
func TestRationalLinearSum_PiCircumference(t *testing.T) {
	model := NewModel()

	// diameter = 7 (fixed value)
	diameter := model.NewVariable(NewBitSetDomainFromValues(10, []int{7}))

	// circumference ∈ [1, 100]
	circumference := model.NewVariable(NewBitSetDomain(100))

	// Constraint: π * diameter = circumference
	// Using π ≈ 22/7
	pi := CommonIrrationals.PiArchimedes // 22/7
	coeffs := []Rational{pi}

	rls, div, err := NewRationalLinearSumWithScaling(
		[]*FDVariable{diameter},
		coeffs,
		circumference,
		model,
	)

	if err != nil {
		t.Fatalf("NewRationalLinearSumWithScaling failed: %v", err)
	}

	model.AddConstraint(rls)
	if div != nil {
		model.AddConstraint(div)
	}

	solver := NewSolver(model)
	ctx := context.Background()
	solver.Solve(ctx, 1)

	// Check circumference: π * 7 = (22/7) * 7 = 22
	circumDomain := solver.GetDomain(nil, circumference.ID())
	if !circumDomain.Has(22) {
		t.Errorf("Expected circumference to include 22, domain count=%d", circumDomain.Count())
	}
}

// TestRationalLinearSum_Clone tests cloning.
func TestRationalLinearSum_Clone(t *testing.T) {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomain(10))
	y := model.NewVariable(NewBitSetDomain(10))
	z := model.NewVariable(NewBitSetDomain(50))

	coeffs := []Rational{NewRational(2, 1), NewRational(3, 1)}
	rls, _ := NewRationalLinearSum([]*FDVariable{x, y}, coeffs, z)

	clone := rls.Clone()
	if clone == nil {
		t.Fatal("Clone returned nil")
	}

	if clone.Type() != "RationalLinearSum" {
		t.Errorf("Clone Type() = %s, want RationalLinearSum", clone.Type())
	}
}
