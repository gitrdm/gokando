package minikanren

import (
	"context"
)

// Substo relates out to the result of capture-avoiding substitution of all
// free occurrences of the nominal name `name` in `term` with `replacement`.
//
// Contract (deterministic core, relational wrapper):
//   - Inputs: term (Term possibly containing Tie binders), name (*Atom), replacement (Term)
//   - Output: out (Term) such that out ≡ term[name := replacement] with capture avoidance
//   - If the decision depends on unresolved logic variables in term/replacement,
//     the goal yields no solutions until they become instantiated enough.
//
// Binder cases (λ-calculus intuition with Tie(name, body)):
//   - If the binder equals `name`, occurrences are bound; substitution does not enter the body.
//   - Else, if binder is fresh for `replacement` (no free occurrence inside replacement),
//     substitute in the body under the same binder.
//   - Else, pick a fresh nominal name a' (NomFresh("n")), alpha-rename the binder in the
//     body to a' (avoiding inner shadowing), then substitute under the renamed binder.
func Substo(term Term, name *Atom, replacement Term, out Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		// Evaluate deterministically under current bindings; if pending, don't emit.
		sub := store.GetSubstitution()
		walkedTerm := sub.DeepWalk(term)
		walkedRepl := sub.DeepWalk(replacement)

		res, ok := substoDet(walkedTerm, name, walkedRepl)
		if !ok {
			// Not enough information (pending); fail for now. Re-running later will succeed.
			stream := NewStream()
			stream.Close()
			return stream
		}

		return Eq(out, res)(ctx, store)
	}
}

// substoDet performs capture-avoiding substitution deterministically on a ground-enough term.
// Returns (result, true) if computed; (nil, false) if pending due to unresolved vars.
func substoDet(term Term, name *Atom, replacement Term) (Term, bool) {
	switch t := term.(type) {
	case *Var:
		// Structure unknown → pending
		return nil, false
	case *Atom:
		// If atom equals the target nominal name and is free (no binder here),
		// replace with the replacement; otherwise keep as-is.
		if t.Value() == name.Value() {
			return replacement.Clone(), true
		}
		return t.Clone(), true
	case *Pair:
		carRes, ok := substoDet(t.Car(), name, replacement)
		if !ok {
			return nil, false
		}
		cdrRes, ok := substoDet(t.Cdr(), name, replacement)
		if !ok {
			return nil, false
		}
		return NewPair(carRes, cdrRes), true
	case *TieTerm:
		// If binder equals the name, do not enter body (name is bound)
		if t.name.Value() == name.Value() {
			// No substitution under this binder
			return &TieTerm{name: t.name.Clone().(*Atom), body: t.body.Clone()}, true
		}

		// If binder is fresh for replacement, we can substitute under the same binder
		occurs, pending := occursNominalFree(t.name, replacement, nil)
		if pending {
			return nil, false
		}
		if !occurs {
			bodyRes, ok := substoDet(t.body, name, replacement)
			if !ok {
				return nil, false
			}
			return &TieTerm{name: t.name.Clone().(*Atom), body: bodyRes}, true
		}

		// Otherwise, alpha-rename the binder to a fresh name to avoid capture,
		// then substitute under the renamed binder.
		fresh := NomFresh("n")
		// Ensure we didn't accidentally pick the same name (defensive)
		for fresh.Value() == t.name.Value() || fresh.Value() == name.Value() {
			fresh = NomFresh("n")
		}
		renamedBody := renameBound(t.body, t.name, fresh)
		bodyRes, ok := substoDet(renamedBody, name, replacement)
		if !ok {
			return nil, false
		}
		return &TieTerm{name: fresh, body: bodyRes}, true
	default:
		// Unknown compound term kinds default to identity
		return t.Clone(), true
	}
}

// renameBound renames bound occurrences of `oldName` to `newName` within `term`.
// It assumes it's called on the body of a Tie(oldName, body). It does not descend
// into inner Tie that also bind oldName (to respect shadowing). For other inner
// binders, it recurses normally.
func renameBound(term Term, oldName, newName *Atom) Term {
	switch t := term.(type) {
	case *Var:
		return t.Clone()
	case *Atom:
		if t.Value() == oldName.Value() {
			return newName.Clone()
		}
		return t.Clone()
	case *Pair:
		return NewPair(renameBound(t.Car(), oldName, newName), renameBound(t.Cdr(), oldName, newName))
	case *TieTerm:
		// If this inner binder shadows oldName, do not rename inside
		if t.name.Value() == oldName.Value() {
			return &TieTerm{name: t.name.Clone().(*Atom), body: t.body.Clone()}
		}
		return &TieTerm{name: t.name.Clone().(*Atom), body: renameBound(t.body, oldName, newName)}
	default:
		return t.Clone()
	}
}
