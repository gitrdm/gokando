package minikanren

import (
	"context"
	"errors"
	"math"
	"time"
)

// OptimizeOption configures SolveOptimalWithOptions behavior.
// Use helpers like WithTimeLimit, WithNodeLimit, WithTargetObjective, WithParallelWorkers,
// and WithHeuristics to customize the search.
type OptimizeOption func(*optConfig)

type optConfig struct {
	timeLimit       time.Duration
	nodeLimit       int
	targetObjective *int
	parallelWorkers int
	// Optional heuristic overrides for this solve call only
	varHeuristic   *VariableOrderingHeuristic
	valueHeuristic *ValueOrderingHeuristic
	randomSeed     *int64
}

// WithTimeLimit sets a hard time limit for the optimization. When reached,
// the best incumbent is returned together with context.DeadlineExceeded.
func WithTimeLimit(d time.Duration) OptimizeOption {
	return func(c *optConfig) { c.timeLimit = d }
}

// WithNodeLimit limits the number of search node expansions. When reached,
// the best incumbent is returned together with ErrSearchLimitReached.
func WithNodeLimit(n int) OptimizeOption {
	return func(c *optConfig) { c.nodeLimit = n }
}

// WithTargetObjective requests early exit as soon as a solution with objective == target is found.
func WithTargetObjective(target int) OptimizeOption {
	return func(c *optConfig) { c.targetObjective = &target }
}

// WithParallelWorkers enables parallel branch-and-bound using the shared work-queue
// infrastructure. Values <= 1 select sequential mode.
func WithParallelWorkers(workers int) OptimizeOption {
	return func(c *optConfig) { c.parallelWorkers = workers }
}

// WithHeuristics overrides variable/value ordering heuristics for this solve call only.
func WithHeuristics(v VariableOrderingHeuristic, val ValueOrderingHeuristic, seed int64) OptimizeOption {
	return func(c *optConfig) {
		c.varHeuristic = &v
		c.valueHeuristic = &val
		c.randomSeed = &seed
	}
}

// ErrSearchLimitReached indicates an optimization run terminated due to a configured search limit
// (e.g., node limit). The returned incumbent is valid but optimality may not be proven.
var ErrSearchLimitReached = errors.New("search limit reached")

// SolveOptimal finds a solution that optimizes the given objective variable.
//
// Contract:
//   - obj is an FD variable participating in the model. Its domain encodes the
//     objective value (smaller is better when minimize=true).
//   - minimize selects the direction (true: minimize, false: maximize).
//   - On success, returns the best solution found (values for all model variables
//     in model order) and the objective value. If the model is infeasible, returns
//     (nil, 0, nil). If ctx is cancelled, returns the best incumbent if any
//     together with ctx.Err().
//
// Implementation notes:
//   - This is a native branch-and-bound layered on the existing FD solver. It
//     reuses propagation and branching; adds a fast admissible bound check and an
//     incumbent cutoff applied as a dynamic constraint on the objective domain.
//   - Lower bound (LB) for minimize is obj.Min() from the current state; for
//     maximize, the symmetric upper bound is used.
//   - Incumbent cutoff is injected by tightening the objective domain at nodes:
//     minimize: obj ≤ (best-1)  via RemoveAtOrAbove(best)
//     maximize: obj ≥ (best+1)  via RemoveAtOrBelow(best)
func (s *Solver) SolveOptimal(ctx context.Context, obj *FDVariable, minimize bool) ([]int, int, error) {
	return s.SolveOptimalWithOptions(ctx, obj, minimize)
}

// SolveOptimalWithOptions is like SolveOptimal but supports options (time/node limits,
// target objective, heuristic overrides, and parallel workers).
func (s *Solver) SolveOptimalWithOptions(ctx context.Context, obj *FDVariable, minimize bool, opts ...OptimizeOption) ([]int, int, error) {
	cfg := &optConfig{}
	for _, o := range opts {
		if o != nil {
			o(cfg)
		}
	}

	// Optional deadline
	if cfg.timeLimit > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeLimit)
		defer cancel()
	}

	// Temporary heuristic override (restore after)
	orig := *s.config
	if cfg.varHeuristic != nil {
		s.config.VariableHeuristic = *cfg.varHeuristic
	}
	if cfg.valueHeuristic != nil {
		s.config.ValueHeuristic = *cfg.valueHeuristic
	}
	if cfg.randomSeed != nil {
		s.config.RandomSeed = *cfg.randomSeed
	}
	defer func() { *s.config = orig }()

	// Parallel mode if requested
	if cfg.parallelWorkers > 1 {
		return s.solveOptimalParallel(ctx, obj, minimize, cfg)
	}

	// Validate model first
	if err := s.model.Validate(); err != nil {
		return nil, 0, err
	}

	// Initial propagation
	state, err := s.propagate(nil)
	if err != nil {
		// Root-level inconsistency → no solutions
		return nil, 0, nil
	}

	// Early exit if solved by propagation
	if s.isComplete(state) {
		sol := s.extractSolution(state)
		// Objective must be singleton here; read from domain
		od := s.GetDomain(state, obj.ID())
		if od == nil || od.Count() == 0 {
			// Defensive: inconsistent objective binding
			return nil, 0, nil
		}
		val := od.Min()
		if !minimize {
			val = od.Max()
		}
		return sol, val, nil
	}

	// Incumbent tracking
	var bestSol []int
	bestVal := 0
	haveIncumbent := false
	nodes := 0

	// Depth-first branch-and-bound search specialized for optimization
	type frame struct {
		state      *SolverState
		varID      int
		values     []int
		valueIndex int
	}

	// Helper to compute admissible bound from current state (may use structure-aware LB)
	bound := func(st *SolverState) (int, bool) {
		return s.computeObjectiveBound(st, obj, minimize)
	}

	// Helper to apply incumbent cutoff to objective domain on a given state
	applyCutoff := func(st *SolverState) *SolverState {
		if !haveIncumbent {
			return st
		}
		d := s.GetDomain(st, obj.ID())
		if d == nil || d.Count() == 0 {
			return st
		}
		var tightened Domain
		if minimize {
			// obj ≤ best-1 → remove values ≥ best
			tightened = d.RemoveAtOrAbove(bestVal)
		} else {
			// obj ≥ best+1 → remove values ≤ best
			tightened = d.RemoveAtOrBelow(bestVal)
		}
		if tightened.Equal(d) {
			return st
		}
		ns, _ := s.SetDomain(st, obj.ID(), tightened)
		return ns
	}

	// Initialize stack with the already propagated root state (with optional cutoff)
	root := applyCutoff(state)

	// Bound check at root
	if b, ok := bound(root); ok {
		if haveIncumbent {
			if minimize && b >= bestVal {
				return nil, 0, nil
			}
			if !minimize && b <= bestVal {
				return nil, 0, nil
			}
		}
	} else {
		return nil, 0, nil
	}

	// Variable/value selection
	varID, values := s.selectVariable(root)
	stack := make([]*frame, 0, 64)
	stack = append(stack, &frame{state: root, varID: varID, values: values, valueIndex: 0})

	// Optimization loop
	for len(stack) > 0 {
		// Cancellation
		select {
		case <-ctx.Done():
			if haveIncumbent {
				return bestSol, bestVal, ctx.Err()
			}
			return nil, 0, ctx.Err()
		default:
		}

		fr := stack[len(stack)-1]

		if fr.varID == -1 {
			// Leaf: all variables bound
			if s.isComplete(fr.state) {
				d := s.GetDomain(fr.state, obj.ID())
				if d != nil && d.IsSingleton() {
					val := d.SingletonValue()
					if !haveIncumbent || (minimize && val < bestVal) || (!minimize && val > bestVal) {
						bestVal = val
						bestSol = s.extractSolution(fr.state)
						haveIncumbent = true
					}
				}
			}
			// Backtrack
			s.ReleaseState(fr.state)
			stack = stack[:len(stack)-1]
			continue
		}

		// Exhausted values for this variable? backtrack
		if fr.valueIndex >= len(fr.values) {
			s.ReleaseState(fr.state)
			stack = stack[:len(stack)-1]
			continue
		}

		// Try next value
		value := fr.values[fr.valueIndex]
		fr.valueIndex++

		// Assign and propagate
		dom := s.GetDomain(fr.state, fr.varID)
		newDom := NewBitSetDomainFromValues(dom.MaxValue(), []int{value})
		child, _ := s.SetDomain(fr.state, fr.varID, newDom)

		// Apply incumbent cutoff on the child before propagation
		child = applyCutoff(child)

		propagated, err := s.propagate(child)
		if err != nil {
			// Inconsistent branch
			s.ReleaseState(child)
			continue
		}

		// Bound check for pruning
		if b, ok := bound(propagated); ok {
			if haveIncumbent {
				if minimize && b >= bestVal {
					s.ReleaseState(propagated)
					continue
				}
				if !minimize && b <= bestVal {
					s.ReleaseState(propagated)
					continue
				}
			}
		} else {
			s.ReleaseState(propagated)
			continue
		}

		// If complete, evaluate and possibly update incumbent, else descend
		if s.isComplete(propagated) {
			d := s.GetDomain(propagated, obj.ID())
			if d != nil && d.IsSingleton() {
				val := d.SingletonValue()
				if !haveIncumbent || (minimize && val < bestVal) || (!minimize && val > bestVal) {
					bestVal = val
					bestSol = s.extractSolution(propagated)
					haveIncumbent = true
					// Early-accept target objective if requested
					if cfg.targetObjective != nil && val == *cfg.targetObjective {
						s.ReleaseState(propagated)
						return bestSol, bestVal, nil
					}
				}
			}
			// Node accounting and early termination on limits after updating incumbent
			nodes++
			if cfg.nodeLimit > 0 && nodes >= cfg.nodeLimit {
				s.ReleaseState(propagated)
				return bestSol, bestVal, ErrSearchLimitReached
			}
			s.ReleaseState(propagated)
			continue
		}

		// Do not count non-leaf nodes toward the node limit; we count leaves only
		// to ensure we always have a chance to produce an incumbent.

		// Select next variable and push
		nid, nvals := s.selectVariable(propagated)
		stack = append(stack, &frame{state: propagated, varID: nid, values: nvals, valueIndex: 0})
	}

	if !haveIncumbent {
		return nil, 0, nil
	}
	return bestSol, bestVal, nil
}

// BestObjectiveValue computes a trivial admissible bound for the objective in the current state.
// It is a helper primarily for testing and documentation.
func (s *Solver) BestObjectiveValue(state *SolverState, obj *FDVariable, minimize bool) (int, bool) {
	if state == nil {
		// Use base state (post-root propagation) if available
		state = s.baseState
	}
	d := s.GetDomain(state, obj.ID())
	if d == nil || d.Count() == 0 {
		return 0, false
	}
	if minimize {
		return d.Min(), true
	}
	return d.Max(), true
}

// Infinity provides a sentinel large positive value for tests/examples when initializing bounds.
const Infinity = math.MaxInt32

// computeObjectiveBound computes a safe admissible bound for the objective based on
// the current state and known structural constraints. It falls back to the objective
// variable's domain when no better structural bound is available.
func (s *Solver) computeObjectiveBound(state *SolverState, obj *FDVariable, minimize bool) (int, bool) {
	if state == nil {
		state = s.baseState
	}
	// Strategy 1: Detect a LinearSum with total == obj to compute Σ a[i]*min/max(xi)
	for _, c := range s.model.Constraints() {
		if ls, ok := c.(*LinearSum); ok {
			if ls != nil && ls.total != nil && ls.total.ID() == obj.ID() {
				// Compute sum of coeffs times bound of xi
				sum := 0
				for i, v := range ls.vars {
					d := s.GetDomain(state, v.ID())
					if d == nil || d.Count() == 0 {
						return 0, false
					}
					if minimize {
						sum += ls.coeffs[i] * d.Min()
					} else {
						sum += ls.coeffs[i] * d.Max()
					}
				}
				return sum, true
			}
		}
	}
	// Fallback: use the objective variable domain directly
	d := s.GetDomain(state, obj.ID())
	if d == nil || d.Count() == 0 {
		return 0, false
	}
	if minimize {
		return d.Min(), true
	}
	return d.Max(), true
}
