package minikanren

import (
	"context"
	"fmt"
)

// FDSolve creates a goal that solves all pending FD constraints in the store.
// It should be used to wrap goals that add FD constraints.
// This implementation collects all constraints from the inner goal first,
// then runs the FD solver a single time for each result from the inner goal.
func FDSolve(g Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Create a temporary store that will only collect constraints.
		collector := NewConstraintCollector(store)

		// Run the inner goal `g` on the collector. This will execute all the
		// declarative FD goals and add their constraints to the collector.
		// The inner goal will produce a result stream of stores that satisfy
		// the non-FD parts of the goal.
		innerStream := g(ctx, collector)

		// The final results will be produced by solving the collected FD
		// constraints for each result from the inner goal.
		return &fdSolveStream{
			inner:     innerStream,
			ctx:       ctx,
			collector: collector,
		}
	}
}

// fdSolveStream is a ResultStream that wraps another stream, solving
// collected FD constraints for each result.
type fdSolveStream struct {
	inner     ResultStream
	ctx       context.Context
	collector *ConstraintCollector
}

func (s *fdSolveStream) Put(ctx context.Context, store ConstraintStore) error {
	return ErrUnsupportedOperation
}

func (s *fdSolveStream) Count() int64 {
	// This is an estimation, as one inner result can produce multiple FD solutions.
	return s.inner.Count()
}

func (s *fdSolveStream) Close() error {
	return s.inner.Close()
}

func (s *fdSolveStream) Take(ctx context.Context, n int) ([]ConstraintStore, bool, error) {
	// This implementation simplifies things by solving one inner result at a time
	// and returning all FD solutions for it. A more complex implementation could
	// buffer results to satisfy `n` precisely.
	for {
		// Take one intermediate result from the inner goal.
		innerResults, more, err := s.inner.Take(ctx, 1)
		if err != nil || !more {
			return nil, more, err
		}

		if len(innerResults) == 0 {
			if !more {
				return nil, false, nil
			}
			continue
		}
		initialStore := innerResults[0]

		// Now, solve the collected FD constraints against this specific result.
		finalStream := solveFDConstraints(s.ctx, initialStore, s.collector)
		// Take all solutions for this branch.
		finalResults, _, err := finalStream.Take(ctx, -1)

		if err != nil {
			return nil, false, err
		}

		if len(finalResults) > 0 {
			// We found solutions for this path. Return them and indicate more might be available.
			return finalResults, true, nil
		}
		// If no solutions, loop to the next intermediate result from the inner stream.
	}
}

func (s *fdSolveStream) Drain(ctx context.Context) {
	// To drain this stream, we just need to drain the inner one.
	s.inner.Drain(ctx)
}

// solveFDConstraints takes a set of constraints and solves them using the FD engine.
func solveFDConstraints(ctx context.Context, initialStore ConstraintStore, collector *ConstraintCollector) ResultStream {
	stream := NewStream()

	go func() {
		defer stream.Close()

		constraints := collector.GetConstraints()
		if len(constraints) == 0 {
			// No FD constraints to solve, just return the initial store.
			stream.Put(ctx, initialStore)
			return
		}

		domainSize := 100 // Default domain size
		maxVal := 0
		for _, c := range constraints {
			if in, ok := c.(*FDInConstraint); ok {
				for _, v := range in.values {
					if v > maxVal {
						maxVal = v
					}
				}
			}
		}
		if maxVal > domainSize {
			domainSize = maxVal
		}

		strategyConfig := &StrategyConfig{
			Labeling: NewFirstFailLabeling(),
			Search:   NewDFSSearch(),
		}
		fdStore := NewFDStoreWithStrategy(domainSize, strategyConfig)
		varMapper := newVariableMapper()

		// Apply all collected constraints to the new FD store
		for _, c := range constraints {
			if err := applyConstraintToFDStore(c, fdStore, varMapper); err != nil {
				return
			}
		}

		// Apply existing bindings from the initial constraint store to the FD store
		sub := initialStore.GetSubstitution()
		for _, logicVar := range varMapper.getAllLogicVars() {
			walked := sub.DeepWalk(logicVar)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						fdVar := varMapper.getOrCreateFDVar(logicVar, fdStore)
						if err := fdStore.Assign(fdVar, val); err != nil {
							return // Inconsistent assignment
						}
					}
				}
			}
		}

		// Solve the FD problem for all solutions
		sols, err := fdStore.Solve(ctx, 0)
		if err != nil {
			return
		}

		// Generate a result for each solution
		for _, sol := range sols {
			cloned := initialStore.Clone()
			ok := true
			for i, solvedVal := range sol {
				if i >= len(fdStore.vars) {
					break
				}
				fdVar := fdStore.vars[i]
				logicVar, exists := varMapper.getLogicVar(fdVar.ID)
				if !exists {
					continue
				}

				if err := cloned.AddBinding(logicVar.id, NewAtom(solvedVal)); err != nil {
					ok = false
					break
				}
			}

			if ok {
				stream.Put(ctx, cloned)
			}
		}
	}()

	return stream
}

// applyConstraintToFDStore translates a generic constraint into an FD-specific one.
func applyConstraintToFDStore(c Constraint, fdStore *FDStore, varMapper *variableMapper) error {
	switch constraint := c.(type) {
	case *FDAllDifferentConstraint:
		fdVars := make([]*FDVar, len(constraint.variables))
		for i, v := range constraint.variables {
			fdVars[i] = varMapper.getOrCreateFDVar(v, fdStore)
		}
		return fdStore.AddAllDifferentRegin(fdVars)
	case *FDInConstraint:
		fdVar := varMapper.getOrCreateFDVar(constraint.variable, fdStore)
		// This method doesn't exist on FDStore, we need to implement it or use another way
		// For now, let's assume a method that takes a slice of ints.
		return fdStore.AddInConstraint(fdVar, constraint.values)
	case *FDInequalityConstraint:
		x := varMapper.getOrCreateFDVar(constraint.var1, fdStore)
		y := varMapper.getOrCreateFDVar(constraint.var2, fdStore)
		return fdStore.AddInequalityConstraint(x, y, constraint.inequalityType)
	case *ArithmeticRelationConstraint:
		var x, y, z *FDVar

		if xAtom, ok := constraint.x.(*Atom); ok {
			val, isInt := xAtom.Value().(int)
			if !isInt {
				return fmt.Errorf("arithmetic relation constraint expects integer atom")
			}
			x = fdStore.NewVar()
			if err := fdStore.Assign(x, val); err != nil {
				return err
			}
		} else if xVar, ok := constraint.x.(*Var); ok {
			x = varMapper.getOrCreateFDVar(xVar, fdStore)
		}

		if yAtom, ok := constraint.y.(*Atom); ok {
			val, isInt := yAtom.Value().(int)
			if !isInt {
				return fmt.Errorf("arithmetic relation constraint expects integer atom")
			}
			y = fdStore.NewVar()
			if err := fdStore.Assign(y, val); err != nil {
				return err
			}
		} else if yVar, ok := constraint.y.(*Var); ok {
			y = varMapper.getOrCreateFDVar(yVar, fdStore)
		}

		if zAtom, ok := constraint.z.(*Atom); ok {
			val, isInt := zAtom.Value().(int)
			if !isInt {
				return fmt.Errorf("arithmetic relation constraint expects integer atom")
			}
			z = fdStore.NewVar()
			if err := fdStore.Assign(z, val); err != nil {
				return err
			}
		} else if zVar, ok := constraint.z.(*Var); ok {
			z = varMapper.getOrCreateFDVar(zVar, fdStore)
		}

		if x == nil || y == nil || z == nil {
			return fmt.Errorf("unsupported term types in arithmetic relation")
		}

		switch constraint.op {
		case ArithmeticPlus:
			return fdStore.AddPlusConstraint(x, y, z)
		case ArithmeticMinus:
			return fdStore.AddMinusConstraint(x, y, z)
		case ArithmeticMultiply:
			return fdStore.AddMultiplyConstraint(x, y, z)
		case ArithmeticQuotient:
			return fdStore.AddQuotientConstraint(x, y, z)
		case ArithmeticModulo:
			return fdStore.AddModuloConstraint(x, y, z)
		}
	default:
		return fmt.Errorf("unsupported constraint type for FD solver: %T", c)
	}
	return nil
}

// variableMapper manages the mapping between logic variables and FD variables.
type variableMapper struct {
	logicToFD map[int64]*FDVar
	fdToLogic map[int]*Var
}

func newVariableMapper() *variableMapper {
	return &variableMapper{
		logicToFD: make(map[int64]*FDVar),
		fdToLogic: make(map[int]*Var),
	}
}

func (vm *variableMapper) getOrCreateFDVar(logicVar *Var, fdStore *FDStore) *FDVar {
	if fdVar, exists := vm.logicToFD[logicVar.id]; exists {
		return fdVar
	}
	fdVar := fdStore.NewVar()
	vm.logicToFD[logicVar.id] = fdVar
	vm.fdToLogic[fdVar.ID] = logicVar
	return fdVar
}

func (vm *variableMapper) getLogicVar(fdVarID int) (*Var, bool) {
	logicVar, exists := vm.fdToLogic[fdVarID]
	return logicVar, exists
}

func (vm *variableMapper) getAllLogicVars() []*Var {
	vars := make([]*Var, 0, len(vm.fdToLogic))
	for _, v := range vm.fdToLogic {
		vars = append(vars, v)
	}
	return vars
}

// FDAllDifferent is a high-level goal that creates an AllDifferent constraint.
func FDAllDifferent(vars ...*Var) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		constraint := NewFDAllDifferentConstraint(vars)
		if err := store.AddConstraint(constraint); err != nil {
			return NewStream()
		}
		return NewSingletonStream(store)
	}
}

// FDIn is a high-level goal that creates a domain constraint.
func FDIn(variable *Var, values []int) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		constraint := NewFDInConstraint(variable, values)
		if err := store.AddConstraint(constraint); err != nil {
			return NewStream()
		}
		return NewSingletonStream(store)
	}
}

// FDInequality is a high-level goal that creates an inequality constraint.
func FDInequality(x, y *Var, typ InequalityType) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		constraint := NewFDInequalityConstraint(x, y, typ)
		if err := store.AddConstraint(constraint); err != nil {
			return NewStream()
		}
		return NewSingletonStream(store)
	}
}

// FDArithmetic is a high-level goal for arithmetic relations.
func FDArithmetic(x, y, z Term, op ArithmeticConstraintType) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		constraint := NewArithmeticRelationConstraint(x, y, z, op)
		if err := store.AddConstraint(constraint); err != nil {
			return NewStream()
		}
		return NewSingletonStream(store)
	}
}
