// Package minikanren provides constraint solving infrastructure.
// This file implements the core solver with efficient copy-on-write state management
// for lock-free parallel search.
//
// # Architecture Overview
//
// The solver separates immutable problem definition from mutable solving state:
//
//	Model (immutable during solving):
//	  - Variables with initial domains
//	  - Constraints that reference variables
//	  - Configuration (heuristics, etc.)
//	  - Shared by all parallel workers (zero copy cost)
//
//	SolverState (mutable, copy-on-write):
//	  - Sparse chain of domain modifications
//	  - Each worker maintains its own independent chain
//	  - O(1) cost to create new state node
//	  - Pooled for zero GC pressure
//
// # How Constraint Propagation Works
//
// Constraints need to communicate domain changes. This happens via the SolverState:
//
//  1. Constraint reads current domains: GetDomain(state, varID)
//  2. Constraint computes domain reduction
//  3. Constraint creates new state: SetDomain(state, varID, newDomain)
//  4. Process repeats until fixed point
//
// Example with AllDifferent(x, y, z):
//
//	Initial:  x={1,2,3}, y={1,2,3}, z={1,2,3}
//	Assign:   x := 1  → State1: x={1}
//	Propagate: Remove 1 from y → State2: y={2,3} (parent: State1)
//	Propagate: Remove 1 from z → State3: z={2,3} (parent: State2)
//	Fixed point reached
//
// Each state node is tiny (40 bytes) and creation is O(1). Backtracking just
// discards state nodes. Parallel workers share the Model but have independent
// state chains, enabling lock-free search.
package minikanren

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Solver performs backtracking search to find solutions to constraint satisfaction problems.
// The solver implements:
//   - Efficient copy-on-write state management using sparse state representation
//   - Lock-free parallel search (future enhancement)
//   - Configurable variable and value ordering heuristics
//   - AC-3 style constraint propagation
//   - Smart backtracking with conflict-driven learning (future)
//
// The solver is designed for both sequential and parallel execution.
// State is immutable during search, with modifications creating lightweight
// derived states that share structure with their parent.
//
// Thread safety: Solver instances are NOT thread-safe. For parallel search,
// create multiple Solver instances that share the same immutable Model but
// maintain independent SolverState chains. This is zero-cost as the Model
// is read-only and domains are immutable.
type Solver struct {
	// model is the CSP being solved (read-only during search, shared by all workers)
	model *Model

	// config holds solver configuration and heuristics
	config *SolverConfig

	// statePool manages allocation of solver states for reuse
	statePool *sync.Pool

	// monitor tracks solving statistics (optional)
	monitor *SolverMonitor

	// baseState caches the last root-level propagated state from Solve.
	// When present, GetDomain(nil, varID) will read domains from this state
	// rather than the model's initial domains, allowing callers to inspect
	// propagation effects without threading SolverState explicitly.
	baseState *SolverState
}

// SolverState represents the mutable state of the solver at a point in search.
// States are organized as a persistent data structure where each state consists of:
//   - A pointer to the parent state
//   - The single domain that was modified from the parent
//   - The variable ID that was modified
//
// This sparse representation makes "copying" state at each search node O(1)
// instead of O(n), enabling efficient parallel search.
//
// How constraint propagation works across this architecture:
//
//	Model (immutable, shared):
//	  Variables: [x, y, z] with initial domains
//	  Constraints: [AllDifferent(x,y,z)]
//
//	Worker State Chain:
//	  State3 -> x={5}     (parent: State2)
//	  State2 -> y={2,3}   (parent: State1)
//	  State1 -> z={1,2,3} (parent: nil)
//
//	Propagation Example:
//	  1. Constraint sees x={5} via GetDomain(State3, x.ID)
//	  2. Constraint narrows y: y={2,3} (remove 5)
//	  3. Creates State4: y={2,3} (parent: State3)
//	  4. Constraint narrows z: z={1,2,3} (5 not present, no change)
//	  5. Returns State4 (fixed point reached)
//
// Constraints "communicate" by reading current domains via GetDomain and
// creating new states via SetDomain. The state chain captures all changes.
//
// States are pooled and reused to minimize GC pressure.
type SolverState struct {
	// parent points to the previous state (nil for root)
	parent *SolverState

	// modifiedVarID is the ID of the variable whose domain changed
	modifiedVarID int

	// modifiedDomain is the new domain for the modified variable
	modifiedDomain Domain

	// depth tracks the depth in the search tree for heuristics
	depth int

	// refCount tracks the number of active references to this state node.
	// In sequential search this is typically 1 and ReleaseState will cascade,
	// in parallel search multiple workers may hold references simultaneously.
	// When the count drops to zero, the node can be safely returned to the pool.
	refCount atomic.Int64
}

// NewSolver creates a solver for the given model.
// The model should be fully constructed before creating the solver.
func NewSolver(model *Model) *Solver {
	return &Solver{
		model:  model,
		config: model.Config(),
		statePool: &sync.Pool{
			New: func() interface{} {
				return &SolverState{}
			},
		},
	}
}

// NewSolverWithConfig creates a solver with custom configuration that overrides model config.
func NewSolverWithConfig(model *Model, config *SolverConfig) *Solver {
	if config == nil {
		config = model.Config()
	}
	return &Solver{
		model:  model,
		config: config,
		statePool: &sync.Pool{
			New: func() interface{} {
				return &SolverState{}
			},
		},
	}
}

// SetMonitor enables statistics collection during solving.
func (s *Solver) SetMonitor(monitor *SolverMonitor) {
	s.monitor = monitor
}

// GetDomain returns the current domain of a variable in the given state.
// Walks the state chain to find the most recent domain for the variable.
// This is O(depth) in the worst case, but typically O(1) due to locality.
func (s *Solver) GetDomain(state *SolverState, varID int) Domain {
	// Walk the state chain looking for the most recent modification
	for current := state; current != nil; current = current.parent {
		if current.modifiedVarID == varID && current.modifiedDomain != nil {
			return current.modifiedDomain
		}
	}

	// If no explicit state is provided, try the cached base propagated state
	if state == nil && s.baseState != nil {
		for current := s.baseState; current != nil; current = current.parent {
			if current.modifiedVarID == varID && current.modifiedDomain != nil {
				return current.modifiedDomain
			}
		}
	}

	// No modification found, return original domain from model
	if varID >= 0 && varID < len(s.model.variables) {
		return s.model.variables[varID].Domain()
	}

	return nil
}

// SetDomain creates a new state with an updated domain for the specified variable.
// Returns the new state and a boolean indicating if the domain actually changed.
// If the domain is identical to the current domain, returns the original state
// and false to avoid unnecessary propagation.
//
// This is an O(1) operation for the state update, plus O(domain size) for equality check.
// The returned state should replace the current state in the search.
func (s *Solver) SetDomain(state *SolverState, varID int, domain Domain) (*SolverState, bool) {
	// Check if domain actually changed
	currentDomain := s.GetDomain(state, varID)
	if currentDomain.Equal(domain) {
		return state, false // No change, return original state
	}

	newState := s.statePool.Get().(*SolverState)
	// Initialize new state node
	newState.parent = state
	newState.modifiedVarID = varID
	newState.modifiedDomain = domain

	if state != nil {
		newState.depth = state.depth + 1
		// Retain parent since this child holds a reference to it
		state.refCount.Add(1)
	} else {
		newState.depth = 1
	}

	// New nodes start with a refCount of 1 (owned by the caller)
	newState.refCount.Store(1)

	return newState, true
}

// propagate runs all propagation constraints to a fixed-point.
// Returns a new state with pruned domains, or error if inconsistency detected.
//
// The propagation loop:
//  1. Collect all PropagationConstraints from the model
//  2. Run each constraint once
//  3. If any constraint modified domains, repeat from step 2
//  4. Stop when no changes occur (fixed-point reached)
//
// This maintains copy-on-write semantics: each constraint returns a new state,
// preserving the lock-free property for parallel search.
func (s *Solver) propagate(state *SolverState) (*SolverState, error) {
	constraints := make([]PropagationConstraint, 0)

	// Collect PropagationConstraints from model
	for _, mc := range s.model.Constraints() {
		if pc, ok := mc.(PropagationConstraint); ok {
			constraints = append(constraints, pc)
		}
	}

	if len(constraints) == 0 {
		return state, nil // No propagation constraints
	}

	currentState := state
	maxIterations := 1000 // Prevent infinite loops

	for iteration := 0; iteration < maxIterations; iteration++ {
		changed := false

		for _, constraint := range constraints {
			newState, err := constraint.Propagate(s, currentState)
			if err != nil {
				if s.monitor != nil {
					s.monitor.RecordBacktrack()
				}
				return nil, err // Inconsistency detected
			}

			// Check if state changed (domain pruning occurred)
			if newState != currentState {
				changed = true
				currentState = newState
			}
		}

		if !changed {
			// Fixed-point reached
			return currentState, nil
		}
	}

	// Should never reach here unless there's a bug in constraints
	return nil, fmt.Errorf("propagation failed to reach fixed-point after %d iterations", maxIterations)
}

// ReleaseState returns a state to the pool for reuse.
// Should be called when backtracking to free memory.
// Only the state itself is pooled, not domains (they are immutable and shared).
func (s *Solver) ReleaseState(state *SolverState) {
	// Cascade release: decrement refCount; if it hits zero, release this node
	// and continue to parent (which this node retained at creation time).
	for cur := state; cur != nil; {
		// Nothing to do for nil
		// Decrement and check if others still hold references
		if cur.refCount.Add(-1) > 0 {
			return // someone else still holds this node
		}

		// No more references: capture parent and clear this node before pooling
		parent := cur.parent

		// Clear references to allow GC and avoid accidental reuse races
		cur.parent = nil
		cur.modifiedDomain = nil
		cur.modifiedVarID = 0
		cur.depth = 0
		cur.refCount.Store(0)

		s.statePool.Put(cur)

		// Continue to parent (which we retained when creating cur)
		cur = parent
	}
}

// Solve finds solutions to the constraint satisfaction problem.
// Returns up to maxSolutions solutions, or all solutions if maxSolutions <= 0.
// Solutions are returned as slices of integers, one per variable in order.
//
// The search can be cancelled via the context, enabling timeouts and cancellation.
func (s *Solver) Solve(ctx context.Context, maxSolutions int) ([][]int, error) {
	// Validate model before solving
	if err := s.model.Validate(); err != nil {
		return nil, fmt.Errorf("invalid model: %w", err)
	}

	if s.monitor != nil {
		defer s.monitor.FinishSearch()
	}

	// Initialize search with empty state (all variables at model domains)
	initialState := (*SolverState)(nil)

	// Perform initial constraint propagation
	propagatedState, err := s.propagate(initialState)
	if err != nil {
		// Root-level inconsistency: no solutions exist; return empty result set
		if s.monitor != nil {
			s.monitor.EndPropagation()
		}
		return [][]int{}, nil
	}

	// Cache the root-level propagated state for later inspection via GetDomain(nil, id)
	// Retain an extra reference so search/backtracking won't release it.
	s.baseState = propagatedState
	if s.baseState != nil {
		s.baseState.refCount.Add(1)
	}

	if s.monitor != nil {
		s.monitor.EndPropagation()
	}

	// Check for early termination
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check if already solved after propagation
	if s.isComplete(propagatedState) {
		solution := s.extractSolution(propagatedState)
		if s.monitor != nil {
			s.monitor.RecordSolution()
		}
		return [][]int{solution}, nil
	}

	// Perform backtracking search
	solutions := make([][]int, 0)
	s.search(ctx, propagatedState, &solutions, maxSolutions)

	return solutions, ctx.Err()
}

// search performs iterative deepening backtracking search.
// Uses an explicit stack to avoid deep recursion and enable better control.
func (s *Solver) search(ctx context.Context, state *SolverState, solutions *[][]int, maxSolutions int) {
	// Stack-based iterative search
	type searchFrame struct {
		state      *SolverState
		varID      int
		values     []int
		valueIndex int
	}

	stack := make([]*searchFrame, 0, 100)

	// Select initial variable and values
	varID, values := s.selectVariable(state)
	if varID == -1 {
		// No unbound variables, we have a solution
		if s.isComplete(state) {
			solution := s.extractSolution(state)
			*solutions = append(*solutions, solution)
			if s.monitor != nil {
				s.monitor.RecordSolution()
			}
		}
		return
	}

	stack = append(stack, &searchFrame{
		state:      state,
		varID:      varID,
		values:     values,
		valueIndex: 0,
	})

	for len(stack) > 0 {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame := stack[len(stack)-1]

		// Check if we've tried all values for this variable
		if frame.valueIndex >= len(frame.values) {
			// Backtrack
			s.ReleaseState(frame.state)
			stack = stack[:len(stack)-1]

			if s.monitor != nil {
				s.monitor.RecordBacktrack()
			}

			continue
		}

		if s.monitor != nil {
			s.monitor.RecordNode()
			s.monitor.RecordDepth(len(stack))
		}

		// Try next value
		value := frame.values[frame.valueIndex]
		frame.valueIndex++

		// Create new state with variable assigned to value
		domain := s.GetDomain(frame.state, frame.varID)
		newDomain := NewBitSetDomainFromValues(domain.MaxValue(), []int{value})
		newState, _ := s.SetDomain(frame.state, frame.varID, newDomain)

		// Propagate constraints
		propagatedState, err := s.propagate(newState)
		if err != nil {
			// Propagation failed, try next value
			s.ReleaseState(newState)
			continue
		}

		// Check if complete
		if s.isComplete(propagatedState) {
			solution := s.extractSolution(propagatedState)
			*solutions = append(*solutions, solution)

			if s.monitor != nil {
				s.monitor.RecordSolution()
			}

			s.ReleaseState(propagatedState)

			// Check if we've found enough solutions
			if maxSolutions > 0 && len(*solutions) >= maxSolutions {
				return
			}

			continue
		}

		// Select next variable
		nextVarID, nextValues := s.selectVariable(propagatedState)
		if nextVarID == -1 {
			// No unbound variables but not complete means failure
			s.ReleaseState(propagatedState)
			continue
		}

		// Push new frame
		stack = append(stack, &searchFrame{
			state:      propagatedState,
			varID:      nextVarID,
			values:     nextValues,
			valueIndex: 0,
		})
	}
}

// propagate applies constraint propagation to the current state.
// Returns a new state with reduced domains, or an error if inconsistency detected.
//
// Constraint propagation works as follows:
// 1. Constraints are stored in the immutable Model
// 2. Current domains are retrieved from SolverState via GetDomain(state, varID)
// 3. When a constraint narrows a domain, SetDomain creates a new state node
// 4. Propagation continues until a fixed point (no more changes)
//
// This architecture enables:
// - Constraints to "communicate" via the shared propagation queue
// - Multiple workers to propagate independently (different state chains)
// - Efficient backtracking (just pop state nodes)
// Note: The actual propagate implementation is defined earlier in this file,
// after SetDomain. It runs all PropagationConstraints to a fixed-point.

// isComplete returns true if all variables are bound (singleton domains).
func (s *Solver) isComplete(state *SolverState) bool {
	for i := 0; i < s.model.VariableCount(); i++ {
		domain := s.GetDomain(state, i)
		if !domain.IsSingleton() {
			return false
		}
	}
	return true
}

// extractSolution extracts the variable assignments from a complete state.
func (s *Solver) extractSolution(state *SolverState) []int {
	solution := make([]int, s.model.VariableCount())
	for i := 0; i < s.model.VariableCount(); i++ {
		domain := s.GetDomain(state, i)
		if domain.IsSingleton() {
			solution[i] = domain.SingletonValue()
		} else {
			solution[i] = 0 // Shouldn't happen in complete state
		}
	}
	return solution
}

// selectVariable chooses the next variable to branch on using the configured heuristic.
// Returns the variable ID and the ordered list of values to try.
// Returns (-1, nil) if all variables are bound.
func (s *Solver) selectVariable(state *SolverState) (int, []int) {
	bestVar := -1
	bestScore := float64(-1)
	var bestValues []int

	for i := 0; i < s.model.VariableCount(); i++ {
		domain := s.GetDomain(state, i)
		if domain.IsSingleton() {
			continue // Variable already bound
		}

		score := s.computeVariableScore(i, domain)
		if bestVar == -1 || score < bestScore {
			bestVar = i
			bestScore = score
			bestValues = make([]int, 0, domain.Count())
			domain.IterateValues(func(v int) {
				bestValues = append(bestValues, v)
			})
		}
	}

	if bestVar == -1 {
		return -1, nil
	}

	// Order values according to heuristic
	orderedValues := s.orderValues(bestValues)

	return bestVar, orderedValues
}

// computeVariableScore computes a score for variable selection heuristics.
// Lower scores are better (selected first).
func (s *Solver) computeVariableScore(varID int, domain Domain) float64 {
	switch s.config.VariableHeuristic {
	case HeuristicDom:
		// Smallest domain first
		return float64(domain.Count())

	case HeuristicDomDeg:
		// Domain size divided by degree (constraint count)
		degree := s.computeVariableDegree(varID)
		return float64(domain.Count()) / float64(1+degree)

	case HeuristicDeg:
		// Highest degree first (most constrained)
		degree := s.computeVariableDegree(varID)
		return -float64(degree) // Negative to prefer higher degree

	case HeuristicLex:
		// Lexicographic order (variable ID)
		return float64(varID)

	default:
		// Default to smallest domain
		return float64(domain.Count())
	}
}

// computeVariableDegree returns the number of constraints involving the variable.
func (s *Solver) computeVariableDegree(varID int) int {
	degree := 0
	for _, constraint := range s.model.Constraints() {
		for _, v := range constraint.Variables() {
			if v.ID() == varID {
				degree++
				break
			}
		}
	}
	return degree
}

// orderValues orders domain values according to the configured heuristic.
func (s *Solver) orderValues(values []int) []int {
	// For now, just return values as-is (ascending order)
	// TODO: Implement value ordering heuristics
	return values
}
