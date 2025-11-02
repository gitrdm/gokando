// Package minikanren implements global constraints for finite-domain CP.
//
// This file provides a production-quality implementation of the Cumulative
// constraint, a classic resource scheduling constraint. Given a set of tasks
// with start-time variables, fixed durations and resource demands, and a fixed
// resource capacity, Cumulative enforces that at every time unit the sum of
// demands of tasks executing at that time does not exceed the capacity.
//
// Contract (discrete time, 1-based domains):
//   - For each task i:
//     start[i] is an FD variable with integer domain of possible start times
//     dur[i]   is a strictly positive integer duration (time units)
//     dem[i]   is a non-negative integer resource demand
//   - Capacity C is a strictly positive integer
//   - A task scheduled at start s occupies the half-open interval [s, s+dur[i])
//     which, with discrete 1-based times, we model as the inclusive range
//     [s, s+dur[i]-1]. Two tasks overlap at time t if t is contained in both
//     of their inclusive ranges.
//
// Propagation strength: time-table filtering with compulsory parts.
//   - We compute compulsory parts for each task from the current start windows:
//     est = min(start[i]), lst = max(start[i])
//     If lst <= est+dur[i]-1, the task must execute over the inclusive range
//     [lst, est+dur[i]-1] regardless of the exact start.
//   - We build a resource profile by summing demands over the union of all
//     compulsory parts. If the profile ever exceeds capacity, we report
//     inconsistency immediately.
//   - For pruning, we remove any start value s for task i such that placing
//     the task at [s, s+dur[i]-1] would push the profile above capacity at any
//     time t in that range (i.e., profile[t] + dem[i] > capacity).
//
// This achieves sound bounds-consistent pruning commonly known as time-table
// propagation. It is not as strong as edge-finding, but is fast, robust, and
// catches many practical conflicts. The solver's fixed-point loop composes
// this filtering with other constraints.
package minikanren

import (
	"fmt"
)

// Cumulative models a single renewable resource with fixed capacity consumed
// by a set of tasks with fixed durations and demands.
type Cumulative struct {
	starts    []*FDVariable // start-time variables (1-based discrete time)
	durations []int         // strictly positive
	demands   []int         // non-negative
	capacity  int           // strictly positive
}

// NewCumulative constructs a Cumulative constraint.
//
// Parameters:
//   - starts: start-time variables (length n > 0)
//   - durations: positive durations (length n; each > 0)
//   - demands: non-negative demands (length n; each >= 0)
//   - capacity: total resource capacity (must be > 0)
//
// Returns an error if inputs are invalid.
func NewCumulative(starts []*FDVariable, durations, demands []int, capacity int) (PropagationConstraint, error) {
	n := len(starts)
	if n == 0 {
		return nil, fmt.Errorf("Cumulative requires at least one task")
	}
	if len(durations) != n || len(demands) != n {
		return nil, fmt.Errorf("Cumulative: mismatched lengths (starts=%d, durations=%d, demands=%d)", n, len(durations), len(demands))
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("Cumulative: capacity must be > 0")
	}
	for i := 0; i < n; i++ {
		if starts[i] == nil {
			return nil, fmt.Errorf("Cumulative: starts[%d] is nil", i)
		}
		if durations[i] <= 0 {
			return nil, fmt.Errorf("Cumulative: durations[%d] must be > 0", i)
		}
		if demands[i] < 0 {
			return nil, fmt.Errorf("Cumulative: demands[%d] must be >= 0", i)
		}
	}

	// Defensive copies
	startsCopy := make([]*FDVariable, n)
	copy(startsCopy, starts)
	dursCopy := make([]int, n)
	copy(dursCopy, durations)
	demsCopy := make([]int, n)
	copy(demsCopy, demands)

	return &Cumulative{
		starts:    startsCopy,
		durations: dursCopy,
		demands:   demsCopy,
		capacity:  capacity,
	}, nil
}

// Variables returns the variables involved in this constraint.
func (c *Cumulative) Variables() []*FDVariable { return c.starts }

// Type returns the constraint identifier.
func (c *Cumulative) Type() string { return "Cumulative" }

// String returns a readable description.
func (c *Cumulative) String() string {
	return fmt.Sprintf("Cumulative(n=%d, capacity=%d)", len(c.starts), c.capacity)
}

// Propagate performs time-table filtering using compulsory parts.
// See the file header for algorithmic notes.
func (c *Cumulative) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("Cumulative.Propagate: nil solver")
	}
	n := len(c.starts)
	if n == 0 {
		return state, nil
	}

	// Gather current domains and quick checks.
	domains := make([]Domain, n)
	est := make([]int, n) // earliest start
	lst := make([]int, n) // latest start
	maxEnd := 0
	for i, v := range c.starts {
		d := solver.GetDomain(state, v.ID())
		if d == nil {
			return nil, fmt.Errorf("Cumulative: variable %d has nil domain", v.ID())
		}
		if d.Count() == 0 {
			return nil, fmt.Errorf("Cumulative: variable %d has empty domain", v.ID())
		}
		domains[i] = d
		est[i] = d.Min()
		lst[i] = d.Max()
		end := lst[i] + c.durations[i] - 1
		if end > maxEnd {
			maxEnd = end
		}
	}

	// Build resource profile from compulsory parts.
	// profile[t] sums demands of tasks that must execute at time t.
	// We index time from 1..maxEnd inclusive; index 0 unused.
	if maxEnd < 1 {
		return state, nil // degenerate, but nothing to do
	}
	profile := make([]int, maxEnd+1)
	// Record each task's compulsory part for later self-load adjustment during pruning
	cpStart := make([]int, n)
	cpEnd := make([]int, n)
	for i := 0; i < n; i++ {
		// Compulsory part exists when lst <= est+dur-1
		cpStart[i] = lst[i]
		cpEnd[i] = est[i] + c.durations[i] - 1
		if cpStart[i] <= cpEnd[i] {
			// Clamp within [1..maxEnd]
			startT := cpStart[i]
			endT := cpEnd[i]
			if startT < 1 {
				startT = 1
			}
			if endT > maxEnd {
				endT = maxEnd
			}
			if c.demands[i] > 0 {
				for t := startT; t <= endT; t++ {
					profile[t] += c.demands[i]
					if profile[t] > c.capacity {
						return nil, fmt.Errorf("Cumulative: capacity exceeded at t=%d (profile=%d > %d)", t, profile[t], c.capacity)
					}
				}
			}
		}
	}

	// Prune start domains where placing a task would exceed capacity
	// at any time covered by the task.
	newState := state
	for i, v := range c.starts {
		if c.demands[i] == 0 {
			// Zero-demand task never affects capacity; skip.
			continue
		}
		orig := domains[i]
		values := orig.ToSlice()
		if len(values) == 0 {
			return nil, fmt.Errorf("Cumulative: variable %d has empty domain", v.ID())
		}
		allowed := make([]int, 0, len(values))
		dur := c.durations[i]
		dem := c.demands[i]
		for _, sVal := range values {
			startT := sVal
			endT := sVal + dur - 1
			// If any covered t would exceed capacity, forbid this start.
			ok := true
			// Clamp to profile bounds; if end beyond maxEnd it is still safe to check
			// overlapping part within [1..maxEnd]. Times beyond maxEnd have implicit
			// profile 0 and thus cannot exceed capacity when adding dem.
			tStart := startT
			if tStart < 1 {
				tStart = 1
			}
			tEnd := endT
			if tEnd > maxEnd {
				tEnd = maxEnd
			}
			for t := tStart; t <= tEnd; t++ {
				load := profile[t]
				// Avoid double-counting this task's own compulsory load
				if cpStart[i] <= t && t <= cpEnd[i] {
					load -= dem
				}
				if load+dem > c.capacity {
					ok = false
					break
				}
			}
			if ok {
				allowed = append(allowed, sVal)
			}
		}

		if len(allowed) == 0 {
			return nil, fmt.Errorf("Cumulative: variable %d domain empty after pruning", v.ID())
		}
		if len(allowed) < orig.Count() {
			newDom := NewBitSetDomainFromValues(orig.MaxValue(), allowed)
			var changed bool
			newState, changed = solver.SetDomain(newState, v.ID(), newDom)
			if changed {
				domains[i] = newDom
			}
		}
	}

	return newState, nil
}
