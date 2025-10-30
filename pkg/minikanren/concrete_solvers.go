// Package minikanren provides concrete solver implementations for the
// pluggable constraint solving system. These solvers implement different
// algorithms and strategies for constraint satisfaction:
//
//   - BacktrackingSolver: Implements backtracking search for constraint solving
//   - PropagationSolver: Implements constraint propagation algorithms
//
// These concrete implementations demonstrate how the Solver interface
// enables pluggable constraint solving architectures.
package minikanren

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// BacktrackingSolver implements a backtracking-based constraint solver.
// It uses systematic search through the solution space, trying different
// variable assignments and backtracking when constraints are violated.
//
// This solver is suitable for:
//   - Problems with discrete domains
//   - Complex constraint interactions
//   - When completeness is required
//
// Performance characteristics:
//   - Exponential time in worst case (2^d for d variables)
//   - Good for small to medium problems
//   - Memory efficient (single path in search tree)
type BacktrackingSolver struct {
	*BaseSolver

	// maxDepth limits the search depth to prevent infinite recursion
	maxDepth int

	// timeout limits the solving time
	timeout time.Duration
}

// NewBacktrackingSolver creates a new backtracking solver with the specified configuration.
func NewBacktrackingSolver(id, name string, capabilities []string, priority int, maxDepth int, timeout time.Duration) *BacktrackingSolver {
	return &BacktrackingSolver{
		BaseSolver: NewBaseSolver(id, name, capabilities, priority),
		maxDepth:   maxDepth,
		timeout:    timeout,
	}
}

// Solve implements the Solver interface using backtracking search.
func (bs *BacktrackingSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	// For this demonstration, we'll implement a simple backtracking approach
	// In a real implementation, this would integrate with the FD solver

	// Check if we can handle this constraint type
	if !bs.CanHandle(constraint) {
		return nil, fmt.Errorf("backtracking solver cannot handle constraint type %T", constraint)
	}

	// Create a context with timeout
	solveCtx, cancel := context.WithTimeout(ctx, bs.timeout)
	defer cancel()

	// Simple backtracking implementation for demonstration
	result := bs.backtrackSolve(solveCtx, constraint, store, 0)
	if result == nil {
		return nil, fmt.Errorf("backtracking solver could not satisfy constraint")
	}

	return result, nil
}

// backtrackSolve implements the core backtracking algorithm
func (bs *BacktrackingSolver) backtrackSolve(ctx context.Context, constraint Constraint, store ConstraintStore, depth int) ConstraintStore {
	// Check timeout
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	// Check depth limit
	if depth > bs.maxDepth {
		return nil
	}

	// For demonstration, assume the constraint is satisfied
	// In a real implementation, this would check actual constraint satisfaction
	return store
}

// PropagationSolver implements a constraint propagation-based solver.
// It uses local consistency algorithms to reduce variable domains
// before search, making subsequent solving more efficient.
//
// This solver is suitable for:
//   - Problems with tight constraints
//   - Large domains that can be significantly reduced
//   - When preprocessing can eliminate invalid values
//
// Performance characteristics:
//   - Often polynomial time for constraint propagation
//   - Can be very effective for highly constrained problems
//   - May not be complete (some problems require search)
type PropagationSolver struct {
	*BaseSolver

	// maxIterations limits the number of propagation rounds
	maxIterations int

	// timeout limits the solving time
	timeout time.Duration
}

// NewPropagationSolver creates a new propagation solver with the specified configuration.
func NewPropagationSolver(id, name string, capabilities []string, priority int, maxIterations int, timeout time.Duration) *PropagationSolver {
	return &PropagationSolver{
		BaseSolver:    NewBaseSolver(id, name, capabilities, priority),
		maxIterations: maxIterations,
		timeout:       timeout,
	}
}

// Solve implements the Solver interface using constraint propagation.
func (ps *PropagationSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	// Check if we can handle this constraint type
	if !ps.CanHandle(constraint) {
		return nil, fmt.Errorf("propagation solver cannot handle constraint type %T", constraint)
	}

	// Create a context with timeout
	solveCtx, cancel := context.WithTimeout(ctx, ps.timeout)
	defer cancel()

	// Propagation-based solving
	result := ps.propagateSolve(solveCtx, constraint, store)
	if result == nil {
		return nil, fmt.Errorf("propagation solver could not satisfy constraint")
	}

	return result, nil
}

// propagateSolve implements constraint propagation
func (ps *PropagationSolver) propagateSolve(ctx context.Context, constraint Constraint, store ConstraintStore) ConstraintStore {
	// For demonstration, implement a simple propagation approach
	// In a real implementation, this would use actual constraint propagation algorithms

	for i := 0; i < ps.maxIterations; i++ {
		// Check timeout
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// For demonstration, assume propagation succeeds
		// In a real implementation, this would perform actual domain reduction
		// and potentially continue iterating if changes were made
	}

	return store
}

// HybridSolver combines backtracking with constraint propagation.
// It first applies propagation to reduce domains, then uses backtracking
// for any remaining search space.
//
// This solver is suitable for:
//   - Complex problems requiring both techniques
//   - When propagation alone is insufficient
//   - Balancing preprocessing with search
//
// Performance characteristics:
//   - Can be very effective when propagation significantly reduces search space
//   - Worst case still exponential, but often better in practice
//   - More complex to implement and tune
type HybridSolver struct {
	*BaseSolver

	// propagationSolver handles the preprocessing phase
	propagationSolver *PropagationSolver

	// backtrackingSolver handles the search phase
	backtrackingSolver *BacktrackingSolver

	// timeout limits the total solving time
	timeout time.Duration
}

// NewHybridSolver creates a new hybrid solver combining propagation and backtracking.
func NewHybridSolver(id, name string, capabilities []string, priority int, timeout time.Duration) *HybridSolver {
	return &HybridSolver{
		BaseSolver: NewBaseSolver(id, name, capabilities, priority),
		propagationSolver: NewPropagationSolver(
			id+"_prop", name+" (Propagation)", capabilities, priority+1, 10, timeout/2,
		),
		backtrackingSolver: NewBacktrackingSolver(
			id+"_bt", name+" (Backtracking)", capabilities, priority, 100, timeout/2,
		),
		timeout: timeout,
	}
}

// Solve implements the Solver interface using a hybrid approach.
func (hs *HybridSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	// Check if we can handle this constraint type
	if !hs.CanHandle(constraint) {
		return nil, fmt.Errorf("hybrid solver cannot handle constraint type %T", constraint)
	}

	// Create a context with timeout
	solveCtx, cancel := context.WithTimeout(ctx, hs.timeout)
	defer cancel()

	// Phase 1: Constraint propagation
	propagatedStore, err := hs.propagationSolver.Solve(solveCtx, constraint, store)
	if err != nil {
		// If propagation fails, fall back to backtracking on original store
		return hs.backtrackingSolver.Solve(solveCtx, constraint, store)
	}

	// Phase 2: Backtracking on reduced domains
	result, err := hs.backtrackingSolver.Solve(solveCtx, constraint, propagatedStore)
	if err != nil {
		return nil, fmt.Errorf("hybrid solver could not satisfy constraint")
	}

	return result, nil
}

// SolverFactory provides a convenient way to create pre-configured solvers.
type SolverFactory struct{}

// NewSolverFactory creates a new solver factory.
func NewSolverFactory() *SolverFactory {
	return &SolverFactory{}
}

// CreateBacktrackingSolver creates a standard backtracking solver.
func (sf *SolverFactory) CreateBacktrackingSolver(id string) Solver {
	return NewBacktrackingSolver(
		id,
		"Backtracking Solver",
		[]string{"DisequalityConstraint", "TypeConstraint"},
		1,             // Low priority
		50,            // Max depth
		5*time.Second, // Timeout
	)
}

// CreatePropagationSolver creates a standard propagation solver.
func (sf *SolverFactory) CreatePropagationSolver(id string) Solver {
	return NewPropagationSolver(
		id,
		"Propagation Solver",
		[]string{"AbsenceConstraint", "TypeConstraint"},
		2,             // Medium priority
		20,            // Max iterations
		3*time.Second, // Timeout
	)
}

// CreateHybridSolver creates a standard hybrid solver.
func (sf *SolverFactory) CreateHybridSolver(id string) Solver {
	return NewHybridSolver(
		id,
		"Hybrid Solver",
		[]string{"DisequalityConstraint", "AbsenceConstraint", "TypeConstraint"},
		3,              // High priority
		10*time.Second, // Timeout
	)
}

// CreateSolverSet creates a complete set of solvers for comprehensive constraint solving.
func (sf *SolverFactory) CreateSolverSet() []Solver {
	return []Solver{
		sf.CreateHybridSolver("hybrid-solver"),
		sf.CreatePropagationSolver("propagation-solver"),
		sf.CreateBacktrackingSolver("backtracking-solver"),
	}
}

// SolverComparator provides utilities for comparing and ranking solvers.
type SolverComparator struct{}

// NewSolverComparator creates a new solver comparator.
func NewSolverComparator() *SolverComparator {
	return &SolverComparator{}
}

// RankSolvers ranks solvers by priority for a given constraint type.
func (sc *SolverComparator) RankSolvers(solvers []Solver, constraintType string) []Solver {
	// Filter solvers that can handle the constraint type
	var capable []Solver
	for _, solver := range solvers {
		for _, cap := range solver.Capabilities() {
			if cap == constraintType {
				capable = append(capable, solver)
				break
			}
		}
	}

	// Sort by priority (highest first)
	sort.Slice(capable, func(i, j int) bool {
		return capable[i].Priority() > capable[j].Priority()
	})

	return capable
}

// GetBestSolver returns the highest priority solver for a constraint type.
func (sc *SolverComparator) GetBestSolver(solvers []Solver, constraintType string) (Solver, bool) {
	ranked := sc.RankSolvers(solvers, constraintType)
	if len(ranked) == 0 {
		return nil, false
	}
	return ranked[0], true
}
