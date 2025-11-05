package minikanren

import (
	"context"
	"fmt"
	"sync/atomic"
)

// Nominal names are represented as atoms (e.g., NewAtom("a")).
// TieTerm encodes a binding form that binds a nominal name within body.
// Semantics: Tie(name, body) roughly corresponds to λ name . body
// This structure is used by freshness constraints and alpha-aware operations.
type TieTerm struct {
	name *Atom
	body Term
}

// Tie creates a binding form for a nominal name within a term body.
func Tie(name *Atom, body Term) *TieTerm {
	return &TieTerm{name: name, body: body}
}

// Lambda is an alias for Tie to emphasize binder semantics.
func Lambda(name *Atom, body Term) *TieTerm { return Tie(name, body) }

// String renders the tie term in a readable form.
func (t *TieTerm) String() string {
	return fmt.Sprintf("(tie %s . %s)", t.name.String(), t.body.String())
}

// Equal performs structural equality (NOT alpha-equivalence).
// Alpha-equivalence-aware equality will be provided by a separate goal/constraint.
func (t *TieTerm) Equal(other Term) bool {
	o, ok := other.(*TieTerm)
	if !ok {
		return false
	}
	return t.name.Equal(o.name) && t.body.Equal(o.body)
}

// IsVar indicates this is not a logic variable.
func (t *TieTerm) IsVar() bool { return false }

// Clone makes a deep copy of the tie term.
func (t *TieTerm) Clone() Term { return &TieTerm{name: t.name.Clone().(*Atom), body: t.body.Clone()} }

// Fresho adds a freshness constraint asserting that name is fresh for term.
// Intuition: name does not occur free in term; occurrences bound by inner Tie(name, ...) are allowed.
func Fresho(name *Atom, term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		c := NewFreshnessConstraint(name, term)
		err := store.AddConstraint(c)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(store)
			}
		}()
		return stream
	}
}

// FreshnessConstraint enforces that a nominal name does not occur free in a term.
// The constraint is local and re-evaluates when any variable inside the term binds.
//
// Note: LocalConstraintStore validates constraints on AddConstraint; if this
// freshness is already violated under current bindings, the add will be rejected
// with an error and the constraint will not be stored.
type FreshnessConstraint struct {
	id   string
	name *Atom
	term Term
}

// NewFreshnessConstraint constructs a freshness constraint a # term.
func NewFreshnessConstraint(name *Atom, term Term) *FreshnessConstraint {
	return &FreshnessConstraint{
		id:   fmt.Sprintf("fresh(%p)", term),
		name: name,
		term: term,
	}
}

// ID implements Constraint.
func (fc *FreshnessConstraint) ID() string { return fc.id }

// IsLocal implements Constraint (freshness is checked locally).
func (fc *FreshnessConstraint) IsLocal() bool { return true }

// Variables returns variables that can affect the freshness decision (all vars in term).
func (fc *FreshnessConstraint) Variables() []*Var {
	vars := make([]*Var, 0, 4)
	collectVars(fc.term, &vars)
	return vars
}

// collectVars recursively gathers all logic variables in a term.
func collectVars(t Term, acc *[]*Var) {
	switch v := t.(type) {
	case *Var:
		*acc = append(*acc, v)
	case *Pair:
		collectVars(v.Car(), acc)
		collectVars(v.Cdr(), acc)
	case *TieTerm:
		collectVars(v.body, acc)
	}
}

// Check evaluates the freshness constraint against current bindings.
func (fc *FreshnessConstraint) Check(bindings map[int64]Term) ConstraintResult {
	// Build a substitution from bindings to walk the term
	sub := NewSubstitution()
	for id, term := range bindings {
		sub = sub.Bind(&Var{id: id}, term)
	}

	walked := sub.DeepWalk(fc.term)

	// Evaluate freshness; if unknown vars remain, return pending
	found, pending := occursNominalFree(fc.name, walked, nil)
	if found {
		return ConstraintViolated
	}
	if pending {
		return ConstraintPending
	}
	return ConstraintSatisfied
}

// String implements Constraint formatting.
func (fc *FreshnessConstraint) String() string {
	return fmt.Sprintf("fresh(%s, %s)", fc.name.String(), fc.term.String())
}

// Clone implements deep copy.
func (fc *FreshnessConstraint) Clone() Constraint {
	return &FreshnessConstraint{
		id:   fc.id,
		name: fc.name.Clone().(*Atom),
		term: fc.term.Clone(),
	}
}

// occursNominalFree checks if name occurs free in term.
// Returns (occursFree, pending) where pending indicates presence of unbound variables.
func occursNominalFree(name *Atom, term Term, bound map[string]struct{}) (bool, bool) {
	// Initialize bound set lazily
	if bound == nil {
		bound = make(map[string]struct{})
	}

	// Helper to compare atoms by underlying value
	sameName := func(a *Atom) bool {
		return a.Value() == name.Value()
	}

	switch t := term.(type) {
	case *Atom:
		// If the atom matches the name and is not currently bound, it's a free occurrence
		if sameName(t) {
			if _, isBound := bound[fmt.Sprint(name.Value())]; !isBound {
				return true, false
			}
		}
		return false, false
	case *Var:
		// Unbound variable may later become the name; conservatively pending
		return false, true
	case *Pair:
		f1, p1 := occursNominalFree(name, t.Car(), bound)
		if f1 {
			return true, false
		}
		f2, p2 := occursNominalFree(name, t.Cdr(), bound)
		if f2 {
			return true, false
		}
		return false, p1 || p2
	case *TieTerm:
		// Enter binder scope; if binder equals name, mark as bound within body
		key := fmt.Sprint(name.Value())
		added := false
		if sameName(t.name) {
			if _, exists := bound[key]; !exists {
				bound[key] = struct{}{}
				added = true
			}
		}
		found, pending := occursNominalFree(name, t.body, bound)
		if added {
			delete(bound, key)
		}
		return found, pending
	default:
		return false, false
	}
}

// NomFresh generates fresh nominal name atoms with unique suffixes to avoid accidental clashes.
// If names are provided, they're used as prefixes; otherwise "n" is used.
func NomFresh(prefix string) *Atom {
	id := atomic.AddInt64(&varCounter, 1)
	return NewAtom(fmt.Sprintf("%s#%d", prefix, id))
}

// AlphaEqo adds an alpha-equivalence constraint between two terms.
// It succeeds when the terms are structurally equal modulo renaming of bound names.
func AlphaEqo(left, right Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		c := NewAlphaEqConstraint(left, right)
		err := store.AddConstraint(c)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(store)
			}
		}()
		return stream
	}
}

// AlphaEqConstraint checks alpha-equivalence between two terms (Tie-aware).
type AlphaEqConstraint struct {
	id    string
	left  Term
	right Term
}

// NewAlphaEqConstraint constructs the constraint object.
func NewAlphaEqConstraint(left, right Term) *AlphaEqConstraint {
	return &AlphaEqConstraint{
		id:    fmt.Sprintf("alphaeq(%p,%p)", left, right),
		left:  left,
		right: right,
	}
}

func (ac *AlphaEqConstraint) ID() string    { return ac.id }
func (ac *AlphaEqConstraint) IsLocal() bool { return true }
func (ac *AlphaEqConstraint) Variables() []*Var {
	vars := make([]*Var, 0, 8)
	collectVars(ac.left, &vars)
	collectVars(ac.right, &vars)
	return vars
}
func (ac *AlphaEqConstraint) String() string {
	return fmt.Sprintf("alphaEq(%s, %s)", ac.left.String(), ac.right.String())
}
func (ac *AlphaEqConstraint) Clone() Constraint {
	return &AlphaEqConstraint{id: ac.id, left: ac.left.Clone(), right: ac.right.Clone()}
}
func (ac *AlphaEqConstraint) Check(bindings map[int64]Term) ConstraintResult {
	sub := NewSubstitution()
	for id, term := range bindings {
		sub = sub.Bind(&Var{id: id}, term)
	}
	l := sub.DeepWalk(ac.left)
	r := sub.DeepWalk(ac.right)

	eq, pending := alphaEq(l, r, map[string]string{}, map[string]string{})
	if eq {
		return ConstraintSatisfied
	}
	if pending {
		return ConstraintPending
	}
	return ConstraintViolated
}

// alphaEq compares two terms modulo renaming of bound names.
// Returns (equal, pending) where pending indicates unresolved variables.
func alphaEq(a, b Term, env map[string]string, inv map[string]string) (bool, bool) {
	switch ta := a.(type) {
	case *Var:
		// Unresolved logic variable: result pending
		return false, true
	case *Atom:
		switch tb := b.(type) {
		case *Var:
			return false, true
		case *Atom:
			// If both atoms are equal by value, ok
			if ta.Value() == tb.Value() {
				return true, false
			}
			// If both look like names (strings), consult env mapping
			as, aok := ta.Value().(string)
			bs, bok := tb.Value().(string)
			if aok && bok {
				if mapped, exists := env[as]; exists {
					return mapped == bs, false
				}
				if mapped, exists := inv[bs]; exists {
					return mapped == as, false
				}
			}
			return false, false
		default:
			return false, false
		}
	case *Pair:
		tb, ok := b.(*Pair)
		if !ok {
			// If the other is var, pending; else not equal
			if _, v := b.(*Var); v {
				return false, true
			}
			return false, false
		}
		e1, p1 := alphaEq(ta.Car(), tb.Car(), env, inv)
		if !e1 {
			return false, p1
		}
		e2, p2 := alphaEq(ta.Cdr(), tb.Cdr(), env, inv)
		return e2, p1 || p2
	case *TieTerm:
		tb, ok := b.(*TieTerm)
		if !ok {
			if _, v := b.(*Var); v {
				return false, true
			}
			return false, false
		}
		// Extend env with binder mapping
		aName, bName := fmt.Sprint(ta.name.Value()), fmt.Sprint(tb.name.Value())
		// Check consistency with existing mappings
		if prev, exists := env[aName]; exists && prev != bName {
			return false, false
		}
		if prev, exists := inv[bName]; exists && prev != aName {
			return false, false
		}
		// Extend and recurse on bodies
		env2 := make(map[string]string, len(env)+1)
		for k, v := range env {
			env2[k] = v
		}
		inv2 := make(map[string]string, len(inv)+1)
		for k, v := range inv {
			inv2[k] = v
		}
		env2[aName] = bName
		inv2[bName] = aName
		return alphaEq(ta.body, tb.body, env2, inv2)
	default:
		// If b is a var, pending
		if _, v := b.(*Var); v {
			return false, true
		}
		// If exact type mismatch and not handled above, not equal
		return a.Equal(b), false
	}
}
