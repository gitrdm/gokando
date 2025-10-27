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
			sols, err := fd.Solve(0)
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
