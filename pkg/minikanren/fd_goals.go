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
			fd := NewFDStoreWithDomain(domainSize)
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
			fd := NewFDStoreWithDomain(2 * n)

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
			fd := NewFDStore()
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
			fd := NewFDStore()
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
