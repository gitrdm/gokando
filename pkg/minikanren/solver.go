// Package minikanren provides the Solver interface and related types for
// the pluggable constraint solving system. Solvers implement different
// algorithms and strategies for constraint satisfaction, enabling:
//
//   - Multiple solving approaches (backtracking, propagation, etc.)
//   - Specialized solvers for different constraint types
//   - Performance optimization through solver selection
//   - Extensible architecture for custom solvers
//
// The solver interface provides a clean abstraction that allows the
// constraint manager to automatically route constraints to appropriate
// solvers based on type, performance characteristics, and availability.
package minikanren

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Solver represents a constraint solving algorithm or strategy.
// Solvers implement different approaches to constraint satisfaction,
// from simple backtracking to advanced propagation algorithms.
//
// Solvers must be:
//   - Thread-safe: Safe for concurrent use across multiple goroutines
//   - Idempotent: Multiple calls with the same input produce the same result
//   - Context-aware: Respect context cancellation for cooperative scheduling
//   - Error-resilient: Handle malformed constraints gracefully
type Solver interface {
	// ID returns a unique identifier for this solver instance.
	// Used for registration, metrics collection, and debugging.
	ID() string

	// Name returns a human-readable name for this solver.
	// Used in logging, metrics, and user-facing messages.
	Name() string

	// Capabilities returns the constraint types this solver can handle.
	// The constraint manager uses this for automatic solver selection.
	Capabilities() []string

	// Solve attempts to satisfy a constraint within the given constraint store.
	// Returns an updated constraint store if successful, or an error if the
	// constraint cannot be satisfied using this solver's algorithm.
	//
	// The solver may modify the input store or create a new one.
	// The context should be respected for cancellation and timeouts.
	Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error)

	// Priority returns the preference level for this solver.
	// Higher priority solvers are preferred when multiple solvers
	// can handle the same constraint type. Default priority is 0.
	Priority() int

	// CanHandle checks if this solver can handle a specific constraint.
	// More specific than Capabilities() - allows runtime constraint inspection.
	CanHandle(constraint Constraint) bool
}

// BaseSolver provides a basic implementation of the Solver interface.
// Custom solvers can embed this struct to inherit common functionality
// and only implement the specific solving logic.
type BaseSolver struct {
	// id uniquely identifies this solver
	id string

	// name is the human-readable solver name
	name string

	// capabilities lists constraint types this solver can handle
	capabilities []string

	// priority indicates solver preference (higher is better)
	priority int
}

// NewBaseSolver creates a new base solver with the specified configuration.
func NewBaseSolver(id, name string, capabilities []string, priority int) *BaseSolver {
	return &BaseSolver{
		id:           id,
		name:         name,
		capabilities: capabilities,
		priority:     priority,
	}
}

// ID returns the unique identifier for this solver.
// Implements the Solver interface.
func (bs *BaseSolver) ID() string {
	return bs.id
}

// Name returns the human-readable name for this solver.
// Implements the Solver interface.
func (bs *BaseSolver) Name() string {
	return bs.name
}

// Capabilities returns the constraint types this solver can handle.
// Implements the Solver interface.
func (bs *BaseSolver) Capabilities() []string {
	caps := make([]string, len(bs.capabilities))
	copy(caps, bs.capabilities)
	return caps
}

// Priority returns the preference level for this solver.
// Implements the Solver interface.
func (bs *BaseSolver) Priority() int {
	return bs.priority
}

// CanHandle checks if this solver can handle a specific constraint.
// Default implementation checks if the constraint type is in capabilities.
// Subclasses can override for more sophisticated checking.
// Implements the Solver interface.
func (bs *BaseSolver) CanHandle(constraint Constraint) bool {
	constraintType := bs.getConstraintTypeName(constraint)
	for _, cap := range bs.capabilities {
		if cap == constraintType {
			return true
		}
	}
	return false
}

// Solve provides a default implementation that returns an error.
// Subclasses must override this method to implement actual solving logic.
// Implements the Solver interface.
func (bs *BaseSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	return nil, fmt.Errorf("solver %s does not implement Solve method", bs.name)
}

// getConstraintTypeName extracts the type name from a constraint instance.
// Uses reflection to determine the concrete type for capability checking.
func (bs *BaseSolver) getConstraintTypeName(constraint Constraint) string {
	constraintType := reflect.TypeOf(constraint)
	if constraintType.Kind() == reflect.Ptr {
		constraintType = constraintType.Elem()
	}
	return constraintType.Name()
}

// SolverRegistry provides a global registry for solver discovery and management.
// It allows solvers to be registered and discovered by constraint type,
// enabling automatic solver selection and pluggable architecture.
type SolverRegistry struct {
	// solvers maps solver IDs to solver instances
	solvers map[string]Solver

	// solversByType maps constraint types to lists of capable solvers
	solversByType map[string][]Solver

	// mu protects concurrent access to registry
	mu sync.RWMutex
}

// NewSolverRegistry creates a new solver registry.
func NewSolverRegistry() *SolverRegistry {
	return &SolverRegistry{
		solvers:       make(map[string]Solver),
		solversByType: make(map[string][]Solver),
	}
}

// RegisterSolver registers a solver with the registry.
// The solver becomes available for automatic discovery and routing.
func (sr *SolverRegistry) RegisterSolver(solver Solver) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	solverID := solver.ID()
	if _, exists := sr.solvers[solverID]; exists {
		return fmt.Errorf("solver with ID %s already registered", solverID)
	}

	sr.solvers[solverID] = solver

	// Register solver for each of its capabilities
	for _, capability := range solver.Capabilities() {
		sr.solversByType[capability] = append(sr.solversByType[capability], solver)
	}

	return nil
}

// UnregisterSolver removes a solver from the registry.
func (sr *SolverRegistry) UnregisterSolver(solverID string) error {
	sr.mu.RLock()
	_, exists := sr.solvers[solverID]
	sr.mu.RUnlock()

	if !exists {
		return fmt.Errorf("solver with ID %s not registered", solverID)
	}

	sr.mu.Lock()
	defer sr.mu.Unlock()

	delete(sr.solvers, solverID)

	// Remove solver from capability mappings
	for capability, solvers := range sr.solversByType {
		filtered := make([]Solver, 0, len(solvers))
		for _, s := range solvers {
			if s.ID() != solverID {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			delete(sr.solversByType, capability)
		} else {
			sr.solversByType[capability] = filtered
		}
	}

	return nil
}

// GetSolver returns a solver by its ID.
func (sr *SolverRegistry) GetSolver(solverID string) (Solver, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	solver, exists := sr.solvers[solverID]
	if !exists {
		return nil, fmt.Errorf("solver with ID %s not found", solverID)
	}

	return solver, nil
}

// GetSolversForType returns all solvers capable of handling a constraint type.
func (sr *SolverRegistry) GetSolversForType(constraintType string) []Solver {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	solvers, exists := sr.solversByType[constraintType]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]Solver, len(solvers))
	copy(result, solvers)
	return result
}

// GetAllSolvers returns all registered solvers.
func (sr *SolverRegistry) GetAllSolvers() []Solver {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	solvers := make([]Solver, 0, len(sr.solvers))
	for _, solver := range sr.solvers {
		solvers = append(solvers, solver)
	}

	return solvers
}

// FindBestSolver selects the best solver for a given constraint.
// Selection is based on solver priority and capability matching.
func (sr *SolverRegistry) FindBestSolver(constraint Constraint) (Solver, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	constraintType := sr.getConstraintTypeName(constraint)
	solvers, exists := sr.solversByType[constraintType]
	if !exists || len(solvers) == 0 {
		return nil, fmt.Errorf("no solvers available for constraint type %s", constraintType)
	}

	// Find the solver with highest priority
	var bestSolver Solver
	bestPriority := -1

	for _, solver := range solvers {
		if solver.CanHandle(constraint) && solver.Priority() > bestPriority {
			bestSolver = solver
			bestPriority = solver.Priority()
		}
	}

	if bestSolver == nil {
		return nil, fmt.Errorf("no suitable solver found for constraint %s", constraint.ID())
	}

	return bestSolver, nil
}

// getConstraintTypeName extracts the type name from a constraint instance.
func (sr *SolverRegistry) getConstraintTypeName(constraint Constraint) string {
	constraintType := reflect.TypeOf(constraint)
	if constraintType.Kind() == reflect.Ptr {
		constraintType = constraintType.Elem()
	}
	return constraintType.Name()
}
