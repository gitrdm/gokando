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
	return func(ctx context.Context, store ConstraintStore) *Stream {
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
			if !fd.AddAllDifferentRegin(fdVars) {
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
							okAssign := fd.Assign(fdVars[i], val)
							if !okAssign {
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
					stream.Put(cloned)
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
	return func(ctx context.Context, store ConstraintStore) *Stream {
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
					if !fd.Remove(v, i) {
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
				if !fd.AddOffsetConstraint(fdVars[i], i, d1[i]) {
					return
				}
				if !fd.AddOffsetConstraint(fdVars[i], -i+n, d2[i]) {
					return
				}
			}

			// all-different on columns and diagonals
			if !fd.AddAllDifferentRegin(fdVars) {
				return
			}
			if !fd.AddAllDifferentRegin(d1) {
				return
			}
			if !fd.AddAllDifferentRegin(d2) {
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
							if !fd.Assign(fdVars[i], val) {
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
					stream.Put(cloned)
				}
			}
		}()

		return stream
	}
}
