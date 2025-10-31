package minikanren

import (
	"fmt"
)

// Package minikanren provides domain manipulation constraints for finite domain solving.
// These constraints enable declarative domain operations like fd/in, fd/dom, and fd/interval
// for custom domain specification and manipulation.
//
// fd_domains.go: Domain manipulation constraints and utilities

// FDDomainConstraint wraps domain specification constraints for FD solving.
// This corresponds to core.logic's fd/dom, constraining a variable to a specific domain.
type FDDomainConstraint struct {
	id       string
	variable *Var
	domain   BitSet
	isLocal  bool
}

// NewFDDomainConstraint creates a new domain constraint for FD variables.
// The variable will be constrained to have values only from the specified domain.
func NewFDDomainConstraint(variable *Var, domain BitSet) *FDDomainConstraint {
	return &FDDomainConstraint{
		id:       generateConstraintID("fd-domain"),
		variable: variable,
		domain:   domain,
		isLocal:  false, // FD constraints require global coordination
	}
}

// ID returns the unique identifier for this constraint.
func (fdc *FDDomainConstraint) ID() string {
	return fdc.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fdc *FDDomainConstraint) IsLocal() bool {
	return fdc.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fdc *FDDomainConstraint) Variables() []*Var {
	return []*Var{fdc.variable}
}

// Check evaluates the domain constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fdc *FDDomainConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// FD constraints are handled by the FD solver, not local checking
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fdc *FDDomainConstraint) String() string {
	return fmt.Sprintf("(fd/dom %v %v)", fdc.variable, fdc.domain)
}

// Clone creates a deep copy of the constraint.
func (fdc *FDDomainConstraint) Clone() Constraint {
	return &FDDomainConstraint{
		id:       fdc.id,
		variable: fdc.variable, // Variables are immutable
		domain:   fdc.domain.Clone(),
		isLocal:  fdc.isLocal,
	}
}

// FDInConstraint wraps domain membership constraints for FD solving.
// This corresponds to core.logic's fd/in, constraining a variable to be a member of a domain.
type FDInConstraint struct {
	id       string
	variable *Var
	values   []int
	isLocal  bool
}

// NewFDInConstraint creates a new domain membership constraint for FD variables.
// The variable will be constrained to have values only from the specified values slice.
func NewFDInConstraint(variable *Var, values []int) *FDInConstraint {
	return &FDInConstraint{
		id:       generateConstraintID("fd-in"),
		variable: variable,
		values:   append([]int(nil), values...), // copy slice
		isLocal:  false,
	}
}

// ID returns the unique identifier for this constraint.
func (fic *FDInConstraint) ID() string {
	return fic.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fic *FDInConstraint) IsLocal() bool {
	return fic.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fic *FDInConstraint) Variables() []*Var {
	return []*Var{fic.variable}
}

// Check evaluates the domain membership constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fic *FDInConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// FD constraints are handled by the FD solver, not local checking
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fic *FDInConstraint) String() string {
	return fmt.Sprintf("(fd/in %v %v)", fic.variable, fic.values)
}

// Clone creates a deep copy of the constraint.
func (fic *FDInConstraint) Clone() Constraint {
	values := make([]int, len(fic.values))
	copy(values, fic.values)
	return &FDInConstraint{
		id:       fic.id,
		variable: fic.variable,
		values:   values,
		isLocal:  fic.isLocal,
	}
}

// FDIntervalConstraint wraps interval domain constraints for FD solving.
// This corresponds to core.logic's fd/interval, constraining a variable to a range.
type FDIntervalConstraint struct {
	id       string
	variable *Var
	min, max int
	isLocal  bool
}

// NewFDIntervalConstraint creates a new interval constraint for FD variables.
// The variable will be constrained to have values in the range [min, max] inclusive.
func NewFDIntervalConstraint(variable *Var, min, max int) *FDIntervalConstraint {
	return &FDIntervalConstraint{
		id:       generateConstraintID("fd-interval"),
		variable: variable,
		min:      min,
		max:      max,
		isLocal:  false,
	}
}

// ID returns the unique identifier for this constraint.
func (fic *FDIntervalConstraint) ID() string {
	return fic.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fic *FDIntervalConstraint) IsLocal() bool {
	return fic.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fic *FDIntervalConstraint) Variables() []*Var {
	return []*Var{fic.variable}
}

// Check evaluates the interval constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fic *FDIntervalConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// FD constraints are handled by the FD solver, not local checking
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fic *FDIntervalConstraint) String() string {
	return fmt.Sprintf("(fd/interval %v %d %d)", fic.variable, fic.min, fic.max)
}

// Clone creates a deep copy of the constraint.
func (fic *FDIntervalConstraint) Clone() Constraint {
	return &FDIntervalConstraint{
		id:       fic.id,
		variable: fic.variable,
		min:      fic.min,
		max:      fic.max,
		isLocal:  fic.isLocal,
	}
}
