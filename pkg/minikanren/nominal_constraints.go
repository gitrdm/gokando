// Package minikanren provides nominal constraints for the constraint logic programming system.
// Nominal constraints extend the constraint system with support for name binding,
// freshness, and scoping rules. These constraints integrate seamlessly with the
// pluggable constraint architecture from Phase 2.
//
// Key nominal constraints:
//   - Freshness constraints: Ensure names don't conflict
//   - Binding constraints: Establish name-to-term bindings
//   - Scope constraints: Manage lexical scoping rules
//   - Alpha-equivalence constraints: Reason about term equivalence up to renaming
//
// This implementation provides:
//   - NominalConstraint interface extending the base Constraint interface
//   - Concrete constraint implementations (Freshness, Binding, Scope)
//   - Integration with constraint manager and solver system
//   - Thread-safe operations with proper synchronization
package minikanren

import (
	"context"
	"fmt"
	"sync"
)

// NominalConstraint extends the base Constraint interface with nominal logic capabilities.
// Nominal constraints can reason about names, binding, and scope in addition to
// regular constraint logic. They integrate with the constraint store and solver system.
type NominalConstraint interface {
	Constraint

	// GetNominalScope returns the nominal scope associated with this constraint.
	GetNominalScope() *NominalScope

	// InvolvesName returns true if the constraint involves the given name.
	InvolvesName(name *Name) bool

	// GetInvolvedNames returns all names involved in this constraint.
	GetInvolvedNames() []*Name
}

// NominalConstraintBase provides common functionality for nominal constraints.
// It implements the basic Constraint interface methods and provides nominal-specific utilities.
type NominalConstraintBase struct {
	id       string
	vars     []*Var
	scope    *NominalScope
	priority int
	mu       sync.RWMutex
}

// NewNominalConstraintBase creates a new base for nominal constraints.
func NewNominalConstraintBase(id string, vars []*Var, scope *NominalScope) *NominalConstraintBase {
	return &NominalConstraintBase{
		id:       id,
		vars:     vars,
		scope:    scope,
		priority: 0, // Default priority
	}
}

// ID returns the unique identifier of this constraint.
func (ncb *NominalConstraintBase) ID() string {
	ncb.mu.RLock()
	defer ncb.mu.RUnlock()
	return ncb.id
}

// Variables returns the logic variables involved in this constraint.
func (ncb *NominalConstraintBase) Variables() []*Var {
	ncb.mu.RLock()
	defer ncb.mu.RUnlock()
	// Return a copy to prevent external modification
	vars := make([]*Var, len(ncb.vars))
	copy(vars, ncb.vars)
	return vars
}

// IsLocal returns true if this constraint can be evaluated purely
// within a local constraint store, false if it requires global coordination.
func (ncb *NominalConstraintBase) IsLocal() bool {
	return true // Nominal constraints are typically local
}

// Propagate attempts to narrow variable domains based on this constraint.
// Returns true if any propagation occurred, false otherwise.
func (ncb *NominalConstraintBase) Propagate(store ConstraintStore) bool {
	// Base implementation - subclasses should override
	return false
}

// Clone creates a deep copy of the constraint.
// This base implementation should not be called - concrete constraints must override.
func (ncb *NominalConstraintBase) Clone() Constraint {
	panic("NominalConstraintBase.Clone() should be overridden by concrete constraints")
}

// String returns a string representation of the constraint.
func (ncb *NominalConstraintBase) String() string {
	ncb.mu.RLock()
	defer ncb.mu.RUnlock()
	return fmt.Sprintf("NominalConstraint{id=%s, vars=%d, scope_size=%d}",
		ncb.id, len(ncb.vars), ncb.scope.Size())
}

// GetNominalScope returns the nominal scope associated with this constraint.
func (ncb *NominalConstraintBase) GetNominalScope() *NominalScope {
	ncb.mu.RLock()
	defer ncb.mu.RUnlock()
	return ncb.scope.Clone()
}

// InvolvesName returns true if the constraint involves the given name.
func (ncb *NominalConstraintBase) InvolvesName(name *Name) bool {
	// Base implementation - subclasses should override if they track specific names
	return false
}

// GetInvolvedNames returns all names involved in this constraint.
func (ncb *NominalConstraintBase) GetInvolvedNames() []*Name {
	// Base implementation - subclasses should override
	return []*Name{}
}

// FreshnessConstraint ensures that a set of names are fresh (not bound) in a given scope.
// This is crucial for avoiding name capture and ensuring proper alpha-equivalence.
// Freshness constraints are used to guarantee that generated names don't conflict
// with existing bindings.
type FreshnessConstraint struct {
	*NominalConstraintBase
	names []*Name // Names that must be fresh
}

// NewFreshnessConstraint creates a new freshness constraint.
func NewFreshnessConstraint(names []*Name, scope *NominalScope) *FreshnessConstraint {
	vars := []*Var{} // Freshness constraints don't directly involve logic variables

	base := NewNominalConstraintBase(
		fmt.Sprintf("freshness-%d", len(names)),
		vars,
		scope,
	)

	return &FreshnessConstraint{
		NominalConstraintBase: base,
		names:                 names,
	}
}

// Check validates that all names in the constraint are fresh in the current scope.
func (fc *FreshnessConstraint) Check(bindings map[int64]Term) ConstraintResult {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	for _, name := range fc.names {
		if fc.scope.IsBound(name) {
			return ConstraintViolated
		}
	}

	return ConstraintSatisfied
}

// InvolvesName returns true if the given name is in the freshness constraint.
func (fc *FreshnessConstraint) InvolvesName(name *Name) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	for _, n := range fc.names {
		if n.Equal(name) {
			return true
		}
	}
	return false
}

// GetInvolvedNames returns all names in the freshness constraint.
func (fc *FreshnessConstraint) GetInvolvedNames() []*Name {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	names := make([]*Name, len(fc.names))
	copy(names, fc.names)
	return names
}

// Clone creates a deep copy of the freshness constraint.
func (fc *FreshnessConstraint) Clone() Constraint {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	newNames := make([]*Name, len(fc.names))
	copy(newNames, fc.names)

	return NewFreshnessConstraint(newNames, fc.scope.Clone())
}

// String returns a string representation of the freshness constraint.
func (fc *FreshnessConstraint) String() string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	nameStrs := make([]string, len(fc.names))
	for i, name := range fc.names {
		nameStrs[i] = name.String()
	}

	return fmt.Sprintf("FreshnessConstraint{names=[%s], scope_size=%d}",
		fmt.Sprintf("%v", nameStrs), fc.scope.Size())
}

// BindingConstraint establishes a binding between a name and a term within a scope.
// This represents lexical binding in nominal logic and affects unification operations.
// Binding constraints are fundamental to representing variable binding in programming languages.
type BindingConstraint struct {
	*NominalConstraintBase
	name *Name // The name being bound
	term Term  // The term the name is bound to
}

// NewBindingConstraint creates a new binding constraint.
func NewBindingConstraint(name *Name, term Term, scope *NominalScope) *BindingConstraint {
	vars := []*Var{} // Binding constraints don't directly involve logic variables

	base := NewNominalConstraintBase(
		fmt.Sprintf("binding-%s", name.String()),
		vars,
		scope,
	)

	return &BindingConstraint{
		NominalConstraintBase: base,
		name:                  name,
		term:                  term,
	}
}

// Check validates the binding constraint.
// In nominal logic, binding constraints are always satisfiable by definition,
// but this could be extended to check for binding conflicts.
func (bc *BindingConstraint) Check(bindings map[int64]Term) ConstraintResult {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Binding constraints are always satisfiable
	// (they establish bindings rather than constrain them)
	return ConstraintSatisfied
}

// Propagate applies the binding to the scope.
// This ensures the binding is active during constraint propagation.
func (bc *BindingConstraint) Propagate(store ConstraintStore) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// The binding is already established in the scope
	// No additional propagation needed for basic binding
	return false
}

// InvolvesName returns true if the given name is the one being bound.
func (bc *BindingConstraint) InvolvesName(name *Name) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.name.Equal(name)
}

// GetInvolvedNames returns the name being bound.
func (bc *BindingConstraint) GetInvolvedNames() []*Name {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return []*Name{bc.name}
}

// Clone creates a deep copy of the binding constraint.
func (bc *BindingConstraint) Clone() Constraint {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return NewBindingConstraint(bc.name, bc.term.Clone(), bc.scope.Clone())
}

// String returns a string representation of the binding constraint.
func (bc *BindingConstraint) String() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return fmt.Sprintf("BindingConstraint{%s ↦ %s, scope_size=%d}",
		bc.name.String(), bc.term.String(), bc.scope.Size())
}

// ScopeConstraint manages lexical scoping for name bindings.
// It ensures that bindings are properly scoped and that name lookups
// respect lexical scoping rules. This is essential for proper handling
// of nested scopes and variable shadowing.
type ScopeConstraint struct {
	*NominalConstraintBase
	parentScope *NominalScope // Parent scope for lexical lookup
	childScope  *NominalScope // Child scope with local bindings
}

// NewScopeConstraint creates a new scope constraint.
func NewScopeConstraint(parentScope, childScope *NominalScope) *ScopeConstraint {
	vars := []*Var{} // Scope constraints don't directly involve logic variables

	base := NewNominalConstraintBase(
		fmt.Sprintf("scope-%d-%d", parentScope.Size(), childScope.Size()),
		vars,
		childScope, // Use child scope as the primary scope
	)

	return &ScopeConstraint{
		NominalConstraintBase: base,
		parentScope:           parentScope,
		childScope:            childScope,
	}
}

// Check validates the scope relationship.
// Ensures that the child scope properly inherits from the parent scope.
func (sc *ScopeConstraint) Check(bindings map[int64]Term) ConstraintResult {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Verify that child scope has parent set correctly
	if sc.childScope.parent != sc.parentScope {
		return ConstraintViolated
	}

	return ConstraintSatisfied
}

// InvolvesName returns true if the name is bound in either scope.
func (sc *ScopeConstraint) InvolvesName(name *Name) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.childScope.IsBound(name) || sc.parentScope.IsBound(name)
}

// GetInvolvedNames returns all names bound in both scopes.
func (sc *ScopeConstraint) GetInvolvedNames() []*Name {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// For now, return empty slice as proper implementation would require
	// maintaining reverse mappings from IDs to names, which is complex
	return []*Name{}
}

// Clone creates a deep copy of the scope constraint.
func (sc *ScopeConstraint) Clone() Constraint {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return NewScopeConstraint(sc.parentScope.Clone(), sc.childScope.Clone())
}

// String returns a string representation of the scope constraint.
func (sc *ScopeConstraint) String() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return fmt.Sprintf("ScopeConstraint{parent_size=%d, child_size=%d}",
		sc.parentScope.Size(), sc.childScope.Size())
}

// NominalConstraintSolver handles solving of nominal constraints.
// It integrates with the solver interface from Phase 2 and provides
// specialized solving for nominal logic constraints.
type NominalConstraintSolver struct {
	id           string
	capabilities []string
	priority     int
	mu           sync.RWMutex
}

// NewNominalConstraintSolver creates a new nominal constraint solver.
func NewNominalConstraintSolver() *NominalConstraintSolver {
	return &NominalConstraintSolver{
		id:           "nominal-solver",
		capabilities: []string{"nominal", "freshness", "binding", "scope"},
		priority:     100, // High priority for nominal constraints
	}
}

// ID returns the solver's unique identifier.
func (ncs *NominalConstraintSolver) ID() string {
	return ncs.id
}

// Name returns a human-readable name for the solver.
func (ncs *NominalConstraintSolver) Name() string {
	return "Nominal Constraint Solver"
}

// Capabilities returns the constraint types this solver can handle.
func (ncs *NominalConstraintSolver) Capabilities() []string {
	ncs.mu.RLock()
	defer ncs.mu.RUnlock()
	caps := make([]string, len(ncs.capabilities))
	copy(caps, ncs.capabilities)
	return caps
}

// Solve attempts to solve a nominal constraint.
func (ncs *NominalConstraintSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	// Check if this is a nominal constraint
	nominalConstraint, ok := constraint.(NominalConstraint)
	if !ok {
		return nil, fmt.Errorf("not a nominal constraint")
	}

	// Get current bindings from the store
	substitution := store.GetSubstitution()
	substitution.mu.RLock()
	bindings := make(map[int64]Term, len(substitution.bindings))
	for k, v := range substitution.bindings {
		bindings[k] = v
	}
	substitution.mu.RUnlock()

	// Check the constraint against current bindings
	result := nominalConstraint.Check(bindings)

	// For nominal constraints, if satisfied, return the store unchanged
	// If violated, return nil to indicate failure
	// If pending, return the store unchanged (constraint remains active)
	if result == ConstraintViolated {
		return nil, fmt.Errorf("nominal constraint violated")
	}

	return store, nil
}

// CanHandle returns true if this solver can handle the given constraint.
func (ncs *NominalConstraintSolver) CanHandle(constraint Constraint) bool {
	_, ok := constraint.(NominalConstraint)
	return ok
}

// Priority returns the solver's priority (higher numbers = higher priority).
func (ncs *NominalConstraintSolver) Priority() int {
	ncs.mu.RLock()
	defer ncs.mu.RUnlock()
	return ncs.priority
}
