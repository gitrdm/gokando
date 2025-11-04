package minikanren

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test Pluso - Addition

func TestPluso_ForwardMode(t *testing.T) {
	t.Parallel()
	// 2 + 3 = ?
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(2), NewAtom(3), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

func TestPluso_BackwardModeX(t *testing.T) {
	t.Parallel()
	// ? + 3 = 8
	result := Run(1, func(q *Var) Goal {
		return Pluso(q, NewAtom(3), NewAtom(8))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

func TestPluso_BackwardModeY(t *testing.T) {
	t.Parallel()
	// 7 + ? = 10
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(7), q, NewAtom(10))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(3)) {
		t.Errorf("Expected 3, got %v", result[0])
	}
}

func TestPluso_Verification(t *testing.T) {
	t.Parallel()
	// 4 + 5 = 9 (should succeed)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Pluso(NewAtom(4), NewAtom(5), NewAtom(9)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestPluso_VerificationFail(t *testing.T) {
	t.Parallel()
	// 4 + 5 = 10 (should fail)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Pluso(NewAtom(4), NewAtom(5), NewAtom(10)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

func TestPluso_GenerateMode(t *testing.T) {
	t.Parallel()
	// ? + ? = 5 (generate pairs)
	result := Run(6, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Conj(
			Pluso(x, y, NewAtom(5)),
			Eq(q, NewPair(x, y)),
		)
	})

	if len(result) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(result))
	}

	// Verify all results are valid pairs that sum to 5
	seen := make(map[string]bool)
	for i, res := range result {
		pair := res.(*Pair)
		xVal, xOk := extractNumber(pair.Car())
		yVal, yOk := extractNumber(pair.Cdr())
		if !xOk || !yOk {
			t.Errorf("Result %d: non-numeric pair (%v, %v)", i, pair.Car(), pair.Cdr())
			continue
		}
		if xVal+yVal != 5 {
			t.Errorf("Result %d: (%d,%d) doesn't sum to 5", i, xVal, yVal)
		}
		key := fmt.Sprintf("%d,%d", xVal, yVal)
		if seen[key] {
			t.Errorf("Result %d: duplicate pair (%d,%d)", i, xVal, yVal)
		}
		seen[key] = true
	}
}

func TestPluso_NegativeNumbers(t *testing.T) {
	t.Parallel()
	// -3 + 7 = ?
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(-3), NewAtom(7), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(4)) {
		t.Errorf("Expected 4, got %v", result[0])
	}
}

func TestPluso_Zero(t *testing.T) {
	t.Parallel()
	// 0 + 5 = ?
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(0), NewAtom(5), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

// Test Minuso - Subtraction

func TestMinuso_ForwardMode(t *testing.T) {
	t.Parallel()
	// 10 - 3 = ?
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), NewAtom(3), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(7)) {
		t.Errorf("Expected 7, got %v", result[0])
	}
}

func TestMinuso_BackwardModeX(t *testing.T) {
	t.Parallel()
	// ? - 3 = 5
	result := Run(1, func(q *Var) Goal {
		return Minuso(q, NewAtom(3), NewAtom(5))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(8)) {
		t.Errorf("Expected 8, got %v", result[0])
	}
}

func TestMinuso_BackwardModeY(t *testing.T) {
	t.Parallel()
	// 10 - ? = 6
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), q, NewAtom(6))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(4)) {
		t.Errorf("Expected 4, got %v", result[0])
	}
}

func TestMinuso_NegativeResult(t *testing.T) {
	t.Parallel()
	// 3 - 7 = ?
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(3), NewAtom(7), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(-4)) {
		t.Errorf("Expected -4, got %v", result[0])
	}
}

// Test Timeso - Multiplication

func TestTimeso_ForwardMode(t *testing.T) {
	t.Parallel()
	// 4 * 5 = ?
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(4), NewAtom(5), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(20)) {
		t.Errorf("Expected 20, got %v", result[0])
	}
}

func TestTimeso_BackwardModeX(t *testing.T) {
	t.Parallel()
	// ? * 6 = 24
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(6), NewAtom(24))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(4)) {
		t.Errorf("Expected 4, got %v", result[0])
	}
}

func TestTimeso_BackwardModeY(t *testing.T) {
	t.Parallel()
	// 7 * ? = 35
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(7), q, NewAtom(35))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

func TestTimeso_ZeroMultiplication(t *testing.T) {
	t.Parallel()
	// 0 * 5 = ?
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(0), NewAtom(5), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(0)) {
		t.Errorf("Expected 0, got %v", result[0])
	}
}

func TestTimeso_ZeroBackward(t *testing.T) {
	t.Parallel()
	// 0 * ? = 0 (should succeed with any value for y)
	result := Run(1, func(q *Var) Goal {
		y := Fresh("y")
		return Conj(
			Timeso(NewAtom(0), y, NewAtom(0)),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
}

func TestTimeso_NotDivisible(t *testing.T) {
	t.Parallel()
	// ? * 3 = 10 (10 not divisible by 3, should fail)
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(3), NewAtom(10))
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

func TestTimeso_NegativeNumbers(t *testing.T) {
	t.Parallel()
	// -3 * 4 = ?
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(-3), NewAtom(4), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(-12)) {
		t.Errorf("Expected -12, got %v", result[0])
	}
}

// Test Divo - Division

func TestDivo_ForwardMode(t *testing.T) {
	t.Parallel()
	// 15 / 3 = ?
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(15), NewAtom(3), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

func TestDivo_IntegerDivision(t *testing.T) {
	t.Parallel()
	// 7 / 2 = ? (integer division)
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(7), NewAtom(2), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(3)) {
		t.Errorf("Expected 3, got %v", result[0])
	}
}

func TestDivo_DivisionByZero(t *testing.T) {
	t.Parallel()
	// 10 / 0 = ? (should fail)
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(10), NewAtom(0), q)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (division by zero), got %d", len(result))
	}
}

func TestDivo_BackwardModeY(t *testing.T) {
	t.Parallel()
	// 20 / ? = 4
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(20), q, NewAtom(4))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(5)) {
		t.Errorf("Expected 5, got %v", result[0])
	}
}

func TestDivo_BackwardModeX(t *testing.T) {
	t.Parallel()
	// ? / 5 = 3
	result := Run(1, func(q *Var) Goal {
		return Divo(q, NewAtom(5), NewAtom(3))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(15)) {
		t.Errorf("Expected 15, got %v", result[0])
	}
}

// Test Expo - Exponentiation

func TestExpo_ForwardMode(t *testing.T) {
	t.Parallel()
	// 2^10 = ?
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), NewAtom(10), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(1024)) {
		t.Errorf("Expected 1024, got %v", result[0])
	}
}

func TestExpo_ZeroExponent(t *testing.T) {
	t.Parallel()
	// 5^0 = ?
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(5), NewAtom(0), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(1)) {
		t.Errorf("Expected 1, got %v", result[0])
	}
}

func TestExpo_OneExponent(t *testing.T) {
	t.Parallel()
	// 7^1 = ?
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(7), NewAtom(1), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(7)) {
		t.Errorf("Expected 7, got %v", result[0])
	}
}

func TestExpo_NegativeExponent(t *testing.T) {
	t.Parallel()
	// 2^-3 = ? (should fail - negative exponents not supported)
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), NewAtom(-3), q)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (negative exponent), got %d", len(result))
	}
}

func TestExpo_Verification(t *testing.T) {
	t.Parallel()
	// 3^4 = 81 (should succeed)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Expo(NewAtom(3), NewAtom(4), NewAtom(81)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

// Test Logo - Logarithm

func TestLogo_ForwardMode(t *testing.T) {
	t.Parallel()
	// log2(1024) = ?
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), NewAtom(1024), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(10)) {
		t.Errorf("Expected 10, got %v", result[0])
	}
}

func TestLogo_Base10(t *testing.T) {
	t.Parallel()
	// log10(1000) = ?
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(10), NewAtom(1000), q)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(3)) {
		t.Errorf("Expected 3, got %v", result[0])
	}
}

func TestLogo_InvalidBase(t *testing.T) {
	t.Parallel()
	// log1(10) = ? (base 1 is invalid)
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(1), NewAtom(10), q)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (invalid base), got %d", len(result))
	}
}

func TestLogo_InvalidValue(t *testing.T) {
	t.Parallel()
	// log2(0) = ? (log of 0 is undefined)
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), NewAtom(0), q)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (log of 0), got %d", len(result))
	}
}

// Test Expo - Backward Modes

func TestExpo_BackwardModeBase(t *testing.T) {
	t.Parallel()
	// ?^3 = 8 → solve for base
	result := Run(1, func(q *Var) Goal {
		return Expo(q, NewAtom(3), NewAtom(8))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(2)) {
		t.Errorf("Expected 2, got %v", result[0])
	}
}

func TestExpo_BackwardModeExp(t *testing.T) {
	t.Parallel()
	// 2^? = 256 → solve for exponent
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), q, NewAtom(256))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(8)) {
		t.Errorf("Expected 8, got %v", result[0])
	}
}

func TestExpo_BackwardModeExpZeroCase(t *testing.T) {
	t.Parallel()
	// 5^? = 1 → exponent must be 0
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(5), q, NewAtom(1))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(0)) {
		t.Errorf("Expected 0, got %v", result[0])
	}
}

func TestExpo_BackwardModeNonInteger(t *testing.T) {
	t.Parallel()
	// ?^2 = 5 → no integer solution
	result := Run(1, func(q *Var) Goal {
		return Expo(q, NewAtom(2), NewAtom(5))
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (no integer root), got %d", len(result))
	}
}

// Test Logo - Backward Modes

func TestLogo_BackwardModeValue(t *testing.T) {
	t.Parallel()
	// log2(?) = 10 → value = 2^10 = 1024
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), q, NewAtom(10))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(1024)) {
		t.Errorf("Expected 1024, got %v", result[0])
	}
}

func TestLogo_BackwardModeBase(t *testing.T) {
	t.Parallel()
	// log?(8) = 3 → base^3 = 8 → base = 2
	result := Run(1, func(q *Var) Goal {
		return Logo(q, NewAtom(8), NewAtom(3))
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(2)) {
		t.Errorf("Expected 2, got %v", result[0])
	}
}

func TestLogo_BackwardModeBaseNonInteger(t *testing.T) {
	t.Parallel()
	// log?(7) = 2 → base^2 = 7 → no integer solution
	result := Run(1, func(q *Var) Goal {
		return Logo(q, NewAtom(7), NewAtom(2))
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (no integer base), got %d", len(result))
	}
}

// Test LessThano - Less Than

func TestLessThano_True(t *testing.T) {
	t.Parallel()
	// 3 < 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(3), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestLessThano_False(t *testing.T) {
	t.Parallel()
	// 5 < 3 (should fail)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(5), NewAtom(3)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

func TestLessThano_Equal(t *testing.T) {
	t.Parallel()
	// 5 < 5 (should fail)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results (equal values), got %d", len(result))
	}
}

func TestLessThano_NegativeNumbers(t *testing.T) {
	t.Parallel()
	// -5 < -3
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(-5), NewAtom(-3)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

// Test GreaterThano - Greater Than

func TestGreaterThano_True(t *testing.T) {
	t.Parallel()
	// 10 > 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterThano(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestGreaterThano_False(t *testing.T) {
	t.Parallel()
	// 3 > 7 (should fail)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterThano(NewAtom(3), NewAtom(7)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

// Test LessEqualo - Less Than or Equal

func TestLessEqualo_Less(t *testing.T) {
	t.Parallel()
	// 3 <= 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(3), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestLessEqualo_Equal(t *testing.T) {
	t.Parallel()
	// 5 <= 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestLessEqualo_Greater(t *testing.T) {
	t.Parallel()
	// 7 <= 5 (should fail)
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(7), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

// Test GreaterEqualo - Greater Than or Equal

func TestGreaterEqualo_Greater(t *testing.T) {
	t.Parallel()
	// 10 >= 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterEqualo(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

func TestGreaterEqualo_Equal(t *testing.T) {
	t.Parallel()
	// 5 >= 5
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterEqualo(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom("yes")) {
		t.Errorf("Expected 'yes', got %v", result[0])
	}
}

// Composition tests

func TestArithmetic_Composition(t *testing.T) {
	t.Parallel()
	// (x + 3) * 2 = 10, solve for x
	// This works by solving backward: 10/2 = 5, then 5-3 = 2
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Timeso(temp, NewAtom(2), NewAtom(10)), // temp = 5
			Pluso(q, NewAtom(3), temp),            // q + 3 = 5, q = 2
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(2)) {
		t.Errorf("Expected 2, got %v", result[0])
	}
}

func TestArithmetic_ChainedAddition(t *testing.T) {
	t.Parallel()
	// x + y = 5, y + z = 7, solve for x, y, z
	result := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")
		return Conj(
			Pluso(x, y, NewAtom(5)),
			Pluso(y, z, NewAtom(7)),
			Eq(x, NewAtom(2)),
			Eq(q, List(x, y, z)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// x=2, y=3, z=4
	list := result[0]
	vals := listToSlice(list)
	if len(vals) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(vals))
	}
	if !termEqual(vals[0], NewAtom(2)) || !termEqual(vals[1], NewAtom(3)) || !termEqual(vals[2], NewAtom(4)) {
		t.Errorf("Expected [2, 3, 4], got %v", vals)
	}
}

func TestArithmetic_Comparison(t *testing.T) {
	t.Parallel()
	// x + 2 < 10, find first x
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Eq(q, NewAtom(3)),
			Pluso(q, NewAtom(2), temp),
			LessThano(temp, NewAtom(10)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !termEqual(result[0], NewAtom(3)) {
		t.Errorf("Expected 3, got %v", result[0])
	}
}

// Context cancellation tests

func TestArithmetic_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	result := RunWithContext(ctx, -1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		return Conj(
			Pluso(x, y, NewAtom(1000)),
			Eq(q, NewPair(x, y)),
		)
	})

	// Should terminate early due to cancellation
	// Exact count varies based on timing
	if len(result) > 1001 {
		t.Errorf("Expected early termination, got %d results", len(result))
	}
}

// Helper functions

func listToSlice(term Term) []Term {
	var result []Term
	for {
		if pair, ok := term.(*Pair); ok {
			result = append(result, pair.Car())
			term = pair.Cdr()
		} else {
			break
		}
	}
	return result
}
