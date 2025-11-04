package minikanren

import (
	"context"
)

// CopyTerm creates a goal that unifies copy with a structurally identical version
// of original, but with all variables replaced by fresh variables. This is essential
// for meta-programming tasks and implementing certain tabling patterns.
//
// The copy preserves the structure of the original term:
//   - Atoms are copied as-is (they're immutable)
//   - Variables are replaced with fresh variables
//   - Pairs are recursively copied
//
// Example:
//
//	x := Fresh("x")
//	original := List(x, NewAtom("hello"), x)  // [x, "hello", x]
//	result := Run(1, func(copy *Var) Goal {
//	    return CopyTerm(original, copy)
//	})
//	// Result will be a list with TWO fresh variables (preserving sharing):
//	// [_fresh1, "hello", _fresh1]
func CopyTerm(original, copy Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the original term to its current value
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(original)

		// Create a mapping from old variables to fresh variables
		// This ensures that if the same variable appears multiple times,
		// it's replaced with the same fresh variable each time
		varMap := make(map[int64]*Var)

		// Recursively copy the term
		copied := copyTermRecursive(walked, varMap)

		// Unify the copy with the copied term
		return Eq(copy, copied)(ctx, store)
	}
}

// copyTermRecursive performs the actual copying with variable tracking.
// The varMap ensures that shared variables in the original remain shared
// in the copy (with fresh variables).
func copyTermRecursive(term Term, varMap map[int64]*Var) Term {
	switch t := term.(type) {
	case *Var:
		// Check if we've already created a fresh variable for this one
		if fresh, exists := varMap[t.ID()]; exists {
			return fresh
		}

		// Create a new fresh variable and remember the mapping
		fresh := Fresh(t.name)
		varMap[t.ID()] = fresh
		return fresh

	case *Atom:
		// Atoms are immutable, so we can return them as-is
		return t

	case *Pair:
		// Recursively copy both car and cdr
		newCar := copyTermRecursive(t.Car(), varMap)
		newCdr := copyTermRecursive(t.Cdr(), varMap)
		return NewPair(newCar, newCdr)

	default:
		// For any other term type, clone it
		return term.Clone()
	}
}

// Ground creates a goal that succeeds only if the given term is fully ground
// (contains no unbound variables). This is useful for validation and ensuring
// that a term is fully instantiated before performing certain operations.
//
// A term is considered ground if:
//   - It's an atom (atoms have no variables)
//   - It's a variable that's bound to a ground term
//   - It's a pair where both car and cdr are ground
//
// Example:
//
//	x := Fresh("x")
//	result := Run(1, func(q *Var) Goal {
//	    return Conj(
//	        Eq(x, NewAtom("hello")),
//	        Ground(x),  // Succeeds because x is now bound
//	        Eq(q, NewAtom("success")),
//	    )
//	})
//	// Result: ["success"]
func Ground(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the term to resolve variable bindings
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		// Check if the walked term is ground
		isGround := isTermGround(walked)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if isGround {
				stream.Put(store)
			}
		}()
		return stream
	}
}

// isTermGround recursively checks if a term contains any unbound variables.
func isTermGround(term Term) bool {
	switch t := term.(type) {
	case *Var:
		// An unbound variable means the term is not ground
		return false

	case *Atom:
		// Atoms are always ground
		return true

	case *Pair:
		// A pair is ground if both car and cdr are ground
		return isTermGround(t.Car()) && isTermGround(t.Cdr())

	default:
		// Conservatively assume other types are ground
		// (they typically don't contain unbound variables)
		return true
	}
}

// Arityo creates a goal that relates a term to its arity.
// For pairs/lists, the arity is the length of the list.
// For atoms, the arity is 0.
// For variables, the goal fails (cannot determine arity of unbound variable).
//
// This is useful for meta-programming and validating term structure.
//
// Example:
//
//	pair := NewPair(NewAtom("a"), NewPair(NewAtom("b"), Nil))
//	result := Run(1, func(arity *Var) Goal {
//	    return Arityo(pair, arity)
//	})
//	// Result: [2]
func Arityo(term, arity Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the term to resolve variable bindings
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		stream := NewStream()

		// Compute arity based on term type
		switch walked.(type) {
		case *Var:
			// Cannot determine arity of unbound variable
			go stream.Close()
			return stream

		case *Atom:
			// Atoms have arity 0
			go func() {
				defer stream.Close()
				stream.Put(store)
			}()
			return Eq(arity, NewAtom(0))(ctx, store)

		case *Pair:
			// Compute list length as arity
			return LengthoInt(walked, arity)(ctx, store)

		default:
			// Unknown term type, arity 0
			go func() {
				defer stream.Close()
				stream.Put(store)
			}()
			return Eq(arity, NewAtom(0))(ctx, store)
		}
	}
}

// Functoro creates a goal that relates a pair to its "functor" (the car).
// This is useful for working with compound terms in Prolog-like patterns.
//
// For a pair (a . b), the functor is a.
// For atoms and variables, the goal fails.
//
// Example:
//
//	pair := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2)))
//	result := Run(1, func(functor *Var) Goal {
//	    return Functoro(pair, functor)
//	})
//	// Result: ["foo"]
func Functoro(term, functor Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the term to resolve variable bindings
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		stream := NewStream()

		switch t := walked.(type) {
		case *Pair:
			// The functor is the car of the pair
			return Eq(functor, t.Car())(ctx, store)

		default:
			// Not a pair, fail
			go stream.Close()
			return stream
		}
	}
}

// CompoundTermo creates a goal that succeeds only if the term is a compound
// term (a pair). This is useful for validating term structure before attempting
// to decompose it.
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    pair := NewPair(NewAtom("a"), NewAtom("b"))
//	    return Conj(
//	        CompoundTermo(pair),
//	        Eq(q, NewAtom("is-compound")),
//	    )
//	})
//	// Result: ["is-compound"]
func CompoundTermo(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the term to resolve variable bindings
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		stream := NewStream()

		switch walked.(type) {
		case *Pair:
			// It's a compound term
			go func() {
				defer stream.Close()
				stream.Put(store)
			}()

		default:
			// Not a compound term
			go stream.Close()
		}

		return stream
	}
}

// SimpleTermo creates a goal that succeeds only if the term is simple
// (an atom or a fully ground term with no compound structure).
//
// Example:
//
//	result := Run(1, func(q *Var) Goal {
//	    return Conj(
//	        SimpleTermo(NewAtom(42)),
//	        Eq(q, NewAtom("is-simple")),
//	    )
//	})
//	// Result: ["is-simple"]
func SimpleTermo(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			stream := NewStream()
			stream.Close()
			return stream
		default:
		}

		// Walk the term to resolve variable bindings
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		stream := NewStream()

		switch walked.(type) {
		case *Atom:
			// Atoms are simple
			go func() {
				defer stream.Close()
				stream.Put(store)
			}()

		case *Var:
			// Unbound variables are not considered simple
			// (they could unify with anything)
			go stream.Close()

		default:
			// Pairs and other structures are not simple
			go stream.Close()
		}

		return stream
	}
}
