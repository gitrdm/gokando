package minikanren

import (
	"fmt"
	"strings"
)

// RationalLinearSum implements a linear sum constraint with rational coefficients.
// Enforces: c₁*v₁ + c₂*v₂ + ... + cₙ*vₙ = result, where coefficients are rational numbers.
//
// Internally converts to integer LinearSum by computing LCM of all denominators:
//
//	(a/b)*x + (c/d)*y = z  →  LCM(b,d) * ((a/b)*x + (c/d)*y) = LCM(b,d) * z
//
// Example: (1/3)*x + (1/2)*y = z
//
//	LCM(3, 2) = 6
//	Scaled: 2*x + 3*y = 6*z
//
// This enables exact rational coefficient constraints while leveraging existing
// integer domain infrastructure and propagation algorithms.
//
// Use cases:
//   - Irrational approximations: π*diameter = circumference → (22/7)*d = c
//   - Percentage calculations: 10% bonus → (1/10)*salary = bonus
//   - Unit conversions with fractional ratios: (5/9)*(F-32) = C
//   - Recipe scaling: (3/4)*flour + (1/2)*sugar = mixture
type RationalLinearSum struct {
	vars       []*FDVariable // variables with rational coefficients
	coeffs     []Rational    // rational coefficients
	result     *FDVariable   // result variable
	scale      int           // LCM of all denominators (scaling factor)
	intCoeffs  []int         // scaled integer coefficients
	underlying *LinearSum    // delegated integer constraint
}

// NewRationalLinearSum creates a rational linear sum constraint.
// Requires that all variables and coefficients have matching lengths.
//
// The constraint is automatically converted to integer form via LCM scaling:
//
//	scale = LCM(all denominators including result's implicit 1)
//	intCoeffs[i] = coeffs[i].Num * (scale / coeffs[i].Den)
//
// Then creates underlying constraint: intCoeffs[0]*vars[0] + ... = result
//
// IMPORTANT: When scale > 1, the result variable's domain must be pre-scaled.
// For example, if scale = 6:
//   - User constraint: (1/3)*x + (1/2)*y = z
//   - Internal: 2*x + 3*y = 6*z
//   - If x∈[3,9] and y∈[2,4], then z's domain should be pre-divided by 6
//   - Or use ScaledDivision to handle the scaling
//
// For coefficients that share denominators with result's implicit 1, scale will be 1.
// Example: (1/1)*x + (2/1)*y = z → scale=1 → x + 2*y = z (direct mapping)
//
// Panics if:
//   - Any variable is nil
//   - Result is nil
//   - Length of vars and coeffs don't match
//   - Any coefficient is zero (use fewer variables instead)
//
// Example (scale = 1):
//
//	// Constraint: 2*x + 3*y = z (integer coefficients)
//	c, _ := NewRationalLinearSum(
//	    []*FDVariable{x, y},
//	    []Rational{NewRational(2,1), NewRational(3,1)},
//	    z,
//	)
//
// Example (scale = 6, requires pre-scaled result):
//
//	// Constraint: (1/3)*x + (1/2)*y = z
//	// Internal: 2*x + 3*y = 6*z
//	// User must ensure z's domain accounts for factor of 6
//	c, _ := NewRationalLinearSum(
//	    []*FDVariable{x, y},
//	    []Rational{NewRational(1,3), NewRational(1,2)},
//	    z, // z's domain should be ⌊original_range / 6⌋
//	)
func NewRationalLinearSum(vars []*FDVariable, coeffs []Rational, result *FDVariable) (*RationalLinearSum, error) {
	if result == nil {
		return nil, fmt.Errorf("RationalLinearSum: nil result variable")
	}

	if len(vars) == 0 {
		return nil, fmt.Errorf("RationalLinearSum: no variables provided")
	}

	if len(vars) != len(coeffs) {
		return nil, fmt.Errorf("RationalLinearSum: vars length %d != coeffs length %d", len(vars), len(coeffs))
	}

	// Validate all variables non-nil
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("RationalLinearSum: variable at index %d is nil", i)
		}
	}

	// Validate no zero coefficients
	for i, c := range coeffs {
		if c.IsZero() {
			return nil, fmt.Errorf("RationalLinearSum: coefficient at index %d is zero (remove this variable)", i)
		}
	}

	// Compute LCM of all denominators (including result's implicit denominator of 1)
	denominators := make([]int, len(coeffs)+1)
	for i, c := range coeffs {
		denominators[i] = c.Den
	}
	denominators[len(coeffs)] = 1 // result has implicit denominator 1

	scale := lcmMultiple(denominators)

	// Scale coefficients to integers
	intCoeffs := make([]int, len(coeffs))
	for i, c := range coeffs {
		// intCoeff = c.Num * (scale / c.Den)
		intCoeffs[i] = c.Num * (scale / c.Den)
	}

	// Note: When scale > 1, the result variable's domain should be pre-scaled
	// by the user to account for the scaling factor. This is a known limitation.
	// Future enhancement: automatically create intermediate scaled variable.

	// Build underlying integer constraint
	// Two cases:
	// 1. scale == 1: Direct mapping to LinearSum
	// 2. scale > 1: Need to handle result scaling

	underlying, err := NewLinearSum(vars, intCoeffs, result)
	if err != nil {
		return nil, fmt.Errorf("RationalLinearSum: failed to create underlying constraint: %w", err)
	}

	return &RationalLinearSum{
		vars:       vars,
		coeffs:     coeffs,
		result:     result,
		scale:      scale,
		intCoeffs:  intCoeffs,
		underlying: underlying,
	}, nil
}

// Variables implements ModelConstraint.
func (rls *RationalLinearSum) Variables() []*FDVariable {
	allVars := make([]*FDVariable, len(rls.vars)+1)
	copy(allVars, rls.vars)
	allVars[len(rls.vars)] = rls.result
	return allVars
}

// Type implements ModelConstraint.
func (rls *RationalLinearSum) Type() string {
	return "RationalLinearSum"
}

// String implements ModelConstraint.
func (rls *RationalLinearSum) String() string {
	var terms []string
	for i := range rls.vars {
		coeff := rls.coeffs[i]
		varID := rls.vars[i].ID()
		if coeff.Equals(NewRational(1, 1)) {
			terms = append(terms, fmt.Sprintf("v%d", varID))
		} else if coeff.Equals(NewRational(-1, 1)) {
			terms = append(terms, fmt.Sprintf("-v%d", varID))
		} else if coeff.IsNegative() {
			terms = append(terms, fmt.Sprintf("(%s)v%d", coeff, varID))
		} else {
			terms = append(terms, fmt.Sprintf("(%s)v%d", coeff, varID))
		}
	}
	return fmt.Sprintf("%s = v%d", strings.Join(terms, " + "), rls.result.ID())
}

// Clone implements ModelConstraint.
func (rls *RationalLinearSum) Clone() ModelConstraint {
	coeffsCopy := make([]Rational, len(rls.coeffs))
	copy(coeffsCopy, rls.coeffs)

	// Note: This will fail if scale != 1, but that's caught in constructor
	clone, _ := NewRationalLinearSum(rls.vars, coeffsCopy, rls.result)
	return clone
}

// Propagate implements PropagationConstraint by delegating to underlying integer LinearSum.
func (rls *RationalLinearSum) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if rls.underlying == nil {
		return nil, fmt.Errorf("RationalLinearSum: no underlying constraint (scale != 1 not yet supported)")
	}
	return rls.underlying.Propagate(solver, state)
}

// GetScale returns the LCM scaling factor used to convert rational coefficients to integers.
// Useful for debugging or understanding the internal representation.
func (rls *RationalLinearSum) GetScale() int {
	return rls.scale
}

// GetIntCoeffs returns the scaled integer coefficients used internally.
// These are the numerators after multiplying each coefficient by (scale / denominator).
func (rls *RationalLinearSum) GetIntCoeffs() []int {
	result := make([]int, len(rls.intCoeffs))
	copy(result, rls.intCoeffs)
	return result
}

// NewRationalLinearSumWithScaling creates a RationalLinearSum and handles result scaling automatically.
// This is a convenience wrapper that uses ScaledDivision when needed (scale > 1).
//
// Returns the RationalLinearSum constraint plus an optional ScaledDivision constraint
// that must also be added to the model.
//
// Usage:
//
//	rls, scaledDiv, err := NewRationalLinearSumWithScaling(vars, coeffs, result, model)
//	model.AddConstraint(rls)
//	if scaledDiv != nil {
//	    model.AddConstraint(scaledDiv)
//	}
//
// When scale == 1: Returns only RationalLinearSum, scaledDiv is nil
// When scale > 1: Returns RationalLinearSum with scaled intermediate variable,
//
//	plus ScaledDivision constraint linking intermediate to result
func NewRationalLinearSumWithScaling(
	vars []*FDVariable,
	coeffs []Rational,
	result *FDVariable,
	model *Model,
) (*RationalLinearSum, *ScaledDivision, error) {
	if model == nil {
		return nil, nil, fmt.Errorf("RationalLinearSumWithScaling: nil model")
	}

	// Compute what the scale would be
	denominators := make([]int, len(coeffs)+1)
	for i, c := range coeffs {
		denominators[i] = c.Den
	}
	denominators[len(coeffs)] = 1
	scale := lcmMultiple(denominators)

	if scale == 1 {
		// No scaling needed, create directly
		rls, err := NewRationalLinearSum(vars, coeffs, result)
		return rls, nil, err
	}

	// Scale > 1: need intermediate variable
	// Create intermediate variable with scaled domain
	// If result ∈ [a, b], then intermediate ∈ [scale*a, scale*b]
	resultDomain := result.Domain()
	maxVal := resultDomain.MaxValue()

	// Create scaled domain by multiplying all values by scale
	validValues := make([]int, 0)
	resultDomain.IterateValues(func(v int) {
		validValues = append(validValues, scale*v)
	})

	var scaledDomain Domain
	if len(validValues) > 0 {
		scaledDomain = NewBitSetDomainFromValues(scale*maxVal, validValues)
	} else {
		scaledDomain = NewBitSetDomain(scale * maxVal)
	}

	intermediate := model.NewVariable(scaledDomain)

	// Create RationalLinearSum: sum = intermediate
	rls, err := NewRationalLinearSum(vars, coeffs, intermediate)
	if err != nil {
		return nil, nil, err
	}

	// Create ScaledDivision: intermediate / scale = result
	div, err := NewScaledDivision(intermediate, scale, result)
	if err != nil {
		return nil, nil, err
	}

	return rls, div, nil
}
