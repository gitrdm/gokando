// Package minikanren provides constraint propagation for finite-domain variables.
//
// This file implements scaled division constraints for integer arithmetic.
// Scaled division allows division-like reasoning while maintaining pure integer
// domains, following the PicoLisp pattern of global scale factors.
//
// Design Philosophy:
//   - Integer-only: All operations work with scaled integer values
//   - Bidirectional: Propagates both forward (dividend→quotient) and backward (quotient→dividend)
//   - AC-3 compatible: Implements standard arc-consistency propagation
//   - Production-ready: Handles edge cases (zero, negative, bounds checking)
//
// Example Use Case:
// If all monetary values are scaled by 100 (cents), then:
//
//	salary_cents = 5000000 (representing $50,000.00)
//	bonus_cents = salary_cents / 100 (representing 10% bonus)
//
// The ScaledDivision constraint maintains: bonus * 100 ⊆ [salary, salary+99]
package minikanren

import (
	"fmt"
)

// ScaledDivision implements the constraint: dividend / divisor = quotient
// where all values are integers and division is integer division (truncating).
//
// The constraint maintains:
//   - Forward propagation: quotient ⊆ {⌊d/divisor⌋ | d ∈ dividend.domain}
//   - Backward propagation: dividend ⊆ {q*divisor...(q+1)*divisor-1 | q ∈ quotient.domain}
//
// This is arc-consistent propagation suitable for AC-3 and fixed-point iteration.
//
// Invariants:
//   - divisor > 0 (enforced at construction)
//   - All variables must have non-nil domains
//   - Empty domain → immediate failure
//
// Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.
type ScaledDivision struct {
	dividend *FDVariable // The value being divided
	divisor  int         // The constant divisor (must be > 0)
	quotient *FDVariable // The result of division
}

// NewScaledDivision creates a new scaled division constraint.
//
// Parameters:
//   - dividend: The FD variable representing the numerator
//   - divisor: The constant integer divisor (must be > 0)
//   - quotient: The FD variable representing the result
//
// Returns error if:
//   - divisor <= 0 (division by zero or negative)
//   - any variable is nil
//
// Example:
//
//	// salary / 10 = bonus (10% bonus calculation)
//	salaryVar := model.NewVariable(NewBitSetDomainFromValues(100000, []int{50000, 60000, 70000}))
//	bonusVar := model.NewVariable(NewBitSetDomainFromValues(10000, []int{5000, 6000, 7000}))
//	constraint, err := NewScaledDivision(salaryVar, 10, bonusVar)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
func NewScaledDivision(dividend *FDVariable, divisor int, quotient *FDVariable) (*ScaledDivision, error) {
	if dividend == nil {
		return nil, fmt.Errorf("dividend variable cannot be nil")
	}
	if quotient == nil {
		return nil, fmt.Errorf("quotient variable cannot be nil")
	}
	if divisor <= 0 {
		return nil, fmt.Errorf("divisor must be positive, got %d", divisor)
	}

	return &ScaledDivision{
		dividend: dividend,
		divisor:  divisor,
		quotient: quotient,
	}, nil
}

// Propagate implements the PropagationConstraint interface.
//
// Performs bidirectional arc-consistent propagation:
//  1. Forward: Prune quotient based on possible dividend/divisor values
//  2. Backward: Prune dividend based on possible quotient*divisor ranges
//  3. Detect conflicts: Empty domain after propagation → failure
//
// Returns:
//   - New solver state with pruned domains if propagation succeeded
//   - Original state if no changes
//   - Error if domains become empty (inconsistency detected)
//
// Complexity: O(|dividend.domain| + |quotient.domain|) for domain iteration
func (sd *ScaledDivision) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	// Get current domains
	dividendDomain := solver.GetDomain(state, sd.dividend.ID())
	quotientDomain := solver.GetDomain(state, sd.quotient.ID())

	if dividendDomain == nil || quotientDomain == nil {
		return nil, fmt.Errorf("scaled division: variable domains not initialized")
	}

	if dividendDomain.Count() == 0 || quotientDomain.Count() == 0 {
		return nil, fmt.Errorf("scaled division: empty domain detected")
	}

	// Forward propagation: dividend → quotient
	// For each value d in dividend, quotient must include ⌊d/divisor⌋
	newQuotientDomain := sd.forwardPropagate(dividendDomain, quotientDomain)

	// Backward propagation: quotient → dividend
	// For each value q in quotient, dividend must include [q*divisor, (q+1)*divisor - 1]
	newDividendDomain := sd.backwardPropagate(quotientDomain, dividendDomain)

	// Check for failure (empty domains)
	if newQuotientDomain.Count() == 0 {
		return nil, fmt.Errorf("scaled division: quotient domain became empty (no valid division results)")
	}
	if newDividendDomain.Count() == 0 {
		return nil, fmt.Errorf("scaled division: dividend domain became empty (no valid dividend values)")
	}

	// Apply changes if domains were pruned
	changed := false
	if !newQuotientDomain.Equal(quotientDomain) {
		state, changed = solver.SetDomain(state, sd.quotient.ID(), newQuotientDomain)
	}
	if !newDividendDomain.Equal(dividendDomain) {
		state, _ = solver.SetDomain(state, sd.dividend.ID(), newDividendDomain)
		changed = true
	}

	if !changed {
		return state, nil // Fixed point reached
	}

	return state, nil
}

// forwardPropagate prunes the quotient domain based on dividend values.
//
// For each value d in dividend.domain:
//   - Compute q = ⌊d/divisor⌋
//   - Keep q in quotient.domain if already present
//   - Remove from quotient.domain if no dividend value can produce it
//
// Returns a new domain with only feasible quotient values.
func (sd *ScaledDivision) forwardPropagate(dividendDomain, quotientDomain Domain) Domain {
	// Compute all possible quotient values from dividend
	possibleQuotients := make(map[int]bool)

	// Iterate over dividend domain
	min, max := dividendDomain.Min(), dividendDomain.Max()
	for d := min; d <= max; d++ {
		if dividendDomain.Has(d) {
			quotient := d / sd.divisor // Integer division (truncating)
			possibleQuotients[quotient] = true
		}
	}

	// Intersect with current quotient domain
	values := make([]int, 0, len(possibleQuotients))
	qMin, qMax := quotientDomain.Min(), quotientDomain.Max()
	for q := qMin; q <= qMax; q++ {
		if quotientDomain.Has(q) && possibleQuotients[q] {
			values = append(values, q)
		}
	}

	// Create new domain with valid values
	if len(values) == 0 {
		// Empty domain - return minimal empty domain
		return NewBitSetDomainFromValues(1, []int{})
	}

	// Determine max value for BitSetDomain size
	maxVal := qMax
	if len(values) > 0 && values[len(values)-1] > maxVal {
		maxVal = values[len(values)-1]
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}

// backwardPropagate prunes the dividend domain based on quotient values.
//
// For each value q in quotient.domain:
//   - Compute range [q*divisor, (q+1)*divisor - 1]
//   - Keep dividend values in this range
//   - Remove dividend values outside all ranges
//
// Returns a new domain with only feasible dividend values.
func (sd *ScaledDivision) backwardPropagate(quotientDomain, dividendDomain Domain) Domain {
	// Compute all possible dividend values from quotient
	possibleDividends := make(map[int]bool)

	qMin, qMax := quotientDomain.Min(), quotientDomain.Max()
	for q := qMin; q <= qMax; q++ {
		if quotientDomain.Has(q) {
			// For quotient q, dividend can be [q*divisor, (q+1)*divisor - 1]
			rangeStart := q * sd.divisor
			rangeEnd := (q+1)*sd.divisor - 1

			// Add all dividend values in this range
			dMin, dMax := dividendDomain.Min(), dividendDomain.Max()
			for d := rangeStart; d <= rangeEnd; d++ {
				// Only include if within dividend's current domain bounds
				if d >= dMin && d <= dMax && dividendDomain.Has(d) {
					possibleDividends[d] = true
				}
			}
		}
	}

	// Convert to sorted values for domain construction
	values := make([]int, 0, len(possibleDividends))
	for d := range possibleDividends {
		values = append(values, d)
	}

	if len(values) == 0 {
		// Empty domain
		return NewBitSetDomainFromValues(1, []int{})
	}

	// Find max for BitSetDomain size
	maxVal := dividendDomain.Max()
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}

// Vars returns the variables involved in this constraint.
// Used for dependency tracking and constraint graph construction.
// Implements ModelConstraint.
func (sd *ScaledDivision) Variables() []*FDVariable {
	return []*FDVariable{sd.dividend, sd.quotient}
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (sd *ScaledDivision) Type() string {
	return "ScaledDivision"
}

// String returns a human-readable representation of the constraint.
// Useful for debugging and logging.
// Implements ModelConstraint.
func (sd *ScaledDivision) String() string {
	return fmt.Sprintf("ScaledDivision(%s / %d = %s)",
		sd.dividend.Name(), sd.divisor, sd.quotient.Name())
}

// Clone creates a copy of the constraint with the same divisor.
// The variables references are shared (constraints are immutable).
func (sd *ScaledDivision) Clone() PropagationConstraint {
	return &ScaledDivision{
		dividend: sd.dividend,
		divisor:  sd.divisor,
		quotient: sd.quotient,
	}
}
