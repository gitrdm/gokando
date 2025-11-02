// Package minikanren provides global constraints for finite-domain CP.
//
// This file defines a production-quality NoOverlap (a.k.a. Disjunctive)
// constraint constructor built on top of the Cumulative global.
//
// NoOverlap models a set of non-preemptive tasks on a single machine (capacity 1):
// no two tasks may execute at the same time. Each task i has a start-time
// variable start[i] and a fixed positive duration dur[i].
//
// Implementation strategy:
//   - NoOverlap(starts, durations) is modeled as Cumulative with capacity=1,
//     unit demands for all tasks, and the given durations.
//   - Propagation strength is that of the Cumulative implementation: time-table
//     filtering with compulsory parts, which is sound and effective for many
//     scheduling problems.
//   - This mirrors a standard CP modeling technique and composes well with other
//     constraints (precedences, objective variables, etc.).
package minikanren

import "fmt"

// NewNoOverlap constructs a NoOverlap (disjunctive) constraint over tasks.
//
// Parameters:
//   - starts: start-time FD variables (len n > 0)
//   - durations: strictly positive integer durations (len n; each > 0)
//
// Semantics:
//   - Each task i occupies the half-open interval [start[i], start[i]+dur[i])
//     modeled as inclusive [start[i], start[i]+dur[i]-1] for 1-based discrete time.
//   - For any time t, at most one task may execute (capacity=1).
//
// Returns a PropagationConstraint implementing NoOverlap, or an error on invalid input.
// Internally this builds a Cumulative(starts, durations, demands=1, capacity=1).
func NewNoOverlap(starts []*FDVariable, durations []int) (PropagationConstraint, error) {
	n := len(starts)
	if n == 0 {
		return nil, fmt.Errorf("NoOverlap: requires at least one task")
	}
	if len(durations) != n {
		return nil, fmt.Errorf("NoOverlap: mismatched lengths (starts=%d, durations=%d)", n, len(durations))
	}
	// Build unit demands and delegate to Cumulative(capacity=1)
	demands := make([]int, n)
	for i := 0; i < n; i++ {
		demands[i] = 1
	}
	return NewCumulative(starts, durations, demands, 1)
}
