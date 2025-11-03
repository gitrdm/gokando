// Package minikanren implements global constraints for finite-domain CP.
//
// This file provides a production implementation of the Global Cardinality
// Constraint (GCC). GCC bounds how many times each value can occur among a
// collection of variables. It is commonly used for assignment and scheduling
// models where per-value capacities must be respected.
//
// Contract:
//   - Variables X[0..n-1] each have a finite domain over positive integers
//   - We consider value set V = {1..M}, where M = max domain value across X
//   - For each v in V, we provide bounds minCount[v] and maxCount[v] with
//     0 <= minCount[v] <= maxCount[v]
//   - GCC enforces that, in any solution, the number of variables assigned
//     to value v lies within [minCount[v], maxCount[v]] for all v in V.
//
// Propagation strength: bounds-consistent checks plus pruning for saturated values.
//   - Compute fixedCount[v]: number of variables already fixed to v
//   - Compute possibleCount[v]: number of variables whose domain contains v
//   - Fail if fixedCount[v] > maxCount[v] or possibleCount[v] < minCount[v]
//   - If fixedCount[v] == maxCount[v], remove v from all other variables
//
// While stronger GAC can be achieved with flow-based algorithms, this
// implementation is efficient, sound, and integrates cleanly with the solver's
// fixed-point loop. It detects overloads early and applies useful pruning when
// some values are saturated.
package minikanren

import "fmt"

// GlobalCardinality constrains occurrence counts per value across variables.
type GlobalCardinality struct {
	vars     []*FDVariable
	minCount []int // indexed by value (1..M); index 0 unused
	maxCount []int // indexed by value (1..M); index 0 unused
	maxValue int   // M
}

// NewGlobalCardinality constructs a GCC over vars with per-value min/max bounds.
// minCount and maxCount must be length >= M+1 where M is the maximum domain
// value across vars; indexes 1..M are used. For values not present, bounds may
// be zero.
func NewGlobalCardinality(vars []*FDVariable, minCount, maxCount []int) (PropagationConstraint, error) {
	if len(vars) == 0 {
		return nil, fmt.Errorf("GlobalCardinality requires at least one variable")
	}
	// Determine maxValue across variables
	M := 0
	for i, v := range vars {
		if v == nil {
			return nil, fmt.Errorf("GlobalCardinality: vars[%d] is nil", i)
		}
		if mv := v.Domain().MaxValue(); mv > M {
			M = mv
		}
	}
	if len(minCount) <= M || len(maxCount) <= M {
		return nil, fmt.Errorf("GlobalCardinality: min/max count slices must have length >= %d (got %d/%d)", M+1, len(minCount), len(maxCount))
	}
	// Validate bounds and aggregate feasibility
	totalMin, totalMax := 0, 0
	for v := 1; v <= M; v++ {
		if minCount[v] < 0 || maxCount[v] < 0 {
			return nil, fmt.Errorf("GlobalCardinality: negative bounds at value %d", v)
		}
		if minCount[v] > maxCount[v] {
			return nil, fmt.Errorf("GlobalCardinality: minCount[%d] > maxCount[%d]", v, v)
		}
		totalMin += minCount[v]
		totalMax += maxCount[v]
	}
	n := len(vars)
	if totalMin > n {
		return nil, fmt.Errorf("GlobalCardinality: sum(minCount)=%d > #vars=%d", totalMin, n)
	}
	if totalMax < n {
		return nil, fmt.Errorf("GlobalCardinality: sum(maxCount)=%d < #vars=%d", totalMax, n)
	}

	// Defensive copies up to M+1 entries
	mc := make([]int, M+1)
	xc := make([]int, M+1)
	copy(mc, minCount[:M+1])
	copy(xc, maxCount[:M+1])
	vs := make([]*FDVariable, len(vars))
	copy(vs, vars)

	return &GlobalCardinality{vars: vs, minCount: mc, maxCount: xc, maxValue: M}, nil
}

// Variables returns variables constrained by GCC.
func (g *GlobalCardinality) Variables() []*FDVariable { return g.vars }

// Type returns the constraint identifier.
func (g *GlobalCardinality) Type() string { return "GlobalCardinality" }

// String returns a readable description.
func (g *GlobalCardinality) String() string {
	return fmt.Sprintf("GCC(n=%d, values=1..%d)", len(g.vars), g.maxValue)
}

// Propagate performs bounds checks and removes saturated values from other domains.
func (g *GlobalCardinality) Propagate(solver *Solver, state *SolverState) (*SolverState, error) {
	if solver == nil {
		return nil, fmt.Errorf("GlobalCardinality.Propagate: nil solver")
	}
	n := len(g.vars)
	if n == 0 {
		return state, nil
	}

	// Build fixed and possible counts.
	fixed := make([]int, g.maxValue+1)
	possible := make([]int, g.maxValue+1)
	domains := make([]Domain, n)
	for i, v := range g.vars {
		d := solver.GetDomain(state, v.ID())
		if d == nil {
			return nil, fmt.Errorf("GCC: variable %d has nil domain", v.ID())
		}
		if d.Count() == 0 {
			return nil, fmt.Errorf("GCC: variable %d has empty domain", v.ID())
		}
		domains[i] = d
		if d.IsSingleton() {
			val := d.SingletonValue()
			if val >= 1 && val <= g.maxValue {
				fixed[val]++
				possible[val]++
			}
		} else {
			d.IterateValues(func(val int) {
				if val >= 1 && val <= g.maxValue {
					possible[val]++
				}
			})
		}
	}

	// Check bounds feasibility per value.
	for v := 1; v <= g.maxValue; v++ {
		if fixed[v] > g.maxCount[v] {
			return nil, fmt.Errorf("GCC: value %d fixed %d times exceeds max %d", v, fixed[v], g.maxCount[v])
		}
		if possible[v] < g.minCount[v] {
			return nil, fmt.Errorf("GCC: value %d possible %d less than min %d", v, possible[v], g.minCount[v])
		}
	}

	// Prune saturated values from other variables.
	newState := state
	for i, v := range g.vars {
		d := domains[i]
		if d.IsSingleton() {
			continue
		}
		// Collect allowed values excluding those saturated (fixed == max)
		allowed := make([]int, 0, d.Count())
		pruned := false
		d.IterateValues(func(val int) {
			if val >= 1 && val <= g.maxValue && fixed[val] >= g.maxCount[val] {
				// Already at max occurrences; unless this variable is already fixed to val (it isn't), prune it
				pruned = true
				return
			}
			allowed = append(allowed, val)
		})
		if pruned {
			if len(allowed) == 0 {
				return nil, fmt.Errorf("GCC: variable %d domain empty after pruning", v.ID())
			}
			newDom := NewBitSetDomainFromValues(d.MaxValue(), allowed)
			var changed bool
			newState, changed = solver.SetDomain(newState, v.ID(), newDom)
			if changed {
				domains[i] = newDom
			}
		}
	}
	return newState, nil
}
