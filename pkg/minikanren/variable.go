// Package minikanren provides constraint programming abstractions.
// This file defines the Variable interface for constraint variables
// that can hold domains and participate in constraints.
package minikanren

import "fmt"

// Variable represents a decision variable in a constraint satisfaction problem.
// Variables have identities, domains of possible values, and participate in constraints.
//
// The Variable abstraction allows the solver to be agnostic to the underlying
// domain representation, enabling different domain types (finite domains,
// intervals, sets, etc.) to coexist in the same model.
//
// Variables in the Model hold initial domains and are immutable once solving begins.
// During solving, the Solver tracks domain changes via SolverState using the variable's ID.
type Variable interface {
	// ID returns a unique identifier for this variable within its model.
	// IDs are used for indexing and constraint tracking.
	ID() int

	// Domain returns the current domain of possible values for this variable.
	// During solving, domains shrink as constraints eliminate values.
	Domain() Domain

	// IsBound returns true if the variable's domain is a singleton.
	// Bound variables effectively have a single assigned value.
	IsBound() bool

	// Value returns the variable's value if IsBound() is true.
	// Panics if the variable is not bound.
	Value() int

	// String returns a human-readable representation of the variable.
	String() string
}

// FDVariable represents a finite-domain constraint variable.
// This is the standard variable type for finite-domain CSPs like Sudoku,
// N-Queens, scheduling, and resource allocation problems.
//
// FDVariable stores the initial domain. During solving, the Solver uses the
// variable's ID to track current domains in SolverState via copy-on-write.
// This separation enables:
//   - Model immutability (can be shared by parallel workers)
//   - Efficient O(1) state updates (only modified domains are tracked)
//   - Lock-free parallel search (each worker has its own SolverState chain)
type FDVariable struct {
	id     int    // Unique identifier within the model
	domain Domain // Current domain of possible values
	name   string // Optional name for debugging
}

// NewFDVariable creates a new finite-domain variable with the given ID and domain.
// The variable is initially unbound (domain may contain multiple values).
func NewFDVariable(id int, domain Domain) *FDVariable {
	return &FDVariable{
		id:     id,
		domain: domain,
		name:   fmt.Sprintf("v%d", id),
	}
}

// NewFDVariableWithName creates a named finite-domain variable for easier debugging.
func NewFDVariableWithName(id int, domain Domain, name string) *FDVariable {
	return &FDVariable{
		id:     id,
		domain: domain,
		name:   name,
	}
}

// ID returns the unique identifier of this variable.
func (v *FDVariable) ID() int {
	return v.id
}

// Domain returns the current domain of possible values.
func (v *FDVariable) Domain() Domain {
	return v.domain
}

// IsBound returns true if the variable has a single value in its domain.
func (v *FDVariable) IsBound() bool {
	return v.domain.IsSingleton()
}

// Value returns the bound value if the variable is bound.
// Panics if the variable is not bound.
func (v *FDVariable) Value() int {
	if !v.IsBound() {
		panic(fmt.Sprintf("Variable %s is not bound (domain size: %d)", v.name, v.domain.Count()))
	}
	return v.domain.SingletonValue()
}

// TryValue returns the variable's value if it is bound; otherwise it
// returns 0 together with a descriptive error. This provides a safe
// alternative to Value() for callers that prefer not to recover panics.
func (v *FDVariable) TryValue() (int, error) {
	if !v.IsBound() {
		return 0, fmt.Errorf("variable %s is not bound (domain size: %d)", v.name, v.domain.Count())
	}
	return v.domain.SingletonValue(), nil
}

// String returns a human-readable representation.
func (v *FDVariable) String() string {
	if v.IsBound() {
		return fmt.Sprintf("%s=%d", v.name, v.Value())
	}
	return fmt.Sprintf("%s∈%s", v.name, v.domain.String())
}

// Name returns the variable's name for debugging.
func (v *FDVariable) Name() string {
	return v.name
}

// SetDomain updates the variable's domain during model construction.
// This method must NOT be called during solving. During solving, domain changes
// are tracked via SolverState, not by modifying the variable directly.
func (v *FDVariable) SetDomain(domain Domain) {
	v.domain = domain
}
