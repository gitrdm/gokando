// Package minikanren provides constraint propagation for finite-domain variables.
//
// This file implements scaling constraints for integer arithmetic.
// Scaling constraints enforce multiplicative relationships between variables
// while maintaining pure integer domains and providing bidirectional propagation.
//
// Design Philosophy:
//   - Integer-only: All operations work with integer values
//   - Bidirectional: Propagates both forward (x→result) and backward (result→x)
//   - AC-3 compatible: Implements standard arc-consistency propagation
//   - Production-ready: Handles edge cases (zero, negative, bounds checking)
//
// Example Use Case:
// In resource allocation problems where capacity scales linearly:
//
//	worker_hours = 40
//	total_cost = hourly_rate * worker_hours
//
// The Scale constraint maintains: total_cost = hourly_rate * 40
package minikanren

import (
	"fmt"
)

// Scale implements the constraint: result = x * multiplier
// where all values are positive integers and multiplier is a positive constant.
//
// Domain Constraints:
//   - All variables must have domains containing only positive integers (≥ 1)
//   - This is enforced by the underlying BitSetDomain implementation
//   - Values 0 and negative numbers are not supported
//
// The constraint maintains:
//   - Forward propagation: result ⊆ {x * multiplier | x ∈ x.domain}
//   - Backward propagation: x ⊆ {result / multiplier | result ∈ result.domain, result % multiplier == 0}
//
// This is arc-consistent propagation suitable for AC-3 and fixed-point iteration.
//
// Invariants:
//   - multiplier > 0 (enforced at construction)
//   - All variables must have non-nil domains with positive integer values
//   - Empty domain → immediate failure
//
// Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.
type Scale struct {
	x          *FDVariable // The value being scaled
	multiplier int         // The constant multiplier (must be > 0)
	result     *FDVariable // The result of scaling (x * multiplier)
}

// NewScale creates a new scaling constraint: result = x * multiplier.
//
// Parameters:
//   - x: The FD variable representing the input value
//   - multiplier: The constant integer multiplier (must be > 0)
//   - result: The FD variable representing the scaled result
//
// Returns error if:
//   - multiplier <= 0 (multiplication by zero or negative)
//   - any variable is nil
//
// Example:
//
//	// hourly_rate * 40 = total_cost (40-hour work week)
//	hourlyRateVar := model.NewVariable(NewBitSetDomainFromValues(101, []int{20, 25, 30}))
//	totalCostVar := model.NewVariable(NewBitSetDomainFromValues(1201, []int{800, 1000, 1200}))
//	constraint, err := NewScale(hourlyRateVar, 40, totalCostVar)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
func NewScale(x *FDVariable, multiplier int, result *FDVariable) (*Scale, error) {
	if x == nil || result == nil {
		return nil, fmt.Errorf("Scale: variables cannot be nil")
	}
	if multiplier <= 0 {
		return nil, fmt.Errorf("Scale: multiplier must be > 0, got %d", multiplier)
	}
	return &Scale{
		x:          x,
		multiplier: multiplier,
		result:     result,
	}, nil
}

// Variables returns the variables involved in this constraint.
// Used for dependency tracking and constraint graph construction.
// Implements ModelConstraint.
func (s *Scale) Variables() []*FDVariable {
	return []*FDVariable{s.x, s.result}
}

// Type returns the constraint type identifier.
// Implements ModelConstraint.
func (s *Scale) Type() string {
	return "Scale"
}

// String returns a human-readable representation of the constraint.
// Useful for debugging and logging.
// Implements ModelConstraint.
func (s *Scale) String() string {
	return fmt.Sprintf("Scale(%s * %d = %s)",
		s.x.Name(), s.multiplier, s.result.Name())
}

// Propagate applies bidirectional arc-consistency.
//
// Performs bidirectional arc-consistent propagation:
//  1. Forward: Prune result based on possible x * multiplier values
//  2. Backward: Prune x based on possible result / multiplier values (where result % multiplier == 0)
//  3. Detect conflicts: Empty domain after propagation → failure
//
// Returns:
//   - New solver state with pruned domains if propagation succeeded
//   - Original state if no changes
//   - Error if domains become empty (inconsistency detected)
//
// Complexity: O(|x.domain| + |result.domain|) for domain iteration
func (s *Scale) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	// Get current domains
	xDomain := solver.GetDomain(state, s.x.ID())
	resultDomain := solver.GetDomain(state, s.result.ID())

	if xDomain == nil || resultDomain == nil {
		return nil, fmt.Errorf("Scale: variable domains not initialized")
	}

	if xDomain.Count() == 0 || resultDomain.Count() == 0 {
		return nil, fmt.Errorf("Scale: empty domain detected")
	}

	// Handle self-reference: X * multiplier = X
	if s.x.ID() == s.result.ID() {
		if s.multiplier == 1 {
			// X * 1 = X is always true, no pruning needed
			return state, nil
		}
		// X * multiplier = X where multiplier != 1 is only true when X = 0
		if xDomain.Has(0) && xDomain.Count() == 1 {
			return state, nil // X = 0 satisfies X * k = X for any k
		}
		// If domain contains non-zero values or doesn't contain 0, it's impossible
		return nil, fmt.Errorf("Scale: X * %d = X is only possible when X = 0", s.multiplier)
	}

	// Forward propagation: result ← x * multiplier
	newResultDomain := s.forwardPropagate(xDomain, resultDomain)

	// Backward propagation: x ← result / multiplier (where result % multiplier == 0)
	newXDomain := s.backwardPropagate(newResultDomain, xDomain)

	// Check for failure (empty domains)
	if newResultDomain.Count() == 0 {
		return nil, fmt.Errorf("Scale: result domain became empty (no valid scaling results)")
	}
	if newXDomain.Count() == 0 {
		return nil, fmt.Errorf("Scale: x domain became empty (no valid input values)")
	}

	// Apply changes if domains were pruned
	changed := false
	if !newResultDomain.Equal(resultDomain) {
		state, changed = solver.SetDomain(state, s.result.ID(), newResultDomain)
	}
	if !newXDomain.Equal(xDomain) {
		state, _ = solver.SetDomain(state, s.x.ID(), newXDomain)
		changed = true
	}

	if !changed {
		return state, nil // Fixed point reached
	}

	return state, nil
}

// forwardPropagate prunes the result domain based on x values.
//
// For each value v in x.domain:
//   - Compute r = v * multiplier
//   - Keep r in result.domain if already present
//   - Remove from result.domain if no x value can produce it
//
// Returns a new domain with only feasible result values.
func (s *Scale) forwardPropagate(xDomain, resultDomain Domain) Domain {
	// Compute all possible result values from x
	possibleResults := make(map[int]bool)

	// Iterate over x domain
	min, max := xDomain.Min(), xDomain.Max()
	for v := min; v <= max; v++ {
		if xDomain.Has(v) {
			result := v * s.multiplier
			possibleResults[result] = true
		}
	}

	// Intersect with current result domain
	values := make([]int, 0, len(possibleResults))
	rMin, rMax := resultDomain.Min(), resultDomain.Max()
	for r := rMin; r <= rMax; r++ {
		if resultDomain.Has(r) && possibleResults[r] {
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

// backwardPropagate prunes the x domain based on result values.
//
// For each value r in result.domain:
//   - If r % multiplier == 0, compute x = r / multiplier
//   - Keep x in x.domain if already present
//   - Remove from x.domain if no result value can be produced by it
//
// Returns a new domain with only feasible x values.
func (s *Scale) backwardPropagate(resultDomain, xDomain Domain) Domain {
	// Compute all possible x values from result
	possibleXValues := make(map[int]bool)

	rMin, rMax := resultDomain.Min(), resultDomain.Max()
	for r := rMin; r <= rMax; r++ {
		if resultDomain.Has(r) {
			// Only consider results that are divisible by multiplier
			if r%s.multiplier == 0 {
				x := r / s.multiplier
				possibleXValues[x] = true
			}
		}
	}

	// Intersect with current x domain
	values := make([]int, 0, len(possibleXValues))
	xMin, xMax := xDomain.Min(), xDomain.Max()
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

// Clone creates a copy of the constraint with the same multiplier.
// The variable references are shared (constraints are immutable).
func (s *Scale) Clone() PropagationConstraint {
	return &Scale{
		x:          s.x,
		multiplier: s.multiplier,
		result:     s.result,
	}
}
