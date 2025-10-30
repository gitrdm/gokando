package minikanren

import "fmt"

// Package minikanren provides FD constraint wrappers for the generic constraint system.
// These wrappers allow finite domain constraints to be used with the pluggable
// constraint manager while maintaining the separation between FD solving and
// general constraint logic.
//
// fd_constraints.go: FD constraint wrappers for the generic constraint system

// FDAllDifferentConstraint wraps all-different constraints for FD solving.
type FDAllDifferentConstraint struct {
	id        string
	variables []*Var
	isLocal   bool
}

// NewFDAllDifferentConstraint creates a new all-different constraint for FD variables.
func NewFDAllDifferentConstraint(variables []*Var) *FDAllDifferentConstraint {
	return &FDAllDifferentConstraint{
		id:        generateConstraintID("fd-alldiff"),
		variables: variables,
		isLocal:   false, // FD constraints typically require global coordination
	}
}

// ID returns the unique identifier for this constraint.
func (fdc *FDAllDifferentConstraint) ID() string {
	return fdc.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fdc *FDAllDifferentConstraint) IsLocal() bool {
	return fdc.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fdc *FDAllDifferentConstraint) Variables() []*Var {
	return fdc.variables
}

// Check evaluates the all-different constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fdc *FDAllDifferentConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// FD constraints are handled by the FD solver, not local checking
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fdc *FDAllDifferentConstraint) String() string {
	return fmt.Sprintf("(fd-alldiff %v)", fdc.variables)
}

// Clone creates a deep copy of the constraint.
func (fdc *FDAllDifferentConstraint) Clone() Constraint {
	vars := make([]*Var, len(fdc.variables))
	for i, v := range fdc.variables {
		if cloned, ok := v.Clone().(*Var); ok {
			vars[i] = cloned
		} else {
			// This shouldn't happen for *Var, but handle gracefully
			vars[i] = v
		}
	}
	return &FDAllDifferentConstraint{
		id:        fdc.id,
		variables: vars,
		isLocal:   fdc.isLocal,
	}
}

// FDOffsetConstraint wraps offset constraints (X = Y + offset) for FD solving.
type FDOffsetConstraint struct {
	id         string
	var1, var2 *Var
	offset     int
	isLocal    bool
}

// NewFDOffsetConstraint creates a new offset constraint for FD variables.
func NewFDOffsetConstraint(var1, var2 *Var, offset int) *FDOffsetConstraint {
	return &FDOffsetConstraint{
		id:      generateConstraintID("fd-offset"),
		var1:    var1,
		var2:    var2,
		offset:  offset,
		isLocal: false,
	}
}

// ID returns the unique identifier for this constraint.
func (foc *FDOffsetConstraint) ID() string {
	return foc.id
}

// IsLocal returns whether this constraint can be checked locally.
func (foc *FDOffsetConstraint) IsLocal() bool {
	return foc.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (foc *FDOffsetConstraint) Variables() []*Var {
	return []*Var{foc.var1, foc.var2}
}

// Check evaluates the offset constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (foc *FDOffsetConstraint) Check(bindings map[int64]Term) ConstraintResult {
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (foc *FDOffsetConstraint) String() string {
	return fmt.Sprintf("(fd-offset %s = %s + %d)", foc.var1.String(), foc.var2.String(), foc.offset)
}

// Clone creates a deep copy of the constraint.
func (foc *FDOffsetConstraint) Clone() Constraint {
	var var1, var2 *Var
	if cloned, ok := foc.var1.Clone().(*Var); ok {
		var1 = cloned
	} else {
		var1 = foc.var1
	}
	if cloned, ok := foc.var2.Clone().(*Var); ok {
		var2 = cloned
	} else {
		var2 = foc.var2
	}
	return &FDOffsetConstraint{
		id:      foc.id,
		var1:    var1,
		var2:    var2,
		offset:  foc.offset,
		isLocal: foc.isLocal,
	}
}

// FDInequalityConstraint wraps inequality constraints for FD solving.
type FDInequalityConstraint struct {
	id             string
	var1, var2     *Var
	inequalityType InequalityType
	isLocal        bool
}

// NewFDInequalityConstraint creates a new inequality constraint for FD variables.
func NewFDInequalityConstraint(var1, var2 *Var, inequalityType InequalityType) *FDInequalityConstraint {
	return &FDInequalityConstraint{
		id:             generateConstraintID("fd-ineq"),
		var1:           var1,
		var2:           var2,
		inequalityType: inequalityType,
		isLocal:        false,
	}
}

// ID returns the unique identifier for this constraint.
func (fic *FDInequalityConstraint) ID() string {
	return fic.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fic *FDInequalityConstraint) IsLocal() bool {
	return fic.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fic *FDInequalityConstraint) Variables() []*Var {
	return []*Var{fic.var1, fic.var2}
}

// Check evaluates the inequality constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fic *FDInequalityConstraint) Check(bindings map[int64]Term) ConstraintResult {
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fic *FDInequalityConstraint) String() string {
	return fmt.Sprintf("(fd-ineq %s %s %s)", fic.var1.String(), fic.inequalityType.String(), fic.var2.String())
}

// Clone creates a deep copy of the constraint.
func (fic *FDInequalityConstraint) Clone() Constraint {
	var var1, var2 *Var
	if cloned, ok := fic.var1.Clone().(*Var); ok {
		var1 = cloned
	} else {
		var1 = fic.var1
	}
	if cloned, ok := fic.var2.Clone().(*Var); ok {
		var2 = cloned
	} else {
		var2 = fic.var2
	}
	return &FDInequalityConstraint{
		id:             fic.id,
		var1:           var1,
		var2:           var2,
		inequalityType: fic.inequalityType,
		isLocal:        fic.isLocal,
	}
}

// FDCustomConstraintWrapper wraps custom FD constraints for the generic system.
type FDCustomConstraintWrapper struct {
	id               string
	variables        []*Var
	customConstraint CustomConstraint
	isLocal          bool
}

// NewFDCustomConstraintWrapper creates a wrapper for custom FD constraints.
func NewFDCustomConstraintWrapper(variables []*Var, customConstraint CustomConstraint) *FDCustomConstraintWrapper {
	return &FDCustomConstraintWrapper{
		id:               generateConstraintID("fd-custom"),
		variables:        variables,
		customConstraint: customConstraint,
		isLocal:          false,
	}
}

// ID returns the unique identifier for this constraint.
func (fccw *FDCustomConstraintWrapper) ID() string {
	return fccw.id
}

// IsLocal returns whether this constraint can be checked locally.
func (fccw *FDCustomConstraintWrapper) IsLocal() bool {
	return fccw.isLocal
}

// Variables returns the logic variables involved in this constraint.
func (fccw *FDCustomConstraintWrapper) Variables() []*Var {
	return fccw.variables
}

// Check evaluates the custom constraint.
// For FD constraints, this is a placeholder - actual checking happens in the FD solver.
func (fccw *FDCustomConstraintWrapper) Check(bindings map[int64]Term) ConstraintResult {
	return ConstraintPending
}

// String returns a human-readable representation of the constraint.
func (fccw *FDCustomConstraintWrapper) String() string {
	return fmt.Sprintf("(fd-custom %v)", fccw.variables)
}

// Clone creates a deep copy of the constraint.
func (fccw *FDCustomConstraintWrapper) Clone() Constraint {
	vars := make([]*Var, len(fccw.variables))
	for i, v := range fccw.variables {
		if cloned, ok := v.Clone().(*Var); ok {
			vars[i] = cloned
		} else {
			// This shouldn't happen for *Var, but handle gracefully
			vars[i] = v
		}
	}
	return &FDCustomConstraintWrapper{
		id:               fccw.id,
		variables:        vars,
		customConstraint: fccw.customConstraint, // Assume CustomConstraint is immutable
		isLocal:          fccw.isLocal,
	}
}
