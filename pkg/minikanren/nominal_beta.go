package minikanren

import (
	"context"
)

// App constructs an application term using Pair(fun, arg).
// In this library, lambda application is represented as a Pair where Car is the
// function and Cdr is the argument. This aligns with s-expression conventions
// and interoperates with existing Pair-based traversal.
func App(fun, arg Term) *Pair { return NewPair(fun, arg) }

// BetaReduceo relates out to the result of a single leftmost-outermost
// beta-reduction step performed on term. If term contains no beta-redex, the
// goal fails (produces no solutions). Reduction is capture-avoiding and uses
// Substo to substitute the argument into the lambda body.
func BetaReduceo(term Term, out Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		res, ok, changed := betaReduceDet(walked)
		if !ok || !changed {
			s := NewStream()
			s.Close()
			return s
		}
		return Eq(out, res)(ctx, store)
	}
}

// BetaNormalizeo relates out to the normal form obtained by repeatedly
// applying leftmost-outermost beta-reduction. If any decision depends on
// unresolved logic variables, the goal yields no solution until enough
// information is available.
func BetaNormalizeo(term Term, out Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		nf, ok := betaNormalizeDet(walked)
		if !ok {
			s := NewStream()
			s.Close()
			return s
		}
		return Eq(out, nf)(ctx, store)
	}
}

// betaReduceDet performs one leftmost-outermost beta-reduction step.
// Returns: (result, ok, changed)
//
//	ok=false means pending (insufficient structure information);
//	changed=false means no redex was found.
func betaReduceDet(term Term) (Term, bool, bool) {
	switch t := term.(type) {
	case *Var:
		// Unknown structure
		return nil, false, false
	case *Atom:
		return t.Clone(), true, false
	case *TieTerm:
		// Try to reduce inside body
		body, ok, changed := betaReduceDet(t.body)
		if !ok {
			return nil, false, false
		}
		if !changed {
			return &TieTerm{name: t.name.Clone().(*Atom), body: t.body.Clone()}, true, false
		}
		return &TieTerm{name: t.name.Clone().(*Atom), body: body}, true, true
	case *Pair:
		// Leftmost-outermost: if function reduces, do that; otherwise if application
		// is (Lambda x. body) arg, perform substitution; else try reducing arg.
		// 1) Try to reduce function part first
		if funTie, isTie := t.Car().(*TieTerm); isTie {
			// Application of a lambda: perform capture-avoiding substitution
			// Compute substitution deterministically; if pending, return ok=false.
			arg := t.Cdr()
			res, ok := substoDet(funTie.body, funTie.name, arg)
			if !ok {
				return nil, false, false
			}
			return res, true, true
		}

		// If function is reducible (not a tie), reduce it
		funRed, ok, changed := betaReduceDet(t.Car())
		if !ok {
			return nil, false, false
		}
		if changed {
			return NewPair(funRed, t.Cdr().Clone()), true, true
		}

		// Otherwise, try reducing the argument
		argRed, ok, changed := betaReduceDet(t.Cdr())
		if !ok {
			return nil, false, false
		}
		if changed {
			return NewPair(t.Car().Clone(), argRed), true, true
		}
		// No change
		return NewPair(t.Car().Clone(), t.Cdr().Clone()), true, false
	default:
		// Unknown compound: clone as-is
		return term.Clone(), true, false
	}
}

// betaNormalizeDet reduces a term to normal form by repeatedly applying
// leftmost-outermost beta-reduction until no change occurs.
// Returns (normalForm, ok) where ok=false indicates pending due to unknown vars.
func betaNormalizeDet(term Term) (Term, bool) {
	current := term.Clone()
	for {
		next, ok, changed := betaReduceDet(current)
		if !ok {
			return nil, false
		}
		if !changed {
			return current, true
		}
		current = next
	}
}

// FreeNameso relates out to a list (proper list using Pair/Nil) of nominal
// Atoms that occur free in term. The list is sorted lexicographically by the
// Atom's string value for determinism and ease of testing.
func FreeNameso(term Term, out Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		sub := store.GetSubstitution()
		walked := sub.DeepWalk(term)

		names, ok := freeNamesDet(walked)
		if !ok {
			s := NewStream()
			s.Close()
			return s
		}
		// Build a proper list from the sorted names
		list := Term(Nil)
		for i := len(names) - 1; i >= 0; i-- {
			list = NewPair(names[i], list)
		}
		return Eq(out, list)(ctx, store)
	}
}

// freeNamesDet computes the set of free nominal names in term.
// Returns a sorted slice of *Atom and ok=false if pending due to unknown vars.
func freeNamesDet(term Term) ([]*Atom, bool) {
	// Gather into a set keyed by string value for structural equality
	set := map[string]*Atom{}
	ok := freeNamesCollect(term, map[string]struct{}{}, set)
	if !ok {
		return nil, false
	}
	// Extract and sort for determinism
	arr := make([]*Atom, 0, len(set))
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		arr = append(arr, set[k].Clone().(*Atom))
	}
	return arr, true
}

// freeNamesCollect traverses term collecting free names into set; returns false if pending.
func freeNamesCollect(term Term, bound map[string]struct{}, set map[string]*Atom) bool {
	switch t := term.(type) {
	case *Var:
		return false
	case *Atom:
		key := asString(t)
		if key != "" {
			if _, b := bound[key]; !b {
				if _, exists := set[key]; !exists {
					set[key] = t
				}
			}
		}
		return true
	case *Pair:
		if !freeNamesCollect(t.Car(), bound, set) {
			return false
		}
		return freeNamesCollect(t.Cdr(), bound, set)
	case *TieTerm:
		key := asString(t.name)
		added := false
		if key != "" {
			if _, exists := bound[key]; !exists {
				bound[key] = struct{}{}
				added = true
			}
		}
		ok := freeNamesCollect(t.body, bound, set)
		if added {
			delete(bound, key)
		}
		return ok
	default:
		return true
	}
}

// asString returns the string value of an Atom if it is string-based; otherwise "".
func asString(a *Atom) string {
	if s, ok := a.Value().(string); ok {
		return s
	}
	return ""
}

// sortStrings sorts a slice of strings in-place (local tiny helper to avoid importing sort everywhere).
func sortStrings(a []string) {
	// Simple insertion sort (small n expected)
	for i := 1; i < len(a); i++ {
		j := i
		for j > 0 && a[j-1] > a[j] {
			a[j-1], a[j] = a[j], a[j-1]
			j--
		}
	}
}
