// Package minikanren provides finite domain constraint programming with MiniKanren-style logical variables.
package minikanren

import (
	"fmt"
)

// IntervalOperation represents the type of interval arithmetic operation to perform.
type IntervalOperation int

const (
	// IntervalContainment ensures the variable's domain is contained within the specified interval
	IntervalContainment IntervalOperation = iota
	// IntervalIntersection computes the intersection of two intervals
	IntervalIntersection
	// IntervalUnion computes the union of two intervals (domain remains convex)
	IntervalUnion
	// IntervalSum adds two intervals: [a,b] + [c,d] = [a+c, b+d]
	IntervalSum
	// IntervalDifference subtracts intervals: [a,b] - [c,d] = [a-d, b-c]
	IntervalDifference
)

// String returns a human-readable representation of the interval operation.
func (op IntervalOperation) String() string {
	switch op {
	case IntervalContainment:
		return "containment"
	case IntervalIntersection:
		return "intersection"
	case IntervalUnion:
		return "union"
	case IntervalSum:
		return "sum"
	case IntervalDifference:
		return "difference"
	default:
		return "unknown"
	}
}

// IntervalArithmetic implements interval-based arithmetic constraints for robust numerical reasoning.
//
// Features:
//   - Production-ready: Handles multiple interval operations with precise bounds propagation
//   - Flexible operations: Containment, intersection, union, sum, difference between intervals
//   - Domain narrowing: Efficiently prunes variable domains based on interval constraints
//   - Bounds consistency: Maintains arc-consistency through interval endpoint propagation
//
// Constraints:
//   - BitSetDomain limitation: Only positive integers ≥ 1 are supported
//   - Interval endpoints must be positive values
//   - Result variable domain is constrained to valid interval operation results
//   - Operations maintain mathematical interval arithmetic properties
//
// Mathematical Properties:
//   - Containment: x ∈ [min, max] → domain(x) ⊆ [min, max]
//   - Intersection: [a,b] ∩ [c,d] = [max(a,c), min(b,d)]
//   - Union: [a,b] ∪ [c,d] = [min(a,c), max(b,d)] (convex hull)
//   - Sum: [a,b] + [c,d] = [a+c, b+d]
//   - Difference: [a,b] - [c,d] = [a-d, b-c]
//
// Thread Safety: Immutable after construction. Propagate() is safe for concurrent use.
type IntervalArithmetic struct {
	variable  *FDVariable       // The primary variable being constrained
	minBound  int               // Minimum bound of the interval (≥ 1 for BitSetDomain)
	maxBound  int               // Maximum bound of the interval
	operation IntervalOperation // The interval operation to perform
	result    *FDVariable       // Optional result variable for binary operations (can be nil for containment)
}

// NewIntervalArithmetic creates a new interval arithmetic constraint.
//
// For containment operations, only variable, minBound, maxBound are used (result should be nil).
// For binary operations (intersection, union, sum, difference), both variable and result are used.
//
// Parameters:
//   - variable: The FD variable to be constrained or first operand
//   - minBound: Minimum bound of the interval (must be ≥ 1)
//   - maxBound: Maximum bound of the interval (must be ≥ minBound)
//   - operation: The interval operation to perform
//   - result: The result variable for binary operations (nil for containment)
//
// Returns error if:
//   - variable is nil
//   - minBound < 1 or maxBound < minBound (invalid interval)
//   - binary operation with nil result variable
//   - containment operation with non-nil result variable
//
// Example (Containment):
//
//	// Ensure temperature is within valid range [1, 100]
//	tempVar := model.NewVariable(NewBitSetDomainFromValues(150, rangeValues(1, 150)))
//	constraint, err := NewIntervalArithmetic(tempVar, 1, 100, IntervalContainment, nil)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
//
// Example (Sum):
//
//	// Interval sum: [a,b] + [c,d] = [a+c, b+d]
//	interval1 := model.NewVariable(NewBitSetDomainFromValues(11, rangeValues(1, 10)))
//	result := model.NewVariable(NewBitSetDomainFromValues(21, rangeValues(1, 20)))
//	constraint, err := NewIntervalArithmetic(interval1, 5, 15, IntervalSum, result)
//	if err != nil {
//	    panic(err)
//	}
//	model.AddConstraint(constraint)
func NewIntervalArithmetic(variable *FDVariable, minBound, maxBound int, operation IntervalOperation, result *FDVariable) (*IntervalArithmetic, error) {
	if variable == nil {
		return nil, fmt.Errorf("IntervalArithmetic: variable cannot be nil")
	}
	if minBound < 1 {
		return nil, fmt.Errorf("IntervalArithmetic: minBound must be ≥ 1, got %d", minBound)
	}
	if maxBound < minBound {
		return nil, fmt.Errorf("IntervalArithmetic: maxBound (%d) must be ≥ minBound (%d)", maxBound, minBound)
	}

	// Validate operation-specific requirements
	switch operation {
	case IntervalContainment:
		if result != nil {
			return nil, fmt.Errorf("IntervalArithmetic: containment operation should have nil result variable")
		}
	case IntervalIntersection, IntervalUnion, IntervalSum, IntervalDifference:
		if result == nil {
			return nil, fmt.Errorf("IntervalArithmetic: %s operation requires non-nil result variable", operation.String())
		}
	default:
		return nil, fmt.Errorf("IntervalArithmetic: unknown operation %d", int(operation))
	}

	return &IntervalArithmetic{
		variable:  variable,
		minBound:  minBound,
		maxBound:  maxBound,
		operation: operation,
		result:    result,
	}, nil
}

// Variables returns the FD variables involved in this constraint.
func (ia *IntervalArithmetic) Variables() []*FDVariable {
	if ia.result == nil {
		return []*FDVariable{ia.variable}
	}
	return []*FDVariable{ia.variable, ia.result}
}

// Clone creates an independent copy of this constraint.
func (ia *IntervalArithmetic) Clone() PropagationConstraint {
	return &IntervalArithmetic{
		variable:  ia.variable,
		minBound:  ia.minBound,
		maxBound:  ia.maxBound,
		operation: ia.operation,
		result:    ia.result,
	}
}

// Type returns the constraint type name.
func (ia *IntervalArithmetic) Type() string {
	return "IntervalArithmetic"
}

// String returns a human-readable representation of the constraint.
func (ia *IntervalArithmetic) String() string {
	if ia.result == nil {
		return fmt.Sprintf("IntervalArithmetic(%s ∈ [%d,%d], %s)",
			ia.variable.Name(), ia.minBound, ia.maxBound, ia.operation.String())
	}
	return fmt.Sprintf("IntervalArithmetic(%s [%d,%d] %s %s)",
		ia.variable.Name(), ia.minBound, ia.maxBound, ia.operation.String(), ia.result.Name())
}

// Propagate performs interval arithmetic constraint propagation.
//
// Algorithm:
//  1. For containment: Intersect variable domain with [minBound, maxBound]
//  2. For binary operations: Compute interval arithmetic and propagate to result
//  3. Bidirectional propagation for binary operations when possible
//  4. Apply domain changes and detect failures
//
// Returns the updated solver state, or error if the constraint is unsatisfiable.
func (ia *IntervalArithmetic) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	variableDomain := solver.GetDomain(state, ia.variable.ID())
	if variableDomain == nil {
		return nil, fmt.Errorf("IntervalArithmetic: variable domain not initialized")
	}
	if variableDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: empty variable domain detected")
	}

	switch ia.operation {
	case IntervalContainment:
		return ia.propagateContainment(solver, state, variableDomain)
	case IntervalIntersection:
		return ia.propagateIntersection(solver, state, variableDomain)
	case IntervalUnion:
		return ia.propagateUnion(solver, state, variableDomain)
	case IntervalSum:
		return ia.propagateSum(solver, state, variableDomain)
	case IntervalDifference:
		return ia.propagateDifference(solver, state, variableDomain)
	default:
		return nil, fmt.Errorf("IntervalArithmetic: unsupported operation %s", ia.operation.String())
	}
}

// propagateContainment ensures the variable domain is contained within [minBound, maxBound].
func (ia *IntervalArithmetic) propagateContainment(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error) {
	// Intersect variable domain with [minBound, maxBound]
	newDomain := ia.intersectDomainWithInterval(variableDomain, ia.minBound, ia.maxBound)

	if newDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: containment constraint makes domain empty")
	}

	// Apply changes if domain was pruned
	if !newDomain.Equal(variableDomain) {
		state, _ = solver.SetDomain(state, ia.variable.ID(), newDomain)
	}

	return state, nil
}

// propagateIntersection computes interval intersection between variable interval and [minBound, maxBound].
func (ia *IntervalArithmetic) propagateIntersection(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error) {
	resultDomain := solver.GetDomain(state, ia.result.ID())
	if resultDomain == nil || resultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: result domain not initialized or empty")
	}

	// Compute intersection of variable interval with [minBound, maxBound]
	varMin, varMax := variableDomain.Min(), variableDomain.Max()
	intersectionMin := max(varMin, ia.minBound)
	intersectionMax := min(varMax, ia.maxBound)

	// Forward propagation: constrain result to intersection interval
	var newResultDomain Domain
	if intersectionMin > intersectionMax {
		// Empty intersection
		newResultDomain = NewBitSetDomainFromValues(1, []int{})
	} else {
		newResultDomain = ia.intersectDomainWithInterval(resultDomain, intersectionMin, intersectionMax)
	}

	// Backward propagation: constrain variable based on result requirements
	if newResultDomain.Count() > 0 {
		resultMin, resultMax := newResultDomain.Min(), newResultDomain.Max()
		// Variable must be within [resultMin, resultMax] and [minBound, maxBound]
		effectiveMin := max(resultMin, ia.minBound)
		effectiveMax := min(resultMax, ia.maxBound)
		newVariableDomain := ia.intersectDomainWithInterval(variableDomain, effectiveMin, effectiveMax)

		if newVariableDomain.Count() == 0 {
			return nil, fmt.Errorf("IntervalArithmetic: intersection constraint makes variable domain empty")
		}

		if !newVariableDomain.Equal(variableDomain) {
			state, _ = solver.SetDomain(state, ia.variable.ID(), newVariableDomain)
		}
	}

	if newResultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: intersection constraint makes result domain empty")
	}

	if !newResultDomain.Equal(resultDomain) {
		state, _ = solver.SetDomain(state, ia.result.ID(), newResultDomain)
	}

	return state, nil
}

// propagateUnion computes interval union (convex hull) between variable interval and [minBound, maxBound].
func (ia *IntervalArithmetic) propagateUnion(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error) {
	resultDomain := solver.GetDomain(state, ia.result.ID())
	if resultDomain == nil || resultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: result domain not initialized or empty")
	}

	// Compute union (convex hull) of variable interval with [minBound, maxBound]
	varMin, varMax := variableDomain.Min(), variableDomain.Max()
	unionMin := min(varMin, ia.minBound)
	unionMax := max(varMax, ia.maxBound)

	// Forward propagation: constrain result to union interval
	newResultDomain := ia.intersectDomainWithInterval(resultDomain, unionMin, unionMax)

	// Backward propagation: variable should contribute to the union
	if newResultDomain.Count() > 0 {
		resultMin, resultMax := newResultDomain.Min(), newResultDomain.Max()
		// Variable should be within the result range or complement [minBound, maxBound]
		newVariableDomain := ia.intersectDomainWithInterval(variableDomain, resultMin, resultMax)

		if newVariableDomain.Count() == 0 {
			return nil, fmt.Errorf("IntervalArithmetic: union constraint makes variable domain empty")
		}

		if !newVariableDomain.Equal(variableDomain) {
			state, _ = solver.SetDomain(state, ia.variable.ID(), newVariableDomain)
		}
	}

	if newResultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: union constraint makes result domain empty")
	}

	if !newResultDomain.Equal(resultDomain) {
		state, _ = solver.SetDomain(state, ia.result.ID(), newResultDomain)
	}

	return state, nil
}

// propagateSum computes interval sum: variable_interval + [minBound, maxBound] = result_interval.
func (ia *IntervalArithmetic) propagateSum(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error) {
	resultDomain := solver.GetDomain(state, ia.result.ID())
	if resultDomain == nil || resultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: result domain not initialized or empty")
	}

	// Forward propagation: result = variable + [minBound, maxBound]
	varMin, varMax := variableDomain.Min(), variableDomain.Max()
	resultMin := varMin + ia.minBound
	resultMax := varMax + ia.maxBound

	newResultDomain := ia.intersectDomainWithInterval(resultDomain, resultMin, resultMax)

	// Backward propagation: variable = result - [minBound, maxBound]
	if newResultDomain.Count() > 0 {
		resMin, resMax := newResultDomain.Min(), newResultDomain.Max()
		// variable = result - [minBound, maxBound] = [resMin - maxBound, resMax - minBound]
		varMinFromResult := max(1, resMin-ia.maxBound) // Ensure ≥ 1 for BitSetDomain
		varMaxFromResult := resMax - ia.minBound

		newVariableDomain := ia.intersectDomainWithInterval(variableDomain, varMinFromResult, varMaxFromResult)

		if newVariableDomain.Count() == 0 {
			return nil, fmt.Errorf("IntervalArithmetic: sum constraint makes variable domain empty")
		}

		if !newVariableDomain.Equal(variableDomain) {
			state, _ = solver.SetDomain(state, ia.variable.ID(), newVariableDomain)
		}
	}

	if newResultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: sum constraint makes result domain empty")
	}

	if !newResultDomain.Equal(resultDomain) {
		state, _ = solver.SetDomain(state, ia.result.ID(), newResultDomain)
	}

	return state, nil
}

// propagateDifference computes interval difference: variable_interval - [minBound, maxBound] = result_interval.
func (ia *IntervalArithmetic) propagateDifference(solver *Solver, state *SolverState, variableDomain Domain) (*SolverState, error) {
	resultDomain := solver.GetDomain(state, ia.result.ID())
	if resultDomain == nil || resultDomain.Count() == 0 {
		return nil, fmt.Errorf("IntervalArithmetic: result domain not initialized or empty")
	}

	// Forward propagation: result = variable - [minBound, maxBound]
	varMin, varMax := variableDomain.Min(), variableDomain.Max()
	resultMin := max(1, varMin-ia.maxBound) // Ensure ≥ 1 for BitSetDomain
	resultMax := varMax - ia.minBound

	// Only proceed if we get a valid result interval
	if resultMin <= resultMax {
		newResultDomain := ia.intersectDomainWithInterval(resultDomain, resultMin, resultMax)

		// Backward propagation: variable = result + [minBound, maxBound]
		if newResultDomain.Count() > 0 {
			resMin, resMax := newResultDomain.Min(), newResultDomain.Max()
			// variable = result + [minBound, maxBound] = [resMin + minBound, resMax + maxBound]
			varMinFromResult := resMin + ia.minBound
			varMaxFromResult := resMax + ia.maxBound

			newVariableDomain := ia.intersectDomainWithInterval(variableDomain, varMinFromResult, varMaxFromResult)

			if newVariableDomain.Count() == 0 {
				return nil, fmt.Errorf("IntervalArithmetic: difference constraint makes variable domain empty")
			}

			if !newVariableDomain.Equal(variableDomain) {
				state, _ = solver.SetDomain(state, ia.variable.ID(), newVariableDomain)
			}
		}

		if newResultDomain.Count() == 0 {
			return nil, fmt.Errorf("IntervalArithmetic: difference constraint makes result domain empty")
		}

		if !newResultDomain.Equal(resultDomain) {
			state, _ = solver.SetDomain(state, ia.result.ID(), newResultDomain)
		}
	} else {
		// Invalid result interval - constraint is unsatisfiable
		return nil, fmt.Errorf("IntervalArithmetic: difference constraint produces invalid interval [%d,%d]", resultMin, resultMax)
	}

	return state, nil
}

// intersectDomainWithInterval creates a new domain containing only values within [minVal, maxVal].
func (ia *IntervalArithmetic) intersectDomainWithInterval(domain Domain, minVal, maxVal int) Domain {
	if minVal > maxVal {
		return NewBitSetDomainFromValues(1, []int{}) // Empty domain
	}

	var values []int
	for v := minVal; v <= maxVal; v++ {
		if domain.Has(v) {
			values = append(values, v)
		}
	}

	if len(values) == 0 {
		return NewBitSetDomainFromValues(1, []int{}) // Empty domain
	}

	return NewBitSetDomainFromValues(maxVal+1, values)
}
