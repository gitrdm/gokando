package minikanren

import (
	"context"
)

// FDAllDifferentGoal creates a Goal that enforces an all-different constraint
// over the provided logic variables. domainSize specifies the integer domain
// (values 1..domainSize). The goal, when executed, will enumerate all
// assignments that satisfy the AllDifferent constraint and existing bindings
// in the provided ConstraintStore.
func FDAllDifferentGoal(vars []*Var, domainSize int) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Build FD store and variables
			var fd *FDStore
			if strategy, ok := ctx.Value(searchStrategyKey).(SearchStrategy); ok {
				// Use strategy from context
				config := &StrategyConfig{
					Labeling: NewFirstFailLabeling(),
					Search:   strategy,
				}
				fd = NewFDStoreWithStrategy(domainSize, config)
			} else {
				// Use default strategy
				fd = NewFDStoreWithDomain(domainSize)
			}
			fdVars := make([]*FDVar, 0, len(vars))
			for range vars {
				fdVars = append(fdVars, fd.NewVar())
			}

			// Add Regin all-different filtering
			if err := fd.AddAllDifferentRegin(fdVars); err != nil {
				return
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			for i, v := range vars {
				walked := sub.DeepWalk(v)
				if !walked.IsVar() {
					if atom, ok := walked.(*Atom); ok {
						// expect integer atom
						if val, ok2 := atom.Value().(int); ok2 {
							if err := fd.Assign(fdVars[i], val); err != nil {
								return
							}
						} else {
							// Non-integer binding - cannot satisfy
							return
						}
					} else {
						// Bound to complex term - not supported
						return
					}
				}
			}

			// Solve all solutions
			sols, err := fd.Solve(context.Background(), 0)
			if err != nil {
				return
			}

			for _, sol := range sols {
				// For each solution, clone original store and add bindings
				cloned := store.Clone()
				ok := true
				for i, v := range vars {
					// bind var.id to Atom(sol[i])
					if err := cloned.AddBinding(v.id, NewAtom(sol[i])); err != nil {
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
}

// FDQueensGoal models N-Queens using the FD engine idiomatically:
// - column variables range 1..n
// - derived diagonal variables are created as offsets of columns
// - AllDifferent is applied to columns and both diagonal sets
func FDQueensGoal(vars []*Var, n int) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// FD domain size: allow shifts for diagonals (Ci +/- i). Use 2n to be safe.
			var fd *FDStore
			if strategy, ok := ctx.Value(searchStrategyKey).(SearchStrategy); ok {
				// Use strategy from context
				config := &StrategyConfig{
					Labeling: NewFirstFailLabeling(),
					Search:   strategy,
				}
				fd = NewFDStoreWithStrategy(2*n, config)
			} else {
				// Use default strategy
				fd = NewFDStoreWithDomain(2 * n)
			}

			// create FD variables matching logic vars
			fdVars := make([]*FDVar, 0, len(vars))
			for range vars {
				fdVars = append(fdVars, fd.NewVar())
			}

			// constrain queen columns to 1..n (remove values n+1 .. 2n)
			for _, v := range fdVars {
				for i := n + 1; i <= 2*n; i++ {
					if err := fd.Remove(v, i); err != nil {
						return
					}
				}
			}

			// derived diagonal vars
			d1 := make([]*FDVar, n)
			d2 := make([]*FDVar, n)
			for i := 0; i < n; i++ {
				d1[i] = fd.NewVar()
				d2[i] = fd.NewVar()
			}

			// link offsets: d1 = C + i ; d2 = C - i + n
			for i := 0; i < n; i++ {
				// dst = src + offset
				if err := fd.AddOffsetConstraint(fdVars[i], i, d1[i]); err != nil {
					return
				}
				if err := fd.AddOffsetConstraint(fdVars[i], -i+n, d2[i]); err != nil {
					return
				}
			}

			// all-different on columns and diagonals
			if err := fd.AddAllDifferentRegin(fdVars); err != nil {
				return
			}
			if err := fd.AddAllDifferentRegin(d1); err != nil {
				return
			}
			if err := fd.AddAllDifferentRegin(d2); err != nil {
				return
			}

			// apply current bindings from the constraint store
			sub := store.GetSubstitution()
			for i, v := range vars {
				walked := sub.DeepWalk(v)
				if !walked.IsVar() {
					if atom, ok := walked.(*Atom); ok {
						if val, ok2 := atom.Value().(int); ok2 {
							// expect 1..n values
							if err := fd.Assign(fdVars[i], val); err != nil {
								return
							}
						} else {
							return
						}
					} else {
						return
					}
				}
			}

			sols, err := fd.Solve(context.Background(), 0)
			if err != nil {
				return
			}

			for _, sol := range sols {
				cloned := store.Clone()
				ok := true
				for i, v := range vars {
					if err := cloned.AddBinding(v.id, NewAtom(sol[i])); err != nil {
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
}

// FDInequalityGoal creates a goal that enforces an inequality constraint between two variables
func FDInequalityGoal(x, y *Var, typ InequalityType) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Create FD store and variables
			var fd *FDStore
			if strategy, ok := ctx.Value(searchStrategyKey).(SearchStrategy); ok {
				// Use strategy from context
				config := &StrategyConfig{
					Labeling: NewFirstFailLabeling(),
					Search:   strategy,
				}
				fd = NewFDStoreWithStrategy(100, config) // Default domain size
			} else {
				// Use default strategy
				fd = NewFDStore()
			}
			fdX := fd.NewVar()
			fdY := fd.NewVar()

			// Add inequality constraint
			if err := fd.AddInequalityConstraint(fdX, fdY, typ); err != nil {
				return
			}

			// Apply current bindings
			sub := store.GetSubstitution()

			xWalked := sub.DeepWalk(x)
			if !xWalked.IsVar() {
				if atom, ok := xWalked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						if err := fd.Assign(fdX, val); err != nil {
							return
						}
					}
				}
			}

			yWalked := sub.DeepWalk(y)
			if !yWalked.IsVar() {
				if atom, ok := yWalked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						if err := fd.Assign(fdY, val); err != nil {
							return
						}
					}
				}
			}

			// Solve and generate solutions
			sols, err := fd.Solve(context.Background(), 0)
			if err != nil {
				return
			}

			for _, sol := range sols {
				cloned := store.Clone()
				ok := true

				// Bind x if it was solved
				if len(sol) > 0 {
					if err := cloned.AddBinding(x.id, NewAtom(sol[0])); err != nil {
						ok = false
					}
				}

				// Bind y if it was solved
				if len(sol) > 1 {
					if err := cloned.AddBinding(y.id, NewAtom(sol[1])); err != nil {
						ok = false
					}
				}

				if ok {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}

// FDCustomGoal creates a goal that enforces a custom constraint
func FDCustomGoal(vars []*Var, constraint CustomConstraint) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Create FD store and map variables
			var fd *FDStore
			if strategy, ok := ctx.Value(searchStrategyKey).(SearchStrategy); ok {
				// Use strategy from context
				config := &StrategyConfig{
					Labeling: NewFirstFailLabeling(),
					Search:   strategy,
				}
				fd = NewFDStoreWithStrategy(100, config) // Default domain size
			} else {
				// Use default strategy
				fd = NewFDStore()
			}
			varMap := make(map[*Var]*FDVar)

			constraintVars := constraint.Variables()
			if len(vars) != len(constraintVars) {
				return // Mismatch in variable count
			}

			fdVars := make([]*FDVar, len(vars))
			for i, v := range vars {
				fdVars[i] = fd.NewVar()
				varMap[v] = fdVars[i]
			}

			// Add custom constraint
			if err := fd.AddCustomConstraint(constraint); err != nil {
				return
			}

			// Apply current bindings
			sub := store.GetSubstitution()
			for logicVar, fdVar := range varMap {
				walked := sub.DeepWalk(logicVar)
				if !walked.IsVar() {
					if atom, ok := walked.(*Atom); ok {
						if val, ok2 := atom.Value().(int); ok2 {
							if err := fd.Assign(fdVar, val); err != nil {
								return
							}
						}
					}
				}
			}

			// Solve and generate solutions
			sols, err := fd.Solve(context.Background(), 0)
			if err != nil {
				return
			}

			for _, sol := range sols {
				cloned := store.Clone()
				ok := true

				for i, logicVar := range vars {
					if i < len(sol) {
						if err := cloned.AddBinding(logicVar.id, NewAtom(sol[i])); err != nil {
							ok = false
							break
						}
					}
				}

				if ok {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}

// FDDomainGoal creates a Goal that constrains a variable to a specific domain.
// This corresponds to core.logic's fd/dom, allowing declarative domain specification.
// The variable will be constrained to have values only from the specified BitSet domain.
func FDDomainGoal(variable *Var, domain BitSet) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Check if domain is empty
			if domain.Count() == 0 {
				return // No solutions
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			walked := sub.DeepWalk(variable)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						// Check if the bound value is in the domain
						if domain.Has(val) {
							// Value is valid, return the current store
							stream.Put(ctx, store)
						}
						// If not in domain, no solutions
						return
					} else {
						// Non-integer binding - cannot satisfy
						return
					}
				} else {
					// Bound to complex term - not supported
					return
				}
			}

			// Variable is unbound - enumerate all values in the domain
			domain.IterateValues(func(val int) {
				cloned := store.Clone()
				if err := cloned.AddBinding(variable.id, NewAtom(val)); err == nil {
					stream.Put(ctx, cloned)
				}
			})
		}()

		return stream
	}
}

// FDInGoal creates a Goal that constrains a variable to be a member of a set of values.
// This corresponds to core.logic's fd/in, allowing variables to have custom value sets.
// The variable will be constrained to have values only from the specified values slice.
func FDInGoal(variable *Var, values []int) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			if len(values) == 0 {
				// Empty domain - no solutions
				return
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			walked := sub.DeepWalk(variable)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						// Check if the bound value is in the values list
						for _, allowedVal := range values {
							if val == allowedVal {
								stream.Put(ctx, store)
								return
							}
						}
						// Value not in allowed list - no solutions
						return
					} else {
						// Non-integer binding - cannot satisfy
						return
					}
				} else {
					// Bound to complex term - not supported
					return
				}
			}

			// Variable is unbound - enumerate all values in the list
			for _, val := range values {
				cloned := store.Clone()
				if err := cloned.AddBinding(variable.id, NewAtom(val)); err == nil {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}

// FDIntervalGoal creates a Goal that constrains a variable to a range of values.
// This corresponds to core.logic's fd/interval, allowing variables to have interval domains.
// The variable will be constrained to have values in the range [min, max] inclusive.
func FDIntervalGoal(variable *Var, min, max int) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			if min > max || min < 1 {
				// Invalid interval - no solutions
				return
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			walked := sub.DeepWalk(variable)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						// Check if the bound value is in the interval
						if val >= min && val <= max {
							stream.Put(ctx, store)
						}
						// If not in interval, no solutions
						return
					} else {
						// Non-integer binding - cannot satisfy
						return
					}
				} else {
					// Bound to complex term - not supported
					return
				}
			}

			// Variable is unbound - enumerate all values in the interval
			for val := min; val <= max; val++ {
				cloned := store.Clone()
				if err := cloned.AddBinding(variable.id, NewAtom(val)); err == nil {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}

// FDGoalOption represents a functional option for configuring FD goals
type FDGoalOption func(*fdGoalConfig)

type fdGoalConfig struct {
	domainSize int
	strategy   SearchStrategy
	labeling   LabelingStrategy
}

// WithDomainSize sets the domain size for FD goals that need it
func WithDomainSize(size int) FDGoalOption {
	return func(c *fdGoalConfig) {
		c.domainSize = size
	}
}

// WithSearchStrategy sets the search strategy for FD goals
func WithSearchStrategy(strategy SearchStrategy) FDGoalOption {
	return func(c *fdGoalConfig) {
		c.strategy = strategy
	}
}

// WithLabelingStrategy sets the labeling strategy for FD goals
func WithLabelingStrategy(labeling LabelingStrategy) FDGoalOption {
	return func(c *fdGoalConfig) {
		c.labeling = labeling
	}
}

// FDAllDifferent creates a Goal that enforces an all-different constraint
// over the provided logic variables using functional options for configuration.
//
// Example:
//
//	x, y, z := Fresh("x"), Fresh("y"), Fresh("z")
//	goal := FDAllDifferent(x, y, z, WithDomainSize(9), WithSearchStrategy(NewFirstFailLabeling()))
func FDAllDifferent(vars ...interface{}) Goal {
	// Extract variables and options
	var logicVars []*Var
	var options []FDGoalOption

	for _, arg := range vars {
		switch v := arg.(type) {
		case *Var:
			logicVars = append(logicVars, v)
		case FDGoalOption:
			options = append(options, v)
		}
	}

	// Apply default configuration
	config := &fdGoalConfig{
		domainSize: 100,            // Default domain size
		strategy:   NewDFSSearch(), // Default search strategy
		labeling:   NewFirstFailLabeling(),
	}

	// Apply options
	for _, opt := range options {
		opt(config)
	}

	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			// Build FD store with strategy
			strategyConfig := &StrategyConfig{
				Labeling: config.labeling,
				Search:   config.strategy,
			}
			fd := NewFDStoreWithStrategy(config.domainSize, strategyConfig)

			fdVars := make([]*FDVar, 0, len(logicVars))
			for range logicVars {
				fdVars = append(fdVars, fd.NewVar())
			}

			// Add Regin all-different filtering
			if err := fd.AddAllDifferentRegin(fdVars); err != nil {
				return
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			for i, v := range logicVars {
				walked := sub.DeepWalk(v)
				if !walked.IsVar() {
					if atom, ok := walked.(*Atom); ok {
						if val, ok2 := atom.Value().(int); ok2 {
							if err := fd.Assign(fdVars[i], val); err != nil {
								return
							}
						} else {
							return
						}
					} else {
						return
					}
				}
			}

			// Solve all solutions
			sols, err := fd.Solve(context.Background(), 0)
			if err != nil {
				return
			}

			for _, sol := range sols {
				cloned := store.Clone()
				ok := true
				for i, v := range logicVars {
					if err := cloned.AddBinding(v.id, NewAtom(sol[i])); err != nil {
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
}

// FDIn creates a Goal that constrains variables to be members of value sets
// using functional options for configuration.
//
// Example:
//
//	x := Fresh("x")
//	goal := FDIn(x, []int{1, 3, 5, 7, 9})
func FDIn(variable *Var, values []int, options ...FDGoalOption) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			if len(values) == 0 {
				return // Empty domain - no solutions
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			walked := sub.DeepWalk(variable)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						for _, allowedVal := range values {
							if val == allowedVal {
								stream.Put(ctx, store)
								return
							}
						}
						return // Value not in allowed list
					} else {
						return // Non-integer binding
					}
				} else {
					return // Bound to complex term
				}
			}

			// Variable is unbound - enumerate all values in the list
			for _, val := range values {
				cloned := store.Clone()
				if err := cloned.AddBinding(variable.id, NewAtom(val)); err == nil {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}

// FDInterval creates a Goal that constrains a variable to a range of values
// using functional options for configuration.
//
// Example:
//
//	x := Fresh("x")
//	goal := FDInterval(x, 1, 9)
func FDInterval(variable *Var, min, max int, options ...FDGoalOption) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			if min > max || min < 1 {
				return // Invalid interval
			}

			// Apply current bindings from the constraint store
			sub := store.GetSubstitution()
			walked := sub.DeepWalk(variable)
			if !walked.IsVar() {
				if atom, ok := walked.(*Atom); ok {
					if val, ok2 := atom.Value().(int); ok2 {
						if val >= min && val <= max {
							stream.Put(ctx, store)
						}
						return
					} else {
						return
					}
				} else {
					return
				}
			}

			// Variable is unbound - enumerate all values in the interval
			for val := min; val <= max; val++ {
				cloned := store.Clone()
				if err := cloned.AddBinding(variable.id, NewAtom(val)); err == nil {
					stream.Put(ctx, cloned)
				}
			}
		}()

		return stream
	}
}
