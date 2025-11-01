// Package minikanren provides plugin implementations for the hybrid solver.
// This file implements the FD (Finite Domain) plugin that wraps the
// existing FD constraint propagation infrastructure from Phase 2.
package minikanren

import (
	"fmt"
)

// FDPlugin wraps the Phase 2 FD solver to work within the hybrid framework.
// It handles PropagationConstraints (AllDifferent, Arithmetic, Inequality)
// by running them on the FD domains stored in the UnifiedStore.
//
// The FDPlugin bridges between:
//   - UnifiedStore: holds FD domains for all variables
//   - Solver: runs propagation using those domains
//   - PropagationConstraints: prune domains based on constraint semantics
//
// During propagation, the FDPlugin:
//  1. Extracts FD domains from the UnifiedStore
//  2. Builds a temporary SolverState representing those domains
//  3. Runs FD propagation constraints to fixed point
//  4. Extracts pruned domains back into a new UnifiedStore
//
// This allows the FD solver to participate in hybrid solving without
// modifying its core architecture.
type FDPlugin struct {
	// model holds the FD variables and constraints
	model *Model

	// solver performs FD constraint propagation
	solver *Solver
}

// NewFDPlugin creates an FD plugin for the given model.
// The model should contain all FD variables and PropagationConstraints.
func NewFDPlugin(model *Model) *FDPlugin {
	return &FDPlugin{
		model:  model,
		solver: NewSolver(model),
	}
}

// Name returns the plugin identifier.
// Implements SolverPlugin.
func (fp *FDPlugin) Name() string {
	return "FD"
}

// CanHandle returns true if the constraint is an FD constraint.
// Implements SolverPlugin.
func (fp *FDPlugin) CanHandle(constraint interface{}) bool {
	// Check if it's a PropagationConstraint
	_, ok := constraint.(PropagationConstraint)
	return ok
}

// Propagate runs FD constraint propagation on the unified store.
// Implements SolverPlugin.
func (fp *FDPlugin) Propagate(store *UnifiedStore) (*UnifiedStore, error) {
	// Extract FD domains from the store into a SolverState
	state := fp.storeToState(store)

	// Check if there are any FD constraints to propagate
	if len(fp.model.Constraints()) == 0 {
		// No FD constraints, nothing to propagate
		return store, nil
	}

	// Run FD propagation
	newState, err := fp.solver.propagate(state)
	if err != nil {
		return nil, fmt.Errorf("FD propagation failed: %w", err)
	}

	// Convert propagated state back to UnifiedStore
	newStore, err := fp.stateToStore(newState, store)
	if err != nil {
		return nil, fmt.Errorf("FD state conversion failed: %w", err)
	}

	return newStore, nil
}

// storeToState builds a SolverState from UnifiedStore FD domains.
// This allows the FD Solver to work with domains from the hybrid store.
func (fp *FDPlugin) storeToState(store *UnifiedStore) *SolverState {
	// Start with empty state
	state := (*SolverState)(nil)

	// Add each variable's domain from the store
	for _, v := range fp.model.Variables() {
		domain := store.GetDomain(v.ID())
		if domain == nil {
			// Variable has no FD domain yet, use its initial domain
			domain = v.Domain()
		}

		// Create state with this domain
		state, _ = fp.solver.SetDomain(state, v.ID(), domain)
	}

	return state
}

// stateToStore converts a SolverState back into a UnifiedStore.
// Extracts all domains from the state and updates the store.
func (fp *FDPlugin) stateToStore(state *SolverState, originalStore *UnifiedStore) (*UnifiedStore, error) {
	newStore := originalStore

	// Update each variable's domain in the store
	for _, v := range fp.model.Variables() {
		domain := fp.solver.GetDomain(state, v.ID())

		// Only update if domain changed
		currentDomain := originalStore.GetDomain(v.ID())
		if currentDomain == nil || !domain.Equal(currentDomain) {
			var err error
			newStore, err = newStore.SetDomain(v.ID(), domain)
			if err != nil {
				return nil, err
			}
		}
	}

	return newStore, nil
}

// GetModel returns the FD model used by this plugin.
// Useful for debugging and testing.
func (fp *FDPlugin) GetModel() *Model {
	return fp.model
}

// GetSolver returns the FD solver used by this plugin.
// Useful for debugging and testing.
func (fp *FDPlugin) GetSolver() *Solver {
	return fp.solver
}
