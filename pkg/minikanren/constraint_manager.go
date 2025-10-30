// Package minikanren provides a constraint manager for automatic solver routing
// and pluggable constraint system architecture. The constraint manager acts as
// the central coordinator between constraints and solvers, enabling:
//
//   - Automatic solver selection based on constraint characteristics
//   - Pluggable solver architecture for extensibility
//   - Fallback mechanisms for unhandled constraints
//   - Performance monitoring and optimization
//   - Thread-safe constraint processing
//
// The constraint manager implements a registry-based approach where:
//   - Constraints are registered with their supported solvers
//   - Solvers are registered with their capabilities
//   - Automatic routing selects the best solver for each constraint
//   - Fallback solvers handle constraints that no specialized solver can process
package minikanren

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ConstraintManager coordinates constraint solving across multiple solvers.
// It provides automatic solver selection, fallback mechanisms, and performance
// monitoring for the pluggable constraint system.
//
// The manager maintains:
//   - A registry of available solvers with their capabilities
//   - Constraint-to-solver mappings for efficient routing
//   - Performance metrics and solver selection heuristics
//   - Fallback solvers for unhandled constraints
type ConstraintManager struct {
	// solvers maps solver IDs to solver instances
	solvers map[string]Solver

	// constraintTypes maps constraint type names to supported solvers
	constraintTypes map[string][]string

	// fallbackSolvers provides default solvers for unhandled constraints
	fallbackSolvers []Solver

	// metrics tracks solver performance and usage statistics
	metrics *SolverMetrics

	// mu protects concurrent access to manager state
	mu sync.RWMutex

	// shutdown indicates if the manager is shutting down
	shutdown bool
}

// SolverMetrics tracks performance and usage statistics for solvers.
type SolverMetrics struct {
	// solverStats maps solver IDs to their performance statistics
	solverStats map[string]*SolverStats

	// totalConstraintsProcessed counts total constraints processed
	totalConstraintsProcessed int64

	// totalProcessingTime tracks cumulative processing time
	totalProcessingTime time.Duration

	// mu protects concurrent access to metrics
	mu sync.RWMutex
}

// SolverStats contains performance statistics for a single solver.
type SolverStats struct {
	// constraintsProcessed counts constraints processed by this solver
	constraintsProcessed int64

	// totalProcessingTime tracks cumulative processing time
	totalProcessingTime time.Duration

	// averageProcessingTime caches the computed average
	averageProcessingTime time.Duration

	// successCount tracks successful constraint resolutions
	successCount int64

	// failureCount tracks failed constraint resolutions
	failureCount int64

	// lastUsed tracks when this solver was last used
	lastUsed time.Time
}

// NewConstraintManager creates a new constraint manager with default configuration.
// The manager starts with no registered solvers and must be configured
// before use by registering solvers and constraint types.
func NewConstraintManager() *ConstraintManager {
	return &ConstraintManager{
		solvers:         make(map[string]Solver),
		constraintTypes: make(map[string][]string),
		metrics: &SolverMetrics{
			solverStats: make(map[string]*SolverStats),
		},
	}
}

// RegisterSolver registers a solver with the constraint manager.
// The solver becomes available for automatic routing based on its capabilities.
//
// Returns an error if a solver with the same ID is already registered.
func (cm *ConstraintManager) RegisterSolver(solver Solver) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.shutdown {
		return fmt.Errorf("constraint manager is shutdown")
	}

	solverID := solver.ID()
	if _, exists := cm.solvers[solverID]; exists {
		return fmt.Errorf("solver with ID %s already registered", solverID)
	}

	cm.solvers[solverID] = solver
	cm.metrics.solverStats[solverID] = &SolverStats{
		lastUsed: time.Now(),
	}

	return nil
}

// UnregisterSolver removes a solver from the constraint manager.
// The solver will no longer be available for constraint processing.
//
// Returns an error if the solver is not registered.
func (cm *ConstraintManager) UnregisterSolver(solverID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.solvers[solverID]; !exists {
		return fmt.Errorf("solver with ID %s not registered", solverID)
	}

	delete(cm.solvers, solverID)
	delete(cm.metrics.solverStats, solverID)

	// Remove this solver from all constraint type mappings
	for constraintType, solverIDs := range cm.constraintTypes {
		filtered := make([]string, 0, len(solverIDs))
		for _, id := range solverIDs {
			if id != solverID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == 0 {
			delete(cm.constraintTypes, constraintType)
		} else {
			cm.constraintTypes[constraintType] = filtered
		}
	}

	return nil
}

// RegisterConstraintType associates a constraint type with a list of capable solvers.
// The constraint type is typically the name of the constraint struct type.
// Solvers are ordered by preference (first solver in the list is preferred).
func (cm *ConstraintManager) RegisterConstraintType(constraintType string, solverIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.shutdown {
		return fmt.Errorf("constraint manager is shutdown")
	}

	// Validate that all specified solvers are registered
	for _, solverID := range solverIDs {
		if _, exists := cm.solvers[solverID]; !exists {
			return fmt.Errorf("solver %s not registered", solverID)
		}
	}

	cm.constraintTypes[constraintType] = make([]string, len(solverIDs))
	copy(cm.constraintTypes[constraintType], solverIDs)

	return nil
}

// SetFallbackSolvers configures the fallback solvers to use when no specialized
// solver can handle a constraint. Fallback solvers are tried in order until
// one succeeds or all fail.
func (cm *ConstraintManager) SetFallbackSolvers(solvers []Solver) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.shutdown {
		return fmt.Errorf("constraint manager is shutdown")
	}

	// Validate that all fallback solvers are registered
	for _, solver := range solvers {
		if _, exists := cm.solvers[solver.ID()]; !exists {
			return fmt.Errorf("fallback solver %s not registered", solver.ID())
		}
	}

	cm.fallbackSolvers = make([]Solver, len(solvers))
	copy(cm.fallbackSolvers, solvers)

	return nil
}

// SolveConstraint attempts to solve a constraint using the most appropriate solver.
// The manager automatically selects a solver based on constraint type and performance
// metrics, falling back to fallback solvers if necessary.
//
// Returns the updated constraint store if successful, or an error if the constraint
// cannot be satisfied by any available solver.
func (cm *ConstraintManager) SolveConstraint(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	startTime := time.Now()

	cm.mu.RLock()
	if cm.shutdown {
		cm.mu.RUnlock()
		return nil, fmt.Errorf("constraint manager is shutdown")
	}
	cm.mu.RUnlock()

	// Get constraint type name for routing
	constraintType := cm.getConstraintTypeName(constraint)

	// Find the best solver for this constraint
	solver, err := cm.selectSolver(constraintType)
	if err != nil {
		return nil, fmt.Errorf("no solver available for constraint type %s: %w", constraintType, err)
	}

	// Attempt to solve with the selected solver
	resultStore, solveErr := solver.Solve(ctx, constraint, store)

	// Record metrics
	cm.recordSolverMetrics(solver.ID(), time.Since(startTime), solveErr == nil)

	if solveErr != nil {
		// Try fallback solvers if the primary solver failed
		for _, fallbackSolver := range cm.fallbackSolvers {
			if fallbackSolver.ID() == solver.ID() {
				continue // Don't retry the same solver
			}

			fallbackResult, fallbackErr := fallbackSolver.Solve(ctx, constraint, store)
			if fallbackErr == nil {
				cm.recordSolverMetrics(fallbackSolver.ID(), time.Since(startTime), true)
				return fallbackResult, nil
			}
			cm.recordSolverMetrics(fallbackSolver.ID(), time.Since(startTime), false)
		}

		return nil, fmt.Errorf("constraint %s could not be satisfied by any solver: %w", constraint.ID(), solveErr)
	}

	return resultStore, nil
}

// getConstraintTypeName extracts the type name from a constraint instance.
// Uses reflection to determine the concrete type for routing decisions.
func (cm *ConstraintManager) getConstraintTypeName(constraint Constraint) string {
	constraintType := reflect.TypeOf(constraint)
	if constraintType.Kind() == reflect.Ptr {
		constraintType = constraintType.Elem()
	}
	return constraintType.Name()
}

// selectSolver chooses the best solver for a given constraint type.
// Selection is based on registered solvers, performance metrics, and load balancing.
func (cm *ConstraintManager) selectSolver(constraintType string) (Solver, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	solverIDs, exists := cm.constraintTypes[constraintType]
	if !exists || len(solverIDs) == 0 {
		// No specific solvers for this type, try fallback solvers
		if len(cm.fallbackSolvers) > 0 {
			return cm.fallbackSolvers[0], nil
		}
		return nil, fmt.Errorf("no solvers registered for constraint type %s", constraintType)
	}

	// Select the best solver based on performance metrics
	bestSolverID := cm.selectBestSolver(solverIDs)
	return cm.solvers[bestSolverID], nil
}

// selectBestSolver chooses the best solver from a list of candidates.
// Selection criteria include success rate, average processing time, and load balancing.
func (cm *ConstraintManager) selectBestSolver(solverIDs []string) string {
	if len(solverIDs) == 1 {
		return solverIDs[0]
	}

	// Score each solver based on performance metrics
	type solverScore struct {
		id    string
		score float64
	}

	scores := make([]solverScore, 0, len(solverIDs))

	for _, solverID := range solverIDs {
		stats, exists := cm.metrics.solverStats[solverID]
		if !exists {
			// No stats yet, use default score
			scores = append(scores, solverScore{id: solverID, score: 1.0})
			continue
		}

		totalAttempts := stats.successCount + stats.failureCount
		if totalAttempts == 0 {
			// No attempts yet, use default score
			scores = append(scores, solverScore{id: solverID, score: 1.0})
			continue
		}

		// Calculate score based on success rate and performance
		successRate := float64(stats.successCount) / float64(totalAttempts)
		avgTime := stats.averageProcessingTime
		timeScore := 1.0 / (1.0 + avgTime.Seconds()) // Lower time is better

		score := successRate * timeScore
		scores = append(scores, solverScore{id: solverID, score: score})
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].id
}

// recordSolverMetrics updates performance statistics for a solver.
func (cm *ConstraintManager) recordSolverMetrics(solverID string, duration time.Duration, success bool) {
	cm.metrics.mu.Lock()
	defer cm.metrics.mu.Unlock()

	stats, exists := cm.metrics.solverStats[solverID]
	if !exists {
		stats = &SolverStats{}
		cm.metrics.solverStats[solverID] = stats
	}

	atomic.AddInt64(&cm.metrics.totalConstraintsProcessed, 1)
	cm.metrics.totalProcessingTime += duration

	stats.totalProcessingTime += duration
	atomic.AddInt64(&stats.constraintsProcessed, 1)

	if success {
		atomic.AddInt64(&stats.successCount, 1)
	} else {
		atomic.AddInt64(&stats.failureCount, 1)
	}

	// Update average processing time
	totalProcessed := atomic.LoadInt64(&stats.constraintsProcessed)
	if totalProcessed > 0 {
		stats.averageProcessingTime = stats.totalProcessingTime / time.Duration(totalProcessed)
	}

	stats.lastUsed = time.Now()
}

// GetMetrics returns a copy of the current solver metrics.
// Useful for monitoring and debugging solver performance.
func (cm *ConstraintManager) GetMetrics() *SolverMetrics {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	// Create a deep copy of metrics
	metricsCopy := &SolverMetrics{
		solverStats:               make(map[string]*SolverStats),
		totalConstraintsProcessed: atomic.LoadInt64(&cm.metrics.totalConstraintsProcessed),
		totalProcessingTime:       cm.metrics.totalProcessingTime,
	}

	for solverID, stats := range cm.metrics.solverStats {
		statsCopy := &SolverStats{
			constraintsProcessed:  atomic.LoadInt64(&stats.constraintsProcessed),
			totalProcessingTime:   stats.totalProcessingTime,
			averageProcessingTime: stats.averageProcessingTime,
			successCount:          atomic.LoadInt64(&stats.successCount),
			failureCount:          atomic.LoadInt64(&stats.failureCount),
			lastUsed:              stats.lastUsed,
		}
		metricsCopy.solverStats[solverID] = statsCopy
	}

	return metricsCopy
}

// Shutdown gracefully shuts down the constraint manager.
// All pending operations complete before shutdown.
func (cm *ConstraintManager) Shutdown() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.shutdown {
		return
	}

	cm.shutdown = true

	// Shutdown all registered solvers
	for _, solver := range cm.solvers {
		if shutdownable, ok := solver.(interface{ Shutdown() }); ok {
			shutdownable.Shutdown()
		}
	}

	// Clear state
	cm.solvers = nil
	cm.constraintTypes = nil
	cm.fallbackSolvers = nil
}

// IsShutdown returns true if the constraint manager has been shut down.
func (cm *ConstraintManager) IsShutdown() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.shutdown
}
