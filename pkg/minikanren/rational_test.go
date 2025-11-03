package minikanren

import (
	"math"
	"testing"
)

// TestRational_NewRational tests creation and normalization.
func TestRational_NewRational(t *testing.T) {
	tests := []struct {
		name     string
		num, den int
		wantNum  int
		wantDen  int
	}{
		{"simple fraction", 3, 4, 3, 4},
		{"reduces to lowest terms", 6, 8, 3, 4},
		{"negative numerator", -3, 4, -3, 4},
		{"negative denominator", 3, -4, -3, 4},
		{"both negative", -3, -4, 3, 4},
		{"zero numerator", 0, 5, 0, 1},
		{"integer (den=1)", 5, 1, 5, 1},
		{"already normalized", 7, 11, 7, 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRational(tt.num, tt.den)
			if r.Num != tt.wantNum || r.Den != tt.wantDen {
				t.Errorf("NewRational(%d, %d) = %d/%d, want %d/%d",
					tt.num, tt.den, r.Num, r.Den, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_NewRationalPanic tests that zero denominator panics.
func TestRational_NewRationalPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewRational(1, 0) did not panic")
		}
	}()
	NewRational(1, 0)
}

// TestRational_Add tests addition.
func TestRational_Add(t *testing.T) {
	tests := []struct {
		name             string
		r1, r2           Rational
		wantNum, wantDen int
	}{
		{"simple addition", NewRational(1, 2), NewRational(1, 3), 5, 6},
		{"same denominator", NewRational(1, 4), NewRational(2, 4), 3, 4},
		{"with negative", NewRational(3, 4), NewRational(-1, 2), 1, 4},
		{"zero", NewRational(3, 4), NewRational(0, 1), 3, 4},
		{"integers", NewRational(2, 1), NewRational(3, 1), 5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r1.Add(tt.r2)
			if result.Num != tt.wantNum || result.Den != tt.wantDen {
				t.Errorf("%s + %s = %s, want %d/%d",
					tt.r1, tt.r2, result, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_Sub tests subtraction.
func TestRational_Sub(t *testing.T) {
	tests := []struct {
		name             string
		r1, r2           Rational
		wantNum, wantDen int
	}{
		{"simple subtraction", NewRational(3, 4), NewRational(1, 2), 1, 4},
		{"result negative", NewRational(1, 2), NewRational(3, 4), -1, 4},
		{"zero result", NewRational(2, 3), NewRational(2, 3), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r1.Sub(tt.r2)
			if result.Num != tt.wantNum || result.Den != tt.wantDen {
				t.Errorf("%s - %s = %s, want %d/%d",
					tt.r1, tt.r2, result, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_Mul tests multiplication.
func TestRational_Mul(t *testing.T) {
	tests := []struct {
		name             string
		r1, r2           Rational
		wantNum, wantDen int
	}{
		{"simple multiplication", NewRational(2, 3), NewRational(3, 4), 1, 2},
		{"with integer", NewRational(2, 3), NewRational(6, 1), 4, 1},
		{"with negative", NewRational(2, 3), NewRational(-3, 4), -1, 2},
		{"zero", NewRational(2, 3), NewRational(0, 1), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r1.Mul(tt.r2)
			if result.Num != tt.wantNum || result.Den != tt.wantDen {
				t.Errorf("%s * %s = %s, want %d/%d",
					tt.r1, tt.r2, result, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_Div tests division.
func TestRational_Div(t *testing.T) {
	tests := []struct {
		name             string
		r1, r2           Rational
		wantNum, wantDen int
	}{
		{"simple division", NewRational(3, 4), NewRational(2, 3), 9, 8},
		{"divide by integer", NewRational(6, 1), NewRational(2, 1), 3, 1},
		{"with negative", NewRational(3, 4), NewRational(-2, 3), -9, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r1.Div(tt.r2)
			if result.Num != tt.wantNum || result.Den != tt.wantDen {
				t.Errorf("%s / %s = %s, want %d/%d",
					tt.r1, tt.r2, result, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_DivPanic tests that division by zero panics.
func TestRational_DivPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Div by zero did not panic")
		}
	}()
	r1 := NewRational(1, 2)
	r2 := NewRational(0, 1)
	r1.Div(r2)
}

// TestRational_Neg tests negation.
func TestRational_Neg(t *testing.T) {
	tests := []struct {
		name             string
		r                Rational
		wantNum, wantDen int
	}{
		{"positive", NewRational(3, 4), -3, 4},
		{"negative", NewRational(-3, 4), 3, 4},
		{"zero", NewRational(0, 1), 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.r.Neg()
			if result.Num != tt.wantNum || result.Den != tt.wantDen {
				t.Errorf("-(%s) = %s, want %d/%d",
					tt.r, result, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_Predicates tests IsZero, IsPositive, IsNegative.
func TestRational_Predicates(t *testing.T) {
	zero := NewRational(0, 1)
	positive := NewRational(3, 4)
	negative := NewRational(-3, 4)

	if !zero.IsZero() {
		t.Error("zero.IsZero() = false, want true")
	}
	if zero.IsPositive() || zero.IsNegative() {
		t.Error("zero should not be positive or negative")
	}

	if !positive.IsPositive() {
		t.Error("positive.IsPositive() = false, want true")
	}
	if positive.IsZero() || positive.IsNegative() {
		t.Error("positive should not be zero or negative")
	}

	if !negative.IsNegative() {
		t.Error("negative.IsNegative() = false, want true")
	}
	if negative.IsZero() || negative.IsPositive() {
		t.Error("negative should not be zero or positive")
	}
}

// TestRational_ToFloat tests conversion to float.
func TestRational_ToFloat(t *testing.T) {
	tests := []struct {
		name  string
		r     Rational
		want  float64
		delta float64
	}{
		{"simple fraction", NewRational(1, 2), 0.5, 0.0001},
		{"third", NewRational(1, 3), 0.333333, 0.0001},
		{"integer", NewRational(5, 1), 5.0, 0.0001},
		{"negative", NewRational(-3, 4), -0.75, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.ToFloat()
			if math.Abs(got-tt.want) > tt.delta {
				t.Errorf("%s.ToFloat() = %f, want %f", tt.r, got, tt.want)
			}
		})
	}
}

// TestRational_String tests string representation.
func TestRational_String(t *testing.T) {
	tests := []struct {
		name string
		r    Rational
		want string
	}{
		{"fraction", NewRational(3, 4), "3/4"},
		{"integer", NewRational(5, 1), "5"},
		{"negative fraction", NewRational(-3, 4), "-3/4"},
		{"zero", NewRational(0, 1), "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.String()
			if got != tt.want {
				t.Errorf("%s.String() = %q, want %q", tt.r, got, tt.want)
			}
		})
	}
}

// TestRational_Equals tests equality comparison.
func TestRational_Equals(t *testing.T) {
	r1 := NewRational(3, 4)
	r2 := NewRational(6, 8) // Same as 3/4
	r3 := NewRational(1, 2)

	if !r1.Equals(r2) {
		t.Error("3/4 should equal 6/8 (normalized)")
	}

	if r1.Equals(r3) {
		t.Error("3/4 should not equal 1/2")
	}
}

// TestRational_GCD tests the gcd function.
func TestRational_GCD(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{12, 8, 4},
		{7, 11, 1},
		{100, 50, 50},
		{0, 5, 5},
		{5, 0, 5},
	}

	for _, tt := range tests {
		got := gcd(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("gcd(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestRational_LCM tests the lcm function.
func TestRational_LCM(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{4, 6, 12},
		{3, 5, 15},
		{10, 15, 30},
		{7, 11, 77},
	}

	for _, tt := range tests {
		got := lcm(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("lcm(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestRational_LCMMultiple tests the lcmMultiple function.
func TestRational_LCMMultiple(t *testing.T) {
	tests := []struct {
		name   string
		values []int
		want   int
	}{
		{"two values", []int{4, 6}, 12},
		{"three values", []int{3, 5, 7}, 105},
		{"with common factors", []int{6, 8, 12}, 24},
		{"single value", []int{5}, 5},
		{"empty", []int{}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lcmMultiple(tt.values)
			if got != tt.want {
				t.Errorf("lcmMultiple(%v) = %d, want %d", tt.values, got, tt.want)
			}
		})
	}
}

// TestRational_CommonIrrationals tests the predefined constants.
func TestRational_CommonIrrationals(t *testing.T) {
	// Test that pi approximations are close
	piActual := 3.141592653589793

	piArch := CommonIrrationals.PiArchimedes.ToFloat()
	if math.Abs(piArch-piActual) > 0.01 {
		t.Errorf("PiArchimedes (22/7) = %f, too far from π = %f", piArch, piActual)
	}

	piZu := CommonIrrationals.PiZu.ToFloat()
	if math.Abs(piZu-piActual) > 0.00001 {
		t.Errorf("PiZu (355/113) = %f, too far from π = %f", piZu, piActual)
	}

	// Test sqrt(2) approximations
	sqrt2Actual := 1.414213562373095

	sqrt2Simple := CommonIrrationals.Sqrt2Simple.ToFloat()
	if math.Abs(sqrt2Simple-sqrt2Actual) > 0.0001 {
		t.Errorf("Sqrt2Simple (99/70) = %f, too far from √2 = %f", sqrt2Simple, sqrt2Actual)
	}

	sqrt2Acc := CommonIrrationals.Sqrt2Accurate.ToFloat()
	if math.Abs(sqrt2Acc-sqrt2Actual) > 0.000001 {
		t.Errorf("Sqrt2Accurate (1393/985) = %f, too far from √2 = %f", sqrt2Acc, sqrt2Actual)
	}
}

// TestRational_ApproximateIrrational tests irrational approximation.
func TestRational_ApproximateIrrational(t *testing.T) {
	tests := []struct {
		name      string
		constant  string
		precision int
		wantNum   int
		wantDen   int
	}{
		{"pi low precision", "pi", 2, 22, 7},
		{"pi high precision", "pi", 6, 355, 113},
		{"sqrt2 low", "sqrt2", 4, 99, 70},
		{"sqrt2 high", "sqrt2", 6, 1393, 985},
		{"e", "e", 4, 2721, 1000},
		{"phi", "phi", 3, 809, 500}, // 1618/1000 = 809/500 normalized
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ApproximateIrrational(tt.constant, tt.precision)
			if err != nil {
				t.Fatalf("ApproximateIrrational(%q, %d) error: %v", tt.constant, tt.precision, err)
			}
			if r.Num != tt.wantNum || r.Den != tt.wantDen {
				t.Errorf("ApproximateIrrational(%q, %d) = %s, want %d/%d",
					tt.constant, tt.precision, r, tt.wantNum, tt.wantDen)
			}
		})
	}
}

// TestRational_ApproximateIrrationalError tests unknown constant error.
func TestRational_ApproximateIrrationalError(t *testing.T) {
	_, err := ApproximateIrrational("unknown", 4)
	if err == nil {
		t.Error("ApproximateIrrational with unknown constant should return error")
	}
}

// TestRational_FromFloat tests float conversion.
func TestRational_FromFloat(t *testing.T) {
	tests := []struct {
		name      string
		f         float64
		maxDen    int
		checkFunc func(r Rational) bool
	}{
		{"simple 0.5", 0.5, 100, func(r Rational) bool { return r.Equals(NewRational(1, 2)) }},
		{"0.75", 0.75, 100, func(r Rational) bool { return r.Equals(NewRational(3, 4)) }},
		{"0.333...", 0.333333, 100, func(r Rational) bool { return r.Equals(NewRational(1, 3)) }},
		{"pi approximation", 3.14159, 1000, func(r Rational) bool {
			// Should be close to pi, within 0.01
			return math.Abs(r.ToFloat()-3.14159) < 0.01
		}},
		{"negative", -0.5, 100, func(r Rational) bool { return r.Equals(NewRational(-1, 2)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := FromFloat(tt.f, tt.maxDen)
			if !tt.checkFunc(r) {
				t.Errorf("FromFloat(%f, %d) = %s, check failed", tt.f, tt.maxDen, r)
			}
		})
	}
}

// TestRational_FromFloatPanic tests that NaN and Inf panic.
func TestRational_FromFloatPanic(t *testing.T) {
	tests := []struct {
		name string
		f    float64
	}{
		{"NaN", math.NaN()},
		{"Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("FromFloat(%f) did not panic", tt.f)
				}
			}()
			FromFloat(tt.f, 100)
		})
	}
}

// TestRational_ArithmeticProperties tests mathematical properties.
func TestRational_ArithmeticProperties(t *testing.T) {
	r1 := NewRational(2, 3)
	r2 := NewRational(3, 4)
	r3 := NewRational(5, 6)

	// Commutativity: a + b = b + a
	if !r1.Add(r2).Equals(r2.Add(r1)) {
		t.Error("Addition not commutative")
	}
	if !r1.Mul(r2).Equals(r2.Mul(r1)) {
		t.Error("Multiplication not commutative")
	}

	// Associativity: (a + b) + c = a + (b + c)
	if !r1.Add(r2).Add(r3).Equals(r1.Add(r2.Add(r3))) {
		t.Error("Addition not associative")
	}
	if !r1.Mul(r2).Mul(r3).Equals(r1.Mul(r2.Mul(r3))) {
		t.Error("Multiplication not associative")
	}

	// Identity: a + 0 = a
	zero := NewRational(0, 1)
	if !r1.Add(zero).Equals(r1) {
		t.Error("Addition identity failed")
	}

	// Identity: a * 1 = a
	one := NewRational(1, 1)
	if !r1.Mul(one).Equals(r1) {
		t.Error("Multiplication identity failed")
	}

	// Inverse: a + (-a) = 0
	if !r1.Add(r1.Neg()).Equals(zero) {
		t.Error("Additive inverse failed")
	}

	// Inverse: a * (1/a) = 1 (for non-zero a)
	if !r1.Mul(one.Div(r1)).Equals(one) {
		t.Error("Multiplicative inverse failed")
	}
}
