package minikanren

import (
	"context"
	"fmt"
)

// VariableMapper manages the mapping between logic variables and FD variables.
// It ensures consistent variable assignment during constraint application and
// solution extraction.
type VariableMapper struct {
	// logicToFD maps logic variable IDs to FD variables
	logicToFD map[int64]*FDVar

	// fdToLogic maps FD variable IDs to logic variables
	fdToLogic map[int]*Var
}

// NewVariableMapper creates a new variable mapper.
func NewVariableMapper() *VariableMapper {
	return &VariableMapper{
		logicToFD: make(map[int64]*FDVar),
		fdToLogic: make(map[int]*Var),
	}
}

// GetOrCreateFDVar gets an existing FD variable for a logic variable,
// or creates a new one if it doesn't exist.
func (vm *VariableMapper) GetOrCreateFDVar(logicVar *Var, fdStore *FDStore) *FDVar {
	if fdVar, exists := vm.logicToFD[logicVar.id]; exists {
		return fdVar
	}

	fdVar := fdStore.NewVar()
	vm.logicToFD[logicVar.id] = fdVar
	vm.fdToLogic[fdVar.ID] = logicVar
	return fdVar
}

// GetLogicVar returns the logic variable for an FD variable ID.
func (vm *VariableMapper) GetLogicVar(fdVarID int) (*Var, bool) {
	logicVar, exists := vm.fdToLogic[fdVarID]
	return logicVar, exists
}

// GetAllLogicVars returns all logic variables that have been mapped.
func (vm *VariableMapper) GetAllLogicVars() []*Var {
	vars := make([]*Var, 0, len(vm.fdToLogic))
	for _, logicVar := range vm.fdToLogic {
		vars = append(vars, logicVar)
	}
	return vars
}

// Package minikanren provides the FDSolver implementation for finite domain constraint solving.
// The FDSolver adapts the existing FDStore to work as a pluggable solver component
// within the constraint manager architecture.
//
// FDSolver implements the Solver interface for finite domain constraint solving.
// It adapts the existing FDStore to work as a pluggable solver component
// within the constraint manager architecture.
//
// This solver is specialized for:
//   - Finite domain variables with integer domains
//   - AllDifferent constraints with Regin filtering
//   - Arithmetic offset constraints
//   - Inequality constraints
//   - Custom user-defined constraints
//
// The solver integrates FD solving capabilities into the broader
// constraint solving ecosystem while maintaining performance.
type FDSolver struct {
	*BaseSolver

	// domainSize specifies the default domain size (1..domainSize) for variables
	domainSize int

	// maxSolutions limits the number of solutions to find (0 = unlimited)
	maxSolutions int

	// config holds solver configuration for heuristics and settings
	config *SolverConfig
}

// NewFDSolver creates a new finite domain solver with the specified configuration.
func NewFDSolver(id, name string, domainSize, maxSolutions int, config *SolverConfig) *FDSolver {
	if config == nil {
		config = DefaultSolverConfig()
	}

	return &FDSolver{
		BaseSolver: NewBaseSolver(id, name, []string{
			"FDAllDifferentConstraint",
			"FDOffsetConstraint",
			"FDInequalityConstraint",
			"FDCustomConstraint",
			"TypeConstraint", // FD solver can also handle basic type constraints
		}, 5), // High priority for FD constraints
		domainSize:   domainSize,
		maxSolutions: maxSolutions,
		config:       config,
	}
}

// Solve implements the Solver interface using finite domain solving.
// This method handles FD-specific constraints by creating an FDStore,
// applying the constraints, and solving the resulting CSP.
func (fds *FDSolver) Solve(ctx context.Context, constraint Constraint, store ConstraintStore) (ConstraintStore, error) {
	// Check if we can handle this constraint type
	if !fds.CanHandle(constraint) {
		return nil, fmt.Errorf("FD solver cannot handle constraint type %T", constraint)
	}

	// Create FD store for solving
	fdStore := NewFDStoreWithConfig(fds.domainSize, fds.config)

	// Create variable mapping between logic variables and FD variables
	varMapper := NewVariableMapper()

	// Apply the constraint to the FD store
	if err := fds.applyConstraintToFDStore(ctx, constraint, store, fdStore, varMapper); err != nil {
		return nil, fmt.Errorf("failed to apply constraint to FD store: %w", err)
	}

	// Solve the FD problem
	solutions, err := fdStore.Solve(ctx, fds.maxSolutions)
	if err != nil {
		return nil, fmt.Errorf("FD solving failed: %w", err)
	}

	if len(solutions) == 0 {
		return nil, fmt.Errorf("no solutions found for FD constraint")
	}

	// Apply the first solution to the constraint store
	resultStore := store.Clone()
	if err := fds.applySolutionToConstraintStore(solutions[0], resultStore, varMapper); err != nil {
		return nil, fmt.Errorf("failed to apply solution to constraint store: %w", err)
	}

	return resultStore, nil
}

// applyConstraintToFDStore converts a generic constraint to FD operations.
// This method handles the translation from the generic Constraint interface
// to FD-specific operations on an FDStore.
func (fds *FDSolver) applyConstraintToFDStore(ctx context.Context, constraint Constraint, logicStore ConstraintStore, fdStore *FDStore, varMapper *VariableMapper) error {
	switch c := constraint.(type) {
	case *FDAllDifferentConstraint:
		return fds.applyFDAllDifferent(ctx, c, fdStore, varMapper)
	case *FDOffsetConstraint:
		return fds.applyFDOffset(ctx, c, fdStore, varMapper)
	case *FDInequalityConstraint:
		return fds.applyFDInequality(ctx, c, fdStore, varMapper)
	case *FDCustomConstraintWrapper:
		return fds.applyFDCustom(ctx, c, fdStore, varMapper)
	case *TypeConstraint:
		// Type constraints don't require FD solving - just return success
		// The constraint will be checked by the constraint manager
		return nil
	default:
		return fmt.Errorf("unsupported constraint type for FD solver: %T", constraint)
	}
}

// applyFDAllDifferent applies an all-different constraint to the FD store.
func (fds *FDSolver) applyFDAllDifferent(ctx context.Context, constraint *FDAllDifferentConstraint, fdStore *FDStore, varMapper *VariableMapper) error {
	// Convert logic variables to FD variables
	fdVars := make([]*FDVar, len(constraint.variables))
	for i, logicVar := range constraint.variables {
		fdVars[i] = varMapper.GetOrCreateFDVar(logicVar, fdStore)
	}

	// Apply all-different constraint with Regin filtering
	return fdStore.AddAllDifferentRegin(fdVars)
}

// applyFDOffset applies an offset constraint (X = Y + offset) to the FD store.
func (fds *FDSolver) applyFDOffset(ctx context.Context, constraint *FDOffsetConstraint, fdStore *FDStore, varMapper *VariableMapper) error {
	// Create FD variables for the constraint
	srcVar := varMapper.GetOrCreateFDVar(constraint.var1, fdStore)
	dstVar := varMapper.GetOrCreateFDVar(constraint.var2, fdStore)

	// Apply offset constraint
	return fdStore.AddOffsetConstraint(srcVar, constraint.offset, dstVar)
}

// applyFDInequality applies an inequality constraint to the FD store.
func (fds *FDSolver) applyFDInequality(ctx context.Context, constraint *FDInequalityConstraint, fdStore *FDStore, varMapper *VariableMapper) error {
	// Create FD variables for the constraint
	var1 := varMapper.GetOrCreateFDVar(constraint.var1, fdStore)
	var2 := varMapper.GetOrCreateFDVar(constraint.var2, fdStore)

	// Apply inequality constraint
	return fdStore.AddInequalityConstraint(var1, var2, constraint.inequalityType)
}

// applyFDCustom applies a custom constraint to the FD store.
func (fds *FDSolver) applyFDCustom(ctx context.Context, constraint *FDCustomConstraintWrapper, fdStore *FDStore, varMapper *VariableMapper) error {
	// Convert logic variables to FD variables
	fdVars := make([]*FDVar, len(constraint.variables))
	for i, logicVar := range constraint.variables {
		fdVars[i] = varMapper.GetOrCreateFDVar(logicVar, fdStore)
	}

	// Apply the custom constraint
	return fdStore.AddCustomConstraint(constraint.customConstraint)
}

// applySolutionToConstraintStore applies an FD solution back to the logic constraint store.
// This method maps FD variable assignments to logic variable bindings.
func (fds *FDSolver) applySolutionToConstraintStore(solution []int, store ConstraintStore, varMapper *VariableMapper) error {
	// Apply each FD variable assignment to the corresponding logic variable
	for fdVarID, value := range solution {
		if logicVar, exists := varMapper.GetLogicVar(fdVarID); exists {
			// Create an atom with the integer value
			valueAtom := NewAtom(value)

			// Add the binding to the constraint store
			if err := store.AddBinding(logicVar.id, valueAtom); err != nil {
				return fmt.Errorf("failed to add binding for variable %s: %w", logicVar.String(), err)
			}
		}
	}

	return nil
}
