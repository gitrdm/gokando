// Package minikanren provides finite domain constraint programming with MiniKanren-style logical variables.
package minikanren

import (
	"fmt"
)

// Absolute implements the absolute value constraint: abs_value = |x|.
//
// Features:
//   - Production-ready: Handles positive/negative inputs via offset encoding
//   - Bidirectional propagation: Forward (x → |x|) and backward (|x| → x)
//   - Self-reference support: x = |x| (only valid for non-negative x)
//   - Domain splitting: Reconstructs both positive and negative solutions
//
// Constraints:
//   - BitSetDomain limitation: Only positive integers ≥ 1 are supported
//   - Uses offset encoding to represent negative values as positive integers
//   - Both variables must be initialized with proper offset-encoded domains
//   - abs_value domain contains only positive results (≥ 1)
//
// Mathematical Properties:
//   - |x| ≥ 0 for all real x, but BitSetDomain requires ≥ 1
//   - |0| = 0 is represented as offset value in the encoding
//   - |-x| = |x| creates symmetry in backward propagation
//   - Self-reference |x| = x implies x ≥ 0
//
// Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.
type Absolute struct {
	x        *FDVariable // The input value (offset-encoded for negative support)
	absValue *FDVariable // The absolute value result (always positive ≥ 1)
	offset   int         // Offset used to encode negative values as positive
}

// NewAbsolute creates a new absolute value constraint: abs_value = |x|.
//
// The constraint uses offset encoding to represent negative numbers within BitSetDomain constraints.
// For an offset O, the encoding is:
//   - Negative value -k is encoded as O - k
//   - Zero is encoded as O
//   - Positive value k is encoded as O + k
//
// Parameters:
//   - x: The FD variable representing the input value (offset-encoded)
//   - offset: The offset used for encoding negative values (must be > 0)
//   - absValue: The FD variable representing the absolute value (always ≥ 1)
//
// Returns error if:
//   - offset <= 0 (invalid encoding)
//   - any variable is nil
//
// Example:
//
//	// Temperature differences: x can be -10 to +10, |x| from 0 to 10
//	// Using offset = 20: domain [-10,10] → [10,30], |x| domain [1,11] (0→1 for BitSetDomain)
//	xVar := model.NewVariable(NewBitSetDomainFromValues(31, rangeValues(10, 30)))
//	absVar := model.NewVariable(NewBitSetDomainFromValues(12, rangeValues(1, 11)))
//	constraint, err := NewAbsolute(xVar, 20, absVar)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
func NewAbsolute(x *FDVariable, offset int, absValue *FDVariable) (*Absolute, error) {
	if x == nil || absValue == nil {
		return nil, fmt.Errorf("Absolute: variables cannot be nil")
	}
	if offset <= 0 {
		return nil, fmt.Errorf("Absolute: offset must be > 0, got %d", offset)
	}
	return &Absolute{
		x:        x,
		absValue: absValue,
		offset:   offset,
	}, nil
}

// Variables returns the FD variables involved in this constraint.
func (a *Absolute) Variables() []*FDVariable {
	return []*FDVariable{a.x, a.absValue}
}

// Clone creates an independent copy of this constraint.
func (a *Absolute) Clone() PropagationConstraint {
	return &Absolute{
		x:        a.x,
		absValue: a.absValue,
		offset:   a.offset,
	}
}

// Type returns the constraint type name.
func (a *Absolute) Type() string {
	return "Absolute"
}

// String returns a human-readable representation of the constraint.
func (a *Absolute) String() string {
	return fmt.Sprintf("Absolute(x:%d, |x|:%d, offset:%d)", a.x.ID(), a.absValue.ID(), a.offset)
}

// Propagate performs bidirectional arc-consistency enforcement for the absolute value constraint.
//
// Algorithm:
//  1. Check for self-reference (x = absValue) and handle specially
//  2. Forward propagation: Compute |x| values and prune absValue domain
//  3. Backward propagation: For each |x| value, find corresponding x values
//  4. Apply domain changes and detect failures
//
// The constraint maintains: absValue = |decode(x)| where decode(x) = x - offset.
//
// Returns the updated solver state, or error if the constraint is unsatisfiable.
func (a *Absolute) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	xDomain := solver.GetDomain(state, a.x.ID())
	absValueDomain := solver.GetDomain(state, a.absValue.ID())

	if xDomain == nil || absValueDomain == nil {
		return nil, fmt.Errorf("Absolute: variable domains not initialized")
	}

	if xDomain.Count() == 0 || absValueDomain.Count() == 0 {
		return nil, fmt.Errorf("Absolute: empty domain detected")
	}

	// Handle self-reference: |x| = x
	if a.x.ID() == a.absValue.ID() {
		// |x| = x is only possible when x ≥ 0
		// In offset encoding: x ≥ offset
		return a.handleSelfReference(solver, state, xDomain)
	}

	// Forward propagation: absValue ← |x|
	newAbsValueDomain := a.forwardPropagate(xDomain, absValueDomain)

	// Backward propagation: x ← values that produce valid absolute values
	newXDomain := a.backwardPropagate(newAbsValueDomain, xDomain)

	// Check for failure (empty domains)
	if newAbsValueDomain.Count() == 0 {
		return nil, fmt.Errorf("Absolute: absValue domain became empty (no valid absolute values)")
	}
	if newXDomain.Count() == 0 {
		return nil, fmt.Errorf("Absolute: x domain became empty (no valid input values)")
	}

	// Apply changes if domains were pruned
	if !newAbsValueDomain.Equal(absValueDomain) {
		state, _ = solver.SetDomain(state, a.absValue.ID(), newAbsValueDomain)
	}

	if !newXDomain.Equal(xDomain) {
		state, _ = solver.SetDomain(state, a.x.ID(), newXDomain)
	}

	// Return updated state
	return state, nil
}

// handleSelfReference handles the special case where |x| = x.
// This is only valid when x ≥ 0 (in offset encoding: x ≥ offset).
func (a *Absolute) handleSelfReference(solver *Solver, state *SolverState, xDomain Domain) (*SolverState, error) {
	// Find values in x domain that represent non-negative numbers (≥ offset)
	validValues := make([]int, 0)
	min, max := xDomain.Min(), xDomain.Max()

	for v := min; v <= max; v++ {
		if xDomain.Has(v) && v >= a.offset {
			// v represents a non-negative number
			validValues = append(validValues, v)
		}
	}

	// Create new domain with only valid values
	var newDomain Domain
	if len(validValues) == 0 {
		// No valid values - create empty domain
		newDomain = NewBitSetDomainFromValues(1, []int{})
	} else {
		maxVal := max
		if len(validValues) > 0 && validValues[len(validValues)-1] > maxVal {
			maxVal = validValues[len(validValues)-1]
		}
		newDomain = NewBitSetDomainFromValues(maxVal+1, validValues)
	}

	// Check if domain actually changed
	if newDomain.Equal(xDomain) {
		return state, nil
	}

	// Check for empty domain (failure)
	if newDomain.Count() == 0 {
		return nil, fmt.Errorf("Absolute: |x| = x has no valid solutions")
	}

	state, _ = solver.SetDomain(state, a.x.ID(), newDomain)
	return state, nil
}

// forwardPropagate prunes the absValue domain based on x values.
//
// For each value v in x.domain:
//   - Decode: actual_value = v - offset
//   - Compute: abs_actual = |actual_value|
//   - Encode: abs_encoded = abs_actual (but ensure ≥ 1 for BitSetDomain)
//   - Add abs_encoded to possible absValue values
//
// Returns a new domain with only feasible absolute values.
func (a *Absolute) forwardPropagate(xDomain, absValueDomain Domain) Domain {
	possibleAbsValues := make(map[int]bool)

	min, max := xDomain.Min(), xDomain.Max()
	for v := min; v <= max; v++ {
		if xDomain.Has(v) {
			absValue := a.computeAbsolute(v)
			possibleAbsValues[absValue] = true
		}
	}

	// Intersect with current absValue domain
	values := make([]int, 0, len(possibleAbsValues))
	aMin, aMax := absValueDomain.Min(), absValueDomain.Max()
	for a := aMin; a <= aMax; a++ {
		if absValueDomain.Has(a) && possibleAbsValues[a] {
			values = append(values, a)
		}
	}

	// Create new domain with valid values
	if len(values) == 0 {
		// Empty domain - return minimal empty domain
		return NewBitSetDomainFromValues(1, []int{})
	}

	// Determine max value for BitSetDomain size
	maxVal := aMax
	if len(values) > 0 && values[len(values)-1] > maxVal {
		maxVal = values[len(values)-1]
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}

// backwardPropagate prunes the x domain based on absValue values.
//
// For each value a in absValue.domain:
//   - Compute the original values that produce |x| = a
//   - These are: x = a and x = -a (in offset encoding)
//   - Offset encoding: +a → offset + a, -a → offset - a
//   - Add valid encoded values to possible x values
//
// Returns a new domain with only feasible x values.
func (a *Absolute) backwardPropagate(absValueDomain, xDomain Domain) Domain {
	possibleXValues := make(map[int]bool)

	aMin, aMax := absValueDomain.Min(), absValueDomain.Max()
	xMin, xMax := xDomain.Min(), xDomain.Max()

	for absVal := aMin; absVal <= aMax; absVal++ {
		if absValueDomain.Has(absVal) {
			// Decode absolute value (handle BitSetDomain ≥ 1 constraint)
			actualAbs := absVal
			if absVal == 1 {
				// Check if this represents 0 or 1
				// We need to check both possibilities
				actualAbs = 0 // Try 0 first
			}

			// Generate x values that produce this absolute value
			// Case 1: positive value (x = actualAbs)
			positiveX := a.offset + actualAbs
			if positiveX >= xMin && positiveX <= xMax {
				possibleXValues[positiveX] = true
			}

			// Case 2: negative value (x = -actualAbs), but only if actualAbs > 0
			if actualAbs > 0 {
				negativeX := a.offset - actualAbs
				if negativeX >= xMin && negativeX <= xMax && negativeX >= 1 {
					possibleXValues[negativeX] = true
				}
			}

			// If absVal was 1, also try actualAbs = 1
			if absVal == 1 {
				actualAbs = 1
				positiveX := a.offset + actualAbs
				if positiveX >= xMin && positiveX <= xMax {
					possibleXValues[positiveX] = true
				}
				negativeX := a.offset - actualAbs
				if negativeX >= xMin && negativeX <= xMax && negativeX >= 1 {
					possibleXValues[negativeX] = true
				}
			}
		}
	}

	// Intersect with current x domain
	values := make([]int, 0, len(possibleXValues))
	for x := xMin; x <= xMax; x++ {
		if xDomain.Has(x) && possibleXValues[x] {
			values = append(values, x)
		}
	}

	// Create new domain with valid values
	if len(values) == 0 {
		// Empty domain
		return NewBitSetDomainFromValues(1, []int{})
	}

	// Find max for BitSetDomain size
	maxVal := xMax
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}

// computeAbsolute computes the absolute value of an offset-encoded x value.
//
// Algorithm:
//   - Decode: actual_value = x - offset
//   - Compute: abs_value = |actual_value|
//   - Handle BitSetDomain constraint: if abs_value = 0, return 1
//
// This handles the BitSetDomain requirement that all values ≥ 1.
func (a *Absolute) computeAbsolute(x int) int {
	actualValue := x - a.offset
	absValue := actualValue
	if actualValue < 0 {
		absValue = -actualValue
	}

	// Handle BitSetDomain constraint (≥ 1)
	if absValue == 0 {
		return 1 // Encode 0 as 1 for BitSetDomain
	}
	return absValue
}
