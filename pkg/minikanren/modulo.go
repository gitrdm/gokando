// Package minikanren provides constraint propagation for finite-domain variables.
//
// This file implements modulo constraints for integer arithmetic.
// Modulo constraints enforce remainder relationships between variables
// while maintaining pure integer domains and providing bidirectional propagation.
//
// Design Philosophy:
//   - Integer-only: All operations work with positive integer values (≥ 1)
//   - Bidirectional: Propagates both forward (x→remainder) and backward (remainder→x)
//   - AC-3 compatible: Implements standard arc-consistency propagation
//   - Production-ready: Handles edge cases (modulo 1, bounds checking)
//
// Example Use Case:
// In scheduling problems where events repeat cyclically:
//
//	day_of_week = day_number % 7
//	time_slot = minute_offset % 30
//
// The Modulo constraint maintains: x mod modulus = remainder
package minikanren

import (
	"fmt"
)

// Modulo implements the constraint: remainder = x mod modulus
// where all values are positive integers and modulus is a positive constant.
//
// Domain Constraints:
//   - All variables must have domains containing only positive integers (≥ 1)
//   - This is enforced by the underlying BitSetDomain implementation
//   - Values 0 and negative numbers are not supported
//
// The constraint maintains:
//   - Forward propagation: remainder ⊆ {x mod modulus | x ∈ x.domain}
//   - Backward propagation: x ⊆ {q*modulus + remainder | q ≥ 0, remainder ∈ remainder.domain}
//
// This is arc-consistent propagation suitable for AC-3 and fixed-point iteration.
//
// Invariants:
//   - modulus > 0 (enforced at construction)
//   - All variables must have non-nil domains with positive integer values
//   - Empty domain → immediate failure
//
// Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.
type Modulo struct {
	x         *FDVariable // The value being divided
	modulus   int         // The constant modulus (must be > 0)
	remainder *FDVariable // The remainder (x mod modulus)
}

// NewModulo creates a new modulo constraint: remainder = x mod modulus.
//
// Parameters:
//   - x: The FD variable representing the input value
//   - modulus: The constant integer modulus (must be > 0)
//   - remainder: The FD variable representing the remainder
//
// Returns error if:
//   - modulus <= 0 (modulo by zero or negative)
//   - any variable is nil
//
// Example:
//
//	// day_number mod 7 = day_of_week (0=Sun, 1=Mon, ..., 6=Sat, but using 1-7)
//	dayNumberVar := model.NewVariable(NewBitSetDomainFromValues(366, rangeValues(1, 365)))
//	dayOfWeekVar := model.NewVariable(NewBitSetDomainFromValues(8, rangeValues(1, 7)))
//	constraint, err := NewModulo(dayNumberVar, 7, dayOfWeekVar)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
func NewModulo(x *FDVariable, modulus int, remainder *FDVariable) (*Modulo, error) {
	if x == nil || remainder == nil {
		return nil, fmt.Errorf("Modulo: variables cannot be nil")
	}
	if modulus <= 0 {
		return nil, fmt.Errorf("Modulo: modulus must be > 0, got %d", modulus)
	}
	return &Modulo{
		x:         x,
		modulus:   modulus,
		remainder: remainder,
	}, nil
}

// Variables returns the variables involved in this constraint.
// Used for dependency tracking and constraint graph construction.
// Implements ModelConstraint.
func (m *Modulo) Variables() []*FDVariable {
	return []*FDVariable{m.x, m.remainder}
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (m *Modulo) Type() string {
	return "Modulo"
}

// String returns a human-readable representation of the constraint.
// Useful for debugging and logging.
// Implements ModelConstraint.
func (m *Modulo) String() string {
	return fmt.Sprintf("Modulo(%s mod %d = %s)",
		m.x.Name(), m.modulus, m.remainder.Name())
}

// Propagate applies bidirectional arc-consistency.
//
// Performs bidirectional arc-consistent propagation:
//  1. Forward: Prune remainder based on possible x mod modulus values
//  2. Backward: Prune x based on possible values that yield valid remainders
//  3. Detect conflicts: Empty domain after propagation → failure
//
// Returns:
//   - New solver state with pruned domains if propagation succeeded
//   - Original state if no changes
//   - Error if domains become empty (inconsistency detected)
//
// Complexity: O(|x.domain| + |remainder.domain|) for domain iteration
func (m *Modulo) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	// Get current domains
	xDomain := solver.GetDomain(state, m.x.ID())
	remainderDomain := solver.GetDomain(state, m.remainder.ID())

	if xDomain == nil || remainderDomain == nil {
		return nil, fmt.Errorf("Modulo: variable domains not initialized")
	}

	if xDomain.Count() == 0 || remainderDomain.Count() == 0 {
		return nil, fmt.Errorf("Modulo: empty domain detected")
	}

	// Handle self-reference: X mod modulus = X
	if m.x.ID() == m.remainder.ID() {
		// X mod k = X is only possible when X < k
		// Filter x domain to values less than modulus
		return m.handleSelfReference(solver, state, xDomain)
	}

	// Forward propagation: remainder ← x mod modulus
	newRemainderDomain := m.forwardPropagate(xDomain, remainderDomain)

	// Backward propagation: x ← values that produce valid remainders
	newXDomain := m.backwardPropagate(newRemainderDomain, xDomain)

	// Check for failure (empty domains)
	if newRemainderDomain.Count() == 0 {
		return nil, fmt.Errorf("Modulo: remainder domain became empty (no valid modulo results)")
	}
	if newXDomain.Count() == 0 {
		return nil, fmt.Errorf("Modulo: x domain became empty (no valid input values)")
	}

	// Apply changes if domains were pruned
	changed := false
	if !newRemainderDomain.Equal(remainderDomain) {
		state, changed = solver.SetDomain(state, m.remainder.ID(), newRemainderDomain)
	}
	if !newXDomain.Equal(xDomain) {
		state, _ = solver.SetDomain(state, m.x.ID(), newXDomain)
		changed = true
	}

	if !changed {
		return state, nil // Fixed point reached
	}

	return state, nil
}

// handleSelfReference handles the special case where X mod modulus = X.
// This is only valid when X < modulus.
func (m *Modulo) handleSelfReference(solver *Solver, state *SolverState, xDomain Domain) (*SolverState, error) {
	// Find values in x domain that are < modulus
	validValues := make([]int, 0)
	min, max := xDomain.Min(), xDomain.Max()

	for v := min; v <= max && v < m.modulus; v++ {
		if xDomain.Has(v) {
			validValues = append(validValues, v)
		}
	}

	// Create new domain with only valid values
	var newDomain Domain
	if len(validValues) == 0 {
		// No valid values - create empty domain
		newDomain = NewBitSetDomainFromValues(1, []int{})
	} else {
		maxVal := m.modulus - 1
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
		return nil, fmt.Errorf("Modulo: X mod %d = X has no valid solutions", m.modulus)
	}

	state, _ = solver.SetDomain(state, m.x.ID(), newDomain)
	return state, nil
}

// forwardPropagate prunes the remainder domain based on x values.
//
// For each value v in x.domain:
//   - Compute r = v mod modulus
//   - Keep r in remainder.domain if already present
//   - Remove from remainder.domain if no x value can produce it
//
// Returns a new domain with only feasible remainder values.
func (m *Modulo) forwardPropagate(xDomain, remainderDomain Domain) Domain {
	// Compute all possible remainder values from x
	possibleRemainders := make(map[int]bool)

	// Iterate over x domain
	min, max := xDomain.Min(), xDomain.Max()
	for v := min; v <= max; v++ {
		if xDomain.Has(v) {
			remainder := m.computeModulo(v)
			possibleRemainders[remainder] = true
		}
	}

	// Intersect with current remainder domain
	values := make([]int, 0, len(possibleRemainders))
	rMin, rMax := remainderDomain.Min(), remainderDomain.Max()
	for r := rMin; r <= rMax; r++ {
		if remainderDomain.Has(r) && possibleRemainders[r] {
			values = append(values, r)
		}
	}

	// Create new domain with valid values
	if len(values) == 0 {
		// Empty domain - return minimal empty domain
		return NewBitSetDomainFromValues(1, []int{})
	}

	// Determine max value for BitSetDomain size
	maxVal := rMax
	if len(values) > 0 && values[len(values)-1] > maxVal {
		maxVal = values[len(values)-1]
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}

// backwardPropagate prunes the x domain based on remainder values.
//
// For each value r in remainder.domain:
//   - Find all x values where x mod modulus = r
//   - Keep x values that are in the original x domain
//   - This generates: x ∈ {r, r+modulus, r+2*modulus, ...} ∩ x.domain
//
// Returns a new domain with only feasible x values.
func (m *Modulo) backwardPropagate(remainderDomain, xDomain Domain) Domain {
	// Compute all possible x values from remainders
	possibleXValues := make(map[int]bool)

	rMin, rMax := remainderDomain.Min(), remainderDomain.Max()
	xMin, xMax := xDomain.Min(), xDomain.Max()

	for r := rMin; r <= rMax; r++ {
		if remainderDomain.Has(r) {
			// Generate all x values where x mod modulus = r within the x domain range
			// x = r + k * modulus for k = 0, 1, 2, ...
			for x := r; x <= xMax; x += m.modulus {
				if x >= xMin {
					possibleXValues[x] = true
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

// computeModulo computes x mod modulus, handling the constraint that
// BitSetDomain only supports positive integers (≥ 1).
// Since we're working with positive integers, this is straightforward.
func (m *Modulo) computeModulo(x int) int {
	result := x % m.modulus
	// Since BitSetDomain requires values ≥ 1, we need to handle the case where result = 0
	// by converting it to modulus (e.g., 6 mod 3 = 0 becomes 3 in our representation)
	if result == 0 {
		return m.modulus
	}
	return result
}

// Clone creates a copy of the constraint with the same modulus.
// The variable references are shared (constraints are immutable).
func (m *Modulo) Clone() PropagationConstraint {
	return &Modulo{
		x:         m.x,
		modulus:   m.modulus,
		remainder: m.remainder,
	}
}
