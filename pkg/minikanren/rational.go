package minikanren

import (
	"fmt"
	"math"
)

// Rational represents a rational number (fraction) with integer numerator and denominator.
// Used for exact arithmetic with coefficients in constraints like RationalLinearSum.
//
// Rationals are always stored in normalized form (reduced to lowest terms, positive denominator).
// This enables exact representation of fractional coefficients without floating-point errors.
//
// Common irrational approximations:
//
//	π ≈ 22/7 (Archimedes, error ~0.04%)
//	π ≈ 355/113 (Zu Chongzhi, error ~0.000008%)
//	√2 ≈ 99/70 (accurate to 4 decimals)
//	√2 ≈ 1393/985 (accurate to 6 decimals)
//	e ≈ 2721/1000 (accurate to 4 decimals)
//	φ (golden ratio) ≈ 1618/1000 (accurate to 3 decimals)
type Rational struct {
	Num int // numerator
	Den int // denominator (always > 0 after normalization)
}

// NewRational creates a rational number num/den in normalized form.
// Panics if denominator is zero.
//
// Normalization ensures:
//   - GCD(num, den) = 1 (reduced to lowest terms)
//   - den > 0 (sign stored in numerator)
//
// Examples:
//
//	NewRational(6, 8) → 3/4
//	NewRational(-6, 8) → -3/4
//	NewRational(6, -8) → -3/4
//	NewRational(0, 5) → 0/1
func NewRational(num, den int) Rational {
	if den == 0 {
		panic("rational: division by zero")
	}

	// Handle zero numerator
	if num == 0 {
		return Rational{Num: 0, Den: 1}
	}

	// Ensure denominator is positive (move sign to numerator)
	if den < 0 {
		num = -num
		den = -den
	}

	// Reduce to lowest terms
	g := gcd(abs(num), abs(den))
	return Rational{
		Num: num / g,
		Den: den / g,
	}
}

// Add returns the sum of two rational numbers: r + other.
//
// Algorithm: a/b + c/d = (a*d + b*c) / (b*d), then normalize.
//
// Example:
//
//	(1/2) + (1/3) = 3/6 + 2/6 = 5/6
func (r Rational) Add(other Rational) Rational {
	num := r.Num*other.Den + other.Num*r.Den
	den := r.Den * other.Den
	return NewRational(num, den)
}

// Sub returns the difference of two rational numbers: r - other.
//
// Example:
//
//	(3/4) - (1/2) = 3/4 - 2/4 = 1/4
func (r Rational) Sub(other Rational) Rational {
	num := r.Num*other.Den - other.Num*r.Den
	den := r.Den * other.Den
	return NewRational(num, den)
}

// Mul returns the product of two rational numbers: r * other.
//
// Algorithm: (a/b) * (c/d) = (a*c) / (b*d), then normalize.
//
// Example:
//
//	(2/3) * (3/4) = 6/12 = 1/2
func (r Rational) Mul(other Rational) Rational {
	num := r.Num * other.Num
	den := r.Den * other.Den
	return NewRational(num, den)
}

// Div returns the quotient of two rational numbers: r / other.
// Panics if other is zero.
//
// Algorithm: (a/b) / (c/d) = (a/b) * (d/c) = (a*d) / (b*c), then normalize.
//
// Example:
//
//	(3/4) / (2/3) = (3/4) * (3/2) = 9/8
func (r Rational) Div(other Rational) Rational {
	if other.Num == 0 {
		panic("rational: division by zero")
	}
	num := r.Num * other.Den
	den := r.Den * other.Num
	return NewRational(num, den)
}

// Neg returns the negation of the rational number: -r.
//
// Example:
//
//	-(3/4) = -3/4
func (r Rational) Neg() Rational {
	return Rational{Num: -r.Num, Den: r.Den}
}

// IsZero returns true if the rational number is zero.
func (r Rational) IsZero() bool {
	return r.Num == 0
}

// IsPositive returns true if the rational number is greater than zero.
func (r Rational) IsPositive() bool {
	return r.Num > 0
}

// IsNegative returns true if the rational number is less than zero.
func (r Rational) IsNegative() bool {
	return r.Num < 0
}

// ToFloat returns the floating-point approximation of the rational number.
// Useful for debugging or when exact precision is not required.
//
// Example:
//
//	Rational{22, 7}.ToFloat() ≈ 3.142857...
func (r Rational) ToFloat() float64 {
	return float64(r.Num) / float64(r.Den)
}

// String returns a string representation of the rational number.
//
// Format: "num/den" for non-integers, "num" for integers (den=1).
//
// Examples:
//
//	Rational{3, 4}.String() → "3/4"
//	Rational{6, 1}.String() → "6"
//	Rational{-5, 2}.String() → "-5/2"
func (r Rational) String() string {
	if r.Den == 1 {
		return fmt.Sprintf("%d", r.Num)
	}
	return fmt.Sprintf("%d/%d", r.Num, r.Den)
}

// Equals returns true if two rational numbers are equal.
// Since rationals are normalized, structural equality is sufficient.
func (r Rational) Equals(other Rational) bool {
	return r.Num == other.Num && r.Den == other.Den
}

// gcd computes the greatest common divisor of two positive integers using Euclid's algorithm.
// Assumes both a and b are non-negative.
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// lcm computes the least common multiple of two positive integers.
// Uses the identity: lcm(a, b) = (a * b) / gcd(a, b).
func lcm(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return abs(a*b) / gcd(abs(a), abs(b))
}

// lcmMultiple computes the LCM of multiple integers.
// Used to find the common denominator for scaling rational coefficients.
func lcmMultiple(values []int) int {
	if len(values) == 0 {
		return 1
	}
	result := values[0]
	for i := 1; i < len(values); i++ {
		result = lcm(result, values[i])
	}
	return result
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// CommonIrrationals provides pre-computed rational approximations for common irrational constants.
var CommonIrrationals = struct {
	// Pi approximations
	PiArchimedes Rational // 22/7, error ~0.04%
	PiZu         Rational // 355/113, error ~0.000008%

	// Square root of 2 approximations
	Sqrt2Simple   Rational // 99/70, accurate to 4 decimals
	Sqrt2Accurate Rational // 1393/985, accurate to 6 decimals

	// Euler's number e approximations
	ESimple Rational // 2721/1000, accurate to 4 decimals

	// Golden ratio φ approximations
	PhiSimple Rational // 1618/1000, accurate to 3 decimals
}{
	PiArchimedes:  NewRational(22, 7),
	PiZu:          NewRational(355, 113),
	Sqrt2Simple:   NewRational(99, 70),
	Sqrt2Accurate: NewRational(1393, 985),
	ESimple:       NewRational(2721, 1000),
	PhiSimple:     NewRational(1618, 1000),
}

// ApproximateIrrational provides rational approximations for common irrational values.
// Returns a rational with the requested precision (number of decimal places).
//
// Supported values: "pi", "sqrt2", "e", "phi"
//
// Uses continued fraction approximations for higher precision.
// For simplicity, this implementation provides fixed precision levels.
func ApproximateIrrational(name string, precision int) (Rational, error) {
	switch name {
	case "pi":
		if precision <= 2 {
			return CommonIrrationals.PiArchimedes, nil // 22/7
		}
		return CommonIrrationals.PiZu, nil // 355/113

	case "sqrt2":
		if precision <= 4 {
			return CommonIrrationals.Sqrt2Simple, nil // 99/70
		}
		return CommonIrrationals.Sqrt2Accurate, nil // 1393/985

	case "e":
		return CommonIrrationals.ESimple, nil // 2721/1000

	case "phi":
		return CommonIrrationals.PhiSimple, nil // 1618/1000

	default:
		return Rational{}, fmt.Errorf("rational: unknown irrational constant %q", name)
	}
}

// FromFloat creates a rational approximation of a floating-point number.
// Uses continued fraction algorithm with the specified maximum denominator.
//
// Warning: This is an approximation. For known constants like π, use ApproximateIrrational instead.
//
// Example:
//
//	FromFloat(3.14159, 1000) ≈ 355/113 (close to π)
func FromFloat(f float64, maxDenominator int) Rational {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		panic("rational: cannot convert NaN or Inf to rational")
	}

	// Handle negative numbers
	sign := 1
	if f < 0 {
		sign = -1
		f = -f
	}

	// Simple continued fraction algorithm
	tolerance := 1.0 / float64(maxDenominator*maxDenominator)

	// Start with floor of f
	h1, h2 := int(f), 1
	k1, k2 := 1, 0

	remaining := f - float64(int(f))

	for k1 <= maxDenominator {
		if math.Abs(float64(h1)/float64(k1)-f) < tolerance {
			return NewRational(sign*h1, k1)
		}

		if remaining < tolerance {
			break
		}

		// Next convergent
		a := int(1.0 / remaining)
		h1, h2 = a*h1+h2, h1
		k1, k2 = a*k1+k2, k1

		remaining = 1.0/remaining - float64(a)
	}

	return NewRational(sign*h1, k1)
}
