// Package minikanren provides hybrid constraint solving by integrating
// relational and finite-domain constraint solvers. This file defines the
// plugin architecture that allows specialized solvers to cooperate on
// problems requiring both types of reasoning.
//
// The hybrid solver uses a plugin pattern where:
//   - Each solver (relational, FD, etc.) implements the SolverPlugin interface
//   - The HybridSolver dispatches constraints to appropriate plugins
//   - The UnifiedStore maintains both relational bindings and FD domains
//   - Plugins propagate changes bidirectionally through the store
//
// This design enables:
//   - Attributed variables: variables with both relational bindings and finite domains
//   - Cross-solver propagation: FD pruning informs relational search and vice versa
//   - Modular extension: new solver types can be added without modifying core infrastructure
//   - Lock-free parallel search: UnifiedStore uses copy-on-write like SolverState
package minikanren

import (
	"fmt"
)

// SolverPlugin represents a specialized solver that can handle specific types
// of constraints. Plugins cooperate to solve hybrid problems by sharing a
// UnifiedStore containing both relational bindings and FD domains.
//
// Each plugin is responsible for:
//   - Identifying which constraints it can handle
//   - Propagating those constraints to prune the search space
//   - Communicating changes through the UnifiedStore
//
// Plugins must be thread-safe as they may be called concurrently during
// parallel search. They must also maintain the copy-on-write semantics
// required for lock-free operation: all state changes return new store versions.
type SolverPlugin interface {
	// Name returns a human-readable identifier for this plugin (e.g., "Relational", "FD").
	// Used for debugging and error reporting.
	Name() string

	// CanHandle returns true if this plugin can process the given constraint.
	// The HybridSolver uses this to route constraints to appropriate plugins.
	//
	// A plugin should only claim constraints it can actively propagate.
	// Returning true indicates the plugin will contribute to solving this constraint.
	CanHandle(constraint interface{}) bool

	// Propagate runs this plugin's constraint propagation algorithm on the given store.
	// Returns a new UnifiedStore with updated bindings/domains, or an error if a
	// conflict is detected.
	//
	// The plugin should:
	//   1. Read current state from the store (bindings for relational, domains for FD)
	//   2. Apply its propagation algorithm to prune the search space
	//   3. Return a new store with changes, preserving copy-on-write semantics
	//
	// Propagate must be idempotent: calling it multiple times on the same store
	// should produce the same result. The HybridSolver may call propagate repeatedly
	// until a fixed point is reached (no plugin makes further changes).
	//
	// Errors indicate unrecoverable conflicts (empty domain, violated constraint).
	// The HybridSolver will backtrack when any plugin returns an error.
	Propagate(store *UnifiedStore) (*UnifiedStore, error)
}

// UnifiedStore is a persistent data structure that holds both relational bindings
// (for miniKanren unification) and finite-domain constraints (for FD propagation).
// This enables attributed variables: a single logical variable can simultaneously
// have a relational binding and a finite domain.
//
// The store uses copy-on-write semantics for lock-free parallel search:
//   - Modifications create new store versions linked to the parent
//   - Most data is shared between versions (structural sharing)
//   - State branching for parallel workers is O(1)
//   - Memory overhead is O(changes) not O(total state)
//
// Store operations:
//   - Relational: AddBinding(), GetBinding(), GetSubstitution()
//   - Finite-domain: SetDomain(), GetDomain()
//   - Cross-solver: NotifyChange() for propagation triggering
//
// Thread safety: UnifiedStore is immutable. All modification methods return
// new instances, making concurrent reads safe without locks.
type UnifiedStore struct {
	// parent points to the previous store version in the chain.
	// nil for the root store.
	parent *UnifiedStore

	// relationalBindings holds variable bindings from unification.
	// Maps variable ID -> Term (Atom, Var, Pair)
	// Uses copy-on-write: only modified bindings are stored,
	// others are inherited from parent chain.
	relationalBindings map[int64]Term

	// fdDomains holds finite domains for FD variables.
	// Maps variable ID -> Domain
	// Uses copy-on-write: only modified domains are stored.
	fdDomains map[int]Domain

	// constraints holds all active constraints (both relational and FD).
	// Inherited from parent unless explicitly modified.
	constraints []interface{}

	// depth tracks the depth in the search tree.
	// Used for heuristics and debugging.
	depth int

	// changedVars tracks which variables were modified in this version.
	// Used to optimize propagation: only re-check constraints involving changed variables.
	changedVars map[int64]bool
}

// NewUnifiedStore creates a new empty unified store.
// This is the root of the store chain for a new search.
func NewUnifiedStore() *UnifiedStore {
	return &UnifiedStore{
		parent:             nil,
		relationalBindings: make(map[int64]Term),
		fdDomains:          make(map[int]Domain),
		constraints:        make([]interface{}, 0),
		depth:              0,
		changedVars:        make(map[int64]bool),
	}
}

// Clone creates a shallow copy of the store for branching search paths.
// The new store shares most data with the parent via structural sharing.
func (s *UnifiedStore) Clone() *UnifiedStore {
	return &UnifiedStore{
		parent:             s,
		relationalBindings: make(map[int64]Term),
		fdDomains:          make(map[int]Domain),
		constraints:        s.constraints, // Shared with parent
		depth:              s.depth + 1,
		changedVars:        make(map[int64]bool),
	}
}

// GetBinding retrieves the relational binding for a variable.
// Walks the parent chain to find the most recent binding.
// Returns nil if the variable is unbound.
func (s *UnifiedStore) GetBinding(varID int64) Term {
	// Check local bindings first
	if binding, ok := s.relationalBindings[varID]; ok {
		return binding
	}

	// Walk parent chain
	if s.parent != nil {
		return s.parent.GetBinding(varID)
	}

	return nil
}

// AddBinding creates a new store with an additional relational binding.
// This is used by the relational solver during unification.
//
// Returns a new store with the binding, or an error if the binding
// would violate constraints.
func (s *UnifiedStore) AddBinding(varID int64, term Term) (*UnifiedStore, error) {
	// Create new store version
	newStore := s.Clone()
	newStore.relationalBindings[varID] = term
	newStore.changedVars[varID] = true

	// Note: Constraint checking happens during propagation,
	// not immediately in AddBinding. This allows batched constraint checks.

	return newStore, nil
}

// GetDomain retrieves the finite domain for an FD variable.
// Walks the parent chain to find the most recent domain.
// Returns nil if the variable has no FD domain (relational-only variable).
func (s *UnifiedStore) GetDomain(varID int) Domain {
	// Check local domains first
	if domain, ok := s.fdDomains[varID]; ok {
		return domain
	}

	// Walk parent chain
	if s.parent != nil {
		return s.parent.GetDomain(varID)
	}

	return nil
}

// SetDomain creates a new store with an updated finite domain.
// This is used by the FD solver during propagation.
//
// Returns a new store with the domain change, or an error if the
// domain is empty (conflict detected).
func (s *UnifiedStore) SetDomain(varID int, domain Domain) (*UnifiedStore, error) {
	// Empty domain means no solution exists along this path
	if domain.Count() == 0 {
		return nil, fmt.Errorf("empty domain for variable %d - conflict detected", varID)
	}

	// Create new store version
	newStore := s.Clone()
	newStore.fdDomains[varID] = domain
	newStore.changedVars[int64(varID)] = true

	return newStore, nil
}

// AddConstraint creates a new store with an additional constraint.
// Constraints are checked during propagation, not immediately.
func (s *UnifiedStore) AddConstraint(constraint interface{}) *UnifiedStore {
	newStore := s.Clone()
	newStore.constraints = append(newStore.constraints, constraint)
	return newStore
}

// GetConstraints returns all active constraints in the store.
// This includes constraints from the entire parent chain.
func (s *UnifiedStore) GetConstraints() []interface{} {
	// Note: constraints are shared with parent chain via structural sharing.
	// If we ever need constraint-specific versioning, we'll need to merge
	// the parent chain here.
	return s.constraints
}

// GetSubstitution returns a Substitution representing all relational bindings.
// This bridges the UnifiedStore to miniKanren's substitution-based APIs.
func (s *UnifiedStore) GetSubstitution() *Substitution {
	sub := NewSubstitution()

	// Collect all bindings from the chain
	bindings := s.getAllBindings()

	for varID, term := range bindings {
		tempVar := &Var{id: varID}
		sub = sub.Bind(tempVar, term)
	}

	return sub
}

// getAllBindings walks the parent chain and collects all relational bindings.
// Used internally to avoid repeatedly walking the chain.
func (s *UnifiedStore) getAllBindings() map[int64]Term {
	bindings := make(map[int64]Term)

	// Walk parent chain from root to current
	s.collectBindings(bindings)

	return bindings
}

// collectBindings recursively collects bindings from the parent chain.
func (s *UnifiedStore) collectBindings(bindings map[int64]Term) {
	if s.parent != nil {
		s.parent.collectBindings(bindings)
	}

	// Overlay local bindings (shadowing parent values)
	for varID, term := range s.relationalBindings {
		bindings[varID] = term
	}
}

// String returns a human-readable representation of the store for debugging.
func (s *UnifiedStore) String() string {
	bindings := s.getAllBindings()
	domains := s.getAllDomains()

	return fmt.Sprintf("UnifiedStore{depth=%d, bindings=%d, domains=%d, constraints=%d}",
		s.depth, len(bindings), len(domains), len(s.constraints))
}

// getAllDomains walks the parent chain and collects all FD domains.
func (s *UnifiedStore) getAllDomains() map[int]Domain {
	domains := make(map[int]Domain)

	s.collectDomains(domains)

	return domains
}

// collectDomains recursively collects domains from the parent chain.
func (s *UnifiedStore) collectDomains(domains map[int]Domain) {
	if s.parent != nil {
		s.parent.collectDomains(domains)
	}

	// Overlay local domains
	for varID, domain := range s.fdDomains {
		domains[varID] = domain
	}
}

// Depth returns the depth of this store in the search tree.
// Used for heuristics and debugging.
func (s *UnifiedStore) Depth() int {
	return s.depth
}

// ChangedVariables returns the set of variables modified in this store version.
// Used to optimize propagation by only re-checking affected constraints.
func (s *UnifiedStore) ChangedVariables() map[int64]bool {
	return s.changedVars
}

// HybridSolver coordinates multiple solver plugins to solve problems requiring
// both relational and finite-domain reasoning. It dispatches constraints to
// appropriate plugins and runs propagation to a fixed point.
//
// The hybrid solver maintains a registry of plugins and routes constraints
// based on each plugin's CanHandle() method. During solving:
//  1. All plugins process the UnifiedStore in sequence
//  2. Each plugin that made changes returns a new store
//  3. The process repeats until no plugin makes further changes (fixed point)
//  4. If any plugin detects a conflict, solving backtracks
//
// Configuration options control:
//   - Maximum propagation iterations (prevent infinite loops)
//   - Plugin execution order (can affect performance)
//   - Timeout and solution limits
//
// Thread safety: HybridSolver is safe for concurrent use. Multiple solvers
// can work on different search branches simultaneously.
type HybridSolver struct {
	// plugins holds all registered solver plugins in execution order
	plugins []SolverPlugin

	// config holds solver parameters
	config *HybridSolverConfig
}

// HybridSolverConfig configures the hybrid solver's behavior.
type HybridSolverConfig struct {
	// MaxPropagationIterations limits how many times the solver will
	// iterate through all plugins before declaring a fixed point.
	// Prevents infinite loops from buggy constraint implementations.
	MaxPropagationIterations int

	// EnablePropagation controls whether constraint propagation runs.
	// Can be disabled for pure backtracking search.
	EnablePropagation bool
}

// DefaultHybridSolverConfig returns sensible default configuration.
func DefaultHybridSolverConfig() *HybridSolverConfig {
	return &HybridSolverConfig{
		MaxPropagationIterations: 1000,
		EnablePropagation:        true,
	}
}

// NewHybridSolver creates a hybrid solver with the given plugins.
// Plugins are executed in the order provided, which can affect performance.
//
// Typically, you'll register both a RelationalPlugin and an FDPlugin:
//
//	solver := NewHybridSolver(
//	    NewRelationalPlugin(),
//	    NewFDPlugin(model),
//	)
func NewHybridSolver(plugins ...SolverPlugin) *HybridSolver {
	return &HybridSolver{
		plugins: plugins,
		config:  DefaultHybridSolverConfig(),
	}
}

// NewHybridSolverWithConfig creates a hybrid solver with custom configuration.
func NewHybridSolverWithConfig(config *HybridSolverConfig, plugins ...SolverPlugin) *HybridSolver {
	return &HybridSolver{
		plugins: plugins,
		config:  config,
	}
}

// RegisterPlugin adds a plugin to the solver.
// Plugins are executed in registration order.
func (hs *HybridSolver) RegisterPlugin(plugin SolverPlugin) {
	hs.plugins = append(hs.plugins, plugin)
}

// GetPlugins returns all registered plugins.
// The returned slice should not be modified.
func (hs *HybridSolver) GetPlugins() []SolverPlugin {
	return hs.plugins
}

// SetConfig updates the solver configuration.
func (hs *HybridSolver) SetConfig(config *HybridSolverConfig) {
	hs.config = config
}

// Propagate runs all registered plugins to a fixed point on the given store.
// Returns a new store with all propagations applied, or an error if a conflict
// is detected.
//
// The propagation algorithm:
//  1. Run each plugin in sequence on the current store
//  2. If any plugin returns a new store (changes made), record it
//  3. After all plugins run, if changes occurred, repeat from step 1
//  4. Stop when no plugin makes changes (fixed point) or max iterations reached
//  5. Return error if any plugin detects a conflict
//
// This implements the "chaotic iteration" algorithm standard in constraint programming.
func (hs *HybridSolver) Propagate(store *UnifiedStore) (*UnifiedStore, error) {
	if !hs.config.EnablePropagation {
		return store, nil
	}

	currentStore := store
	iteration := 0

	for iteration < hs.config.MaxPropagationIterations {
		changed := false

		// Run each plugin
		for _, plugin := range hs.plugins {
			newStore, err := plugin.Propagate(currentStore)
			if err != nil {
				// Conflict detected by plugin
				return nil, fmt.Errorf("plugin %s detected conflict: %w", plugin.Name(), err)
			}

			// Check if plugin made changes
			if newStore != currentStore {
				changed = true
				currentStore = newStore
			}
		}

		// Fixed point reached if no plugin made changes
		if !changed {
			return currentStore, nil
		}

		iteration++
	}

	// Max iterations reached without fixed point
	// This could indicate a bug in constraint propagation
	return nil, fmt.Errorf("propagation failed to reach fixed point after %d iterations", iteration)
}

// PropagateWithConstraints runs propagation after adding new constraints.
// This is a convenience method that combines constraint addition with propagation.
func (hs *HybridSolver) PropagateWithConstraints(store *UnifiedStore, constraints ...interface{}) (*UnifiedStore, error) {
	// Add all constraints
	newStore := store
	for _, constraint := range constraints {
		newStore = newStore.AddConstraint(constraint)
	}

	// Run propagation
	return hs.Propagate(newStore)
}

// CanHandle returns a list of plugins that can handle the given constraint.
// Used for debugging and understanding constraint routing.
func (hs *HybridSolver) CanHandle(constraint interface{}) []SolverPlugin {
	handlers := make([]SolverPlugin, 0)

	for _, plugin := range hs.plugins {
		if plugin.CanHandle(constraint) {
			handlers = append(handlers, plugin)
		}
	}

	return handlers
}

// String returns a human-readable representation of the solver.
func (hs *HybridSolver) String() string {
	pluginNames := make([]string, len(hs.plugins))
	for i, plugin := range hs.plugins {
		pluginNames[i] = plugin.Name()
	}

	return fmt.Sprintf("HybridSolver{plugins=%v}", pluginNames)
}
