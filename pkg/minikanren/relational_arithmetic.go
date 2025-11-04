package minikanren

import (
	"context"
	"fmt"
	"math"
)

// Pluso creates a relational addition goal: x + y = z.
// This operator works bidirectionally - it can solve for any of the three arguments
// given the other two.
//
// Modes of operation:
//   - (x, y, ?) → z = x + y (forward)
//   - (x, ?, z) → y = z - x (backward)
//   - (?, y, z) → x = z - y (backward)
//   - (?, ?, z) → generate pairs that sum to z
//
// Example:
//
//	x := Fresh("x")
//	result := Run(1, func(q *Var) Goal {
//	    return Conj(
//	        Pluso(NewAtom(2), NewAtom(3), q),  // 2 + 3 = ?
//	    )
//	})
//	// Result: [5]
func Pluso(x, y, z Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		xVal := sub.Walk(x)
		yVal := sub.Walk(y)
		zVal := sub.Walk(z)

		// Try to extract numeric values
		xNum, xIsNum := extractNumber(xVal)
		yNum, yIsNum := extractNumber(yVal)
		zNum, zIsNum := extractNumber(zVal)

		// Case 1: All ground - verify the constraint
		if xIsNum && yIsNum && zIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if xNum+yNum == zNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Case 2: Two known, one unknown - compute directly
		if xIsNum && yIsNum {
			// x + y = z, solve for z
			return Eq(z, NewAtom(xNum+yNum))(ctx, store)
		}

		if xIsNum && zIsNum {
			// x + y = z, solve for y
			return Eq(y, NewAtom(zNum-xNum))(ctx, store)
		}

		if yIsNum && zIsNum {
			// x + y = z, solve for x
			return Eq(x, NewAtom(zNum-yNum))(ctx, store)
		}

		// Case 3: Generate-and-test fallback
		// If z is known, generate pairs that sum to z
		if zIsNum {
			return plusoGenerate(x, y, zNum)(ctx, store)
		}

		// Otherwise, fail - need at least one bound value for computation
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// Minuso creates a relational subtraction goal: x - y = z.
// Works bidirectionally like Pluso.
//
// Modes:
//   - (x, y, ?) → z = x - y
//   - (x, ?, z) → y = x - z
//   - (?, y, z) → x = y + z
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Minuso(NewAtom(5), NewAtom(3), q)  // 5 - 3 = ?
//	})
//	// Result: [2]
func Minuso(x, y, z Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		xVal := sub.Walk(x)
		yVal := sub.Walk(y)
		zVal := sub.Walk(z)

		xNum, xIsNum := extractNumber(xVal)
		yNum, yIsNum := extractNumber(yVal)
		zNum, zIsNum := extractNumber(zVal)

		// All ground - verify
		if xIsNum && yIsNum && zIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if xNum-yNum == zNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Two known, compute third
		if xIsNum && yIsNum {
			return Eq(z, NewAtom(xNum-yNum))(ctx, store)
		}

		if xIsNum && zIsNum {
			return Eq(y, NewAtom(xNum-zNum))(ctx, store)
		}

		if yIsNum && zIsNum {
			return Eq(x, NewAtom(yNum+zNum))(ctx, store)
		}

		// Insufficient information to compute result
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// Timeso creates a relational multiplication goal: x * y = z.
// Works bidirectionally when possible.
//
// Modes:
//   - (x, y, ?) → z = x * y
//   - (x, ?, z) → y = z / x (if z divisible by x)
//   - (?, y, z) → x = z / y (if z divisible by y)
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Timeso(NewAtom(4), NewAtom(5), q)  // 4 * 5 = ?
//	})
//	// Result: [20]
func Timeso(x, y, z Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		xVal := sub.Walk(x)
		yVal := sub.Walk(y)
		zVal := sub.Walk(z)

		xNum, xIsNum := extractNumber(xVal)
		yNum, yIsNum := extractNumber(yVal)
		zNum, zIsNum := extractNumber(zVal)

		// All ground - verify
		if xIsNum && yIsNum && zIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if xNum*yNum == zNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Two known, compute third
		if xIsNum && yIsNum {
			return Eq(z, NewAtom(xNum*yNum))(ctx, store)
		}

		if xIsNum && zIsNum {
			// x * y = z, solve for y
			if xNum == 0 {
				// 0 * y = z, z must be 0
				if zNum == 0 {
					// y can be anything, succeed without binding
					stream := NewStream()
					go func() {
						defer stream.Close()
						stream.Put(store)
					}()
					return stream
				}
				// 0 * y = non-zero is impossible
				stream := NewStream()
				stream.Close()
				return stream
			}
			if zNum%xNum == 0 {
				return Eq(y, NewAtom(zNum/xNum))(ctx, store)
			}
			// Not evenly divisible, fail
			stream := NewStream()
			stream.Close()
			return stream
		}

		if yIsNum && zIsNum {
			// x * y = z, solve for x
			if yNum == 0 {
				if zNum == 0 {
					// x can be anything, succeed without binding
					stream := NewStream()
					go func() {
						defer stream.Close()
						stream.Put(store)
					}()
					return stream
				}
				stream := NewStream()
				stream.Close()
				return stream
			}
			if zNum%yNum == 0 {
				return Eq(x, NewAtom(zNum/yNum))(ctx, store)
			}
			stream := NewStream()
			stream.Close()
			return stream
		}

		// Insufficient information to compute result
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// Divo creates a relational division goal: x / y = z (integer division).
// Works bidirectionally when possible.
//
// Modes:
//   - (x, y, ?) → z = x / y
//   - (x, ?, z) → y = x / z
//   - (?, y, z) → x = y * z
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Divo(NewAtom(15), NewAtom(3), q)  // 15 / 3 = ?
//	})
//	// Result: [5]
func Divo(x, y, z Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		xVal := sub.Walk(x)
		yVal := sub.Walk(y)
		zVal := sub.Walk(z)

		xNum, xIsNum := extractNumber(xVal)
		yNum, yIsNum := extractNumber(yVal)
		zNum, zIsNum := extractNumber(zVal)

		// All ground - verify
		if xIsNum && yIsNum && zIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if yNum != 0 && xNum/yNum == zNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Two known, compute third
		if xIsNum && yIsNum {
			if yNum == 0 {
				// Division by zero, fail
				stream := NewStream()
				stream.Close()
				return stream
			}
			return Eq(z, NewAtom(xNum/yNum))(ctx, store)
		}

		if xIsNum && zIsNum {
			// x / y = z, solve for y
			if zNum == 0 {
				// x / y = 0, so |x| < |y|, y can be many values
				// For simplicity, we'll fail here
				stream := NewStream()
				stream.Close()
				return stream
			}
			return Eq(y, NewAtom(xNum/zNum))(ctx, store)
		}

		if yIsNum && zIsNum {
			// x / y = z, solve for x
			if yNum == 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}
			return Eq(x, NewAtom(yNum*zNum))(ctx, store)
		}

		// Fallback
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// Expo creates a relational exponentiation goal: base^exp = result.
// Supports multiple modes:
//   - (base, exp, ?) → result = base^exp (forward)
//   - (base, exp, result) → verify base^exp = result
//   - (?, exp, result) → solve for base (integer root)
//   - (base, ?, result) → solve for exp (logarithm)
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Expo(NewAtom(2), NewAtom(10), q)  // 2^10 = ?
//	})
//	// Result: [1024]
func Expo(base, exp, result Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		baseVal := sub.Walk(base)
		expVal := sub.Walk(exp)
		resultVal := sub.Walk(result)

		baseNum, baseIsNum := extractNumber(baseVal)
		expNum, expIsNum := extractNumber(expVal)
		resultNum, resultIsNum := extractNumber(resultVal)

		// All ground - verify
		if baseIsNum && expIsNum && resultIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if expNum < 0 {
					// Negative exponents produce fractions, not supported
					return
				}
				expected := int(math.Pow(float64(baseNum), float64(expNum)))
				if expected == resultNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Forward mode: base and exp known, solve for result
		if baseIsNum && expIsNum {
			if expNum < 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}
			computed := int(math.Pow(float64(baseNum), float64(expNum)))
			return Eq(result, NewAtom(computed))(ctx, store)
		}

		// Backward mode: exp and result known, solve for base
		// base^exp = result → base = result^(1/exp)
		if expIsNum && resultIsNum {
			if expNum <= 0 || resultNum < 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}

			// For integer roots, compute and verify
			computed := int(math.Round(math.Pow(float64(resultNum), 1.0/float64(expNum))))

			// Verify the result is exact (no fractional roots)
			if int(math.Pow(float64(computed), float64(expNum))) == resultNum {
				return Eq(base, NewAtom(computed))(ctx, store)
			}

			stream := NewStream()
			stream.Close()
			return stream
		}

		// Backward mode: base and result known, solve for exp
		// base^exp = result → exp = log_base(result)
		if baseIsNum && resultIsNum {
			if baseNum <= 0 || baseNum == 1 || resultNum <= 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}

			// Special case: result = 1 means exp = 0
			if resultNum == 1 {
				return Eq(exp, NewAtom(0))(ctx, store)
			}

			// Compute logarithm
			computed := int(math.Round(math.Log(float64(resultNum)) / math.Log(float64(baseNum))))

			// Verify the result is exact
			if int(math.Pow(float64(baseNum), float64(computed))) == resultNum {
				return Eq(exp, NewAtom(computed))(ctx, store)
			}

			stream := NewStream()
			stream.Close()
			return stream
		}

		// Cannot solve with multiple unknowns
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// Logo creates a relational logarithm goal: log_base(value) = result.
// Supports multiple modes:
//   - (base, value, ?) → result = log_base(value) (forward)
//   - (base, value, result) → verify log_base(value) = result
//   - (base, ?, result) → solve for value (exponential: value = base^result)
//   - (?, value, result) → solve for base (inverse logarithm)
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Logo(NewAtom(2), NewAtom(1024), q)  // log2(1024) = ?
//	})
//	// Result: [10]
func Logo(base, value, result Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		sub := store.GetSubstitution()
		baseVal := sub.Walk(base)
		valueVal := sub.Walk(value)
		resultVal := sub.Walk(result)

		baseNum, baseIsNum := extractNumber(baseVal)
		valueNum, valueIsNum := extractNumber(valueVal)
		resultNum, resultIsNum := extractNumber(resultVal)

		// All ground - verify
		if baseIsNum && valueIsNum && resultIsNum {
			stream := NewStream()
			go func() {
				defer stream.Close()
				if baseNum <= 0 || baseNum == 1 || valueNum <= 0 {
					return
				}
				computed := int(math.Round(math.Log(float64(valueNum)) / math.Log(float64(baseNum))))
				if computed == resultNum {
					stream.Put(store)
				}
			}()
			return stream
		}

		// Forward mode: base and value known, solve for result
		// log_base(value) = result
		if baseIsNum && valueIsNum {
			if baseNum <= 0 || baseNum == 1 || valueNum <= 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}
			computed := int(math.Round(math.Log(float64(valueNum)) / math.Log(float64(baseNum))))
			return Eq(result, NewAtom(computed))(ctx, store)
		}

		// Backward mode: base and result known, solve for value
		// log_base(value) = result → value = base^result
		if baseIsNum && resultIsNum {
			if baseNum <= 0 || baseNum == 1 || resultNum < 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}
			computed := int(math.Pow(float64(baseNum), float64(resultNum)))
			return Eq(value, NewAtom(computed))(ctx, store)
		}

		// Backward mode: value and result known, solve for base
		// log_base(value) = result → base^result = value → base = value^(1/result)
		if valueIsNum && resultIsNum {
			if valueNum <= 0 || resultNum <= 0 {
				stream := NewStream()
				stream.Close()
				return stream
			}

			// Special case: value = 1 means any base works (but we'll use base = value)
			if valueNum == 1 {
				stream := NewStream()
				stream.Close()
				return stream
			}

			computed := int(math.Round(math.Pow(float64(valueNum), 1.0/float64(resultNum))))

			// Verify the result is exact
			if int(math.Pow(float64(computed), float64(resultNum))) == valueNum {
				return Eq(base, NewAtom(computed))(ctx, store)
			}

			stream := NewStream()
			stream.Close()
			return stream
		}

		// Cannot solve with multiple unknowns
		stream := NewStream()
		stream.Close()
		return stream
	}
}

// LessThanConstraint represents a constraint that x < y.
// It is evaluated whenever variables become bound.
type LessThanConstraint struct {
	id string
	x  Term
	y  Term
}

// ID returns the unique identifier for this constraint.
func (ltc *LessThanConstraint) ID() string {
	return ltc.id
}

// IsLocal returns true since this constraint can be evaluated locally.
func (ltc *LessThanConstraint) IsLocal() bool {
	return true
}

// Variables returns the logic variables involved in this constraint.
func (ltc *LessThanConstraint) Variables() []*Var {
	vars := make([]*Var, 0, 2)
	if v, ok := ltc.x.(*Var); ok {
		vars = append(vars, v)
	}
	if v, ok := ltc.y.(*Var); ok {
		vars = append(vars, v)
	}
	return vars
}

// Check evaluates the less-than constraint against current bindings.
func (ltc *LessThanConstraint) Check(bindings map[int64]Term) ConstraintResult {
	sub := &Substitution{bindings: bindings}
	xVal := sub.Walk(ltc.x)
	yVal := sub.Walk(ltc.y)

	xNum, xIsNum := extractNumber(xVal)
	yNum, yIsNum := extractNumber(yVal)

	if !xIsNum || !yIsNum {
		// Not both ground yet - constraint is pending
		return ConstraintPending
	}

	if xNum < yNum {
		return ConstraintSatisfied
	}

	return ConstraintViolated
}

// String returns a human-readable representation.
func (ltc *LessThanConstraint) String() string {
	return fmt.Sprintf("LessThan(%v, %v)", ltc.x, ltc.y)
}

// Clone creates a copy of this constraint.
func (ltc *LessThanConstraint) Clone() Constraint {
	return &LessThanConstraint{
		id: ltc.id,
		x:  ltc.x,
		y:  ltc.y,
	}
}

// LessEqualConstraint represents a constraint that x <= y.
type LessEqualConstraint struct {
	id string
	x  Term
	y  Term
}

// ID returns the unique identifier for this constraint.
func (lec *LessEqualConstraint) ID() string {
	return lec.id
}

// IsLocal returns true since this constraint can be evaluated locally.
func (lec *LessEqualConstraint) IsLocal() bool {
	return true
}

// Variables returns the logic variables involved in this constraint.
func (lec *LessEqualConstraint) Variables() []*Var {
	vars := make([]*Var, 0, 2)
	if v, ok := lec.x.(*Var); ok {
		vars = append(vars, v)
	}
	if v, ok := lec.y.(*Var); ok {
		vars = append(vars, v)
	}
	return vars
}

// Check evaluates the less-equal constraint against current bindings.
func (lec *LessEqualConstraint) Check(bindings map[int64]Term) ConstraintResult {
	sub := &Substitution{bindings: bindings}
	xVal := sub.Walk(lec.x)
	yVal := sub.Walk(lec.y)

	xNum, xIsNum := extractNumber(xVal)
	yNum, yIsNum := extractNumber(yVal)

	if !xIsNum || !yIsNum {
		return ConstraintPending
	}

	if xNum <= yNum {
		return ConstraintSatisfied
	}

	return ConstraintViolated
}

// String returns a human-readable representation.
func (lec *LessEqualConstraint) String() string {
	return fmt.Sprintf("LessEqual(%v, %v)", lec.x, lec.y)
}

// Clone creates a copy of this constraint.
func (lec *LessEqualConstraint) Clone() Constraint {
	return &LessEqualConstraint{
		id: lec.id,
		x:  lec.x,
		y:  lec.y,
	}
}

// LessThano creates a relational less-than goal: x < y.
// This constraint is reified - it's added to the constraint store and
// evaluated whenever variables become bound, ensuring goal order independence.
//
// Example:
//
//	result := Run(3, func(q *Var) Goal {
//	    return Conj(
//	        Membero(q, List(NewAtom(1), NewAtom(3), NewAtom(7))),
//	        LessThano(q, NewAtom(5)),
//	    )
//	})
//	// Result: [1, 3]
func LessThano(x, y Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		stream := NewStream()

		go func() {
			defer stream.Close()

			// Create the less-than constraint
			constraint := &LessThanConstraint{
				id: fmt.Sprintf("LessThan-%p", &x),
				x:  x,
				y:  y,
			}

			// Try to add the constraint to the store
			if err := store.AddConstraint(constraint); err != nil {
				// Constraint was immediately violated
				return
			}

			// Constraint was added successfully
			stream.Put(store)
		}()

		return stream
	}
}

// GreaterThano creates a relational greater-than goal: x > y.
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Conj(
//	        GreaterThano(NewAtom(10), NewAtom(5)),
//	        Eq(q, NewAtom("yes")),
//	    )
//	})
//	// Result: ["yes"]
func GreaterThano(x, y Term) Goal {
	return LessThano(y, x) // x > y ⇔ y < x
}

// LessEqualo creates a relational less-than-or-equal goal: x ≤ y.
// This constraint is reified and evaluated when variables become bound.
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Conj(
//	        LessEqualo(NewAtom(5), NewAtom(5)),
//	        Eq(q, NewAtom("yes")),
//	    )
//	})
//	// Result: ["yes"]
func LessEqualo(x, y Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		stream := NewStream()

		go func() {
			defer stream.Close()

			// Create the less-equal constraint
			constraint := &LessEqualConstraint{
				id: fmt.Sprintf("LessEqual-%p", &x),
				x:  x,
				y:  y,
			}

			// Try to add the constraint to the store
			if err := store.AddConstraint(constraint); err != nil {
				// Constraint was immediately violated
				return
			}

			// Constraint was added successfully
			stream.Put(store)
		}()

		return stream
	}
}

// GreaterEqualo creates a relational greater-than-or-equal goal: x ≥ y.
func GreaterEqualo(x, y Term) Goal {
	return LessEqualo(y, x) // x ≥ y ⇔ y ≤ x
}

// Helper functions

// extractNumber tries to extract an integer from a term.
func extractNumber(term Term) (int, bool) {
	if atom, ok := term.(*Atom); ok {
		switch v := atom.Value().(type) {
		case int:
			return v, true
		case int64:
			return int(v), true
		case int32:
			return int(v), true
		case float64:
			// Only accept if it's actually an integer
			if v == float64(int(v)) {
				return int(v), true
			}
		}
	}
	return 0, false
}

// plusoGenerate generates pairs (x, y) that sum to z.
// Used when z is known but x and y are not.
func plusoGenerate(x, y Term, z int) Goal {
	// Generate solutions: for integers from some reasonable range
	// We'll generate pairs where x ranges and y = z - x
	// For simplicity, limit to a reasonable range around z
	const maxRange = 1000
	start := z - maxRange
	end := z + maxRange
	if z >= 0 && z < maxRange {
		start = 0
		end = z
	} else if z < 0 && z > -maxRange {
		start = z
		end = 0
	}

	goals := make([]Goal, 0)
	for i := start; i <= end; i++ {
		j := z - i
		iVal := i
		jVal := j
		goals = append(goals, Conj(
			Eq(x, NewAtom(iVal)),
			Eq(y, NewAtom(jVal)),
		))
	}
	if len(goals) == 0 {
		return func(ctx context.Context, store ConstraintStore) *Stream {
			stream := NewStream()
			stream.Close()
			return stream
		}
	}
	return Disj(goals...)
}
