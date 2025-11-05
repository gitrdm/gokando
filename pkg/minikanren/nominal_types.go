package minikanren

import (
	"context"
	"fmt"
)

// ArrType constructs an arrow type term (t1 -> t2) encoded as
// Pair(Atom("->"), Pair(t1, Pair(t2, Nil))).
func ArrType(t1, t2 Term) Term {
	return NewPair(NewAtom("->"), NewPair(t1, NewPair(t2, Nil)))
}

// isArrType attempts to decompose an arrow type; returns (t1, t2, ok).
func isArrType(t Term) (Term, Term, bool) {
	p, ok := t.(*Pair)
	if !ok {
		return nil, nil, false
	}
	headAtom, ok := p.Car().(*Atom)
	if !ok || fmt.Sprint(headAtom.Value()) != "->" {
		return nil, nil, false
	}
	args, ok := p.Cdr().(*Pair)
	if !ok {
		return nil, nil, false
	}
	t1 := args.Car()
	rest, ok := args.Cdr().(*Pair)
	if !ok {
		return nil, nil, false
	}
	t2 := rest.Car()
	return t1, t2, true
}

// EnvExtend returns a new env mapping name->typ as Pair(Pair(name, typ), env).
func EnvExtend(env Term, name *Atom, typ Term) Term {
	return NewPair(NewPair(name, typ), env)
}

// TypeChecko checks that term has type "typ" under environment env.
// Environment is an association list of (name . type) pairs: ((x . T1) (y . T2) ...)
// Supported term forms: variables (Atoms), application (Pair(fun,arg) via App), and lambda (Tie).
// Typing rules (simply-typed λ-calculus):
//   - Var: type from env
//   - App: if fun : A->B and arg : A then (fun arg) : B
//   - Lam: if typ is of the form A->B, then under env[x:=A] body : B
//
// This is a checker (expects typ shape for lambdas); with logic variables inside typ, it can infer A/B.
func TypeChecko(term Term, env Term, typ Term) Goal {
	return func(ctx context.Context, store ConstraintStore) *Stream {
		sub := store.GetSubstitution()
		t := sub.DeepWalk(term)
		e := sub.DeepWalk(env)
		ty := sub.DeepWalk(typ)

		// Deterministic check with unification at the end
		inferred, ok := typeCheckDet(t, e, ty)
		if !ok {
			s := NewStream()
			s.Close()
			return s
		}
		return Eq(ty, inferred)(ctx, store)
	}
}

// typeCheckDet attempts to infer the type of term under env, possibly guided by expected typ.
// Returns (inferredType, ok). ok=false indicates pending due to insufficient structure.
func typeCheckDet(term Term, env Term, typ Term) (Term, bool) {
	switch t := term.(type) {
	case *Var:
		return nil, false
	case *Atom:
		// Look up in env
		envTy, ok := envLookupDet(env, t)
		if !ok {
			return nil, false
		}
		// If an expected type is provided and ground, enforce equality
		switch typ.(type) {
		case *Atom, *Pair:
			if typ != nil && !envTy.Equal(typ) {
				return nil, false
			}
		case *Var:
			// expected is a logic var; accept envTy and let outer unification relate them
		}
		return envTy, true
	case *TieTerm:
		// Lambda: expected type must be an arrow A->B
		t1, t2, ok := isArrType(typ)
		if !ok {
			return nil, false
		}
		// Extend environment with binder : t1
		env2 := EnvExtend(env, t.name, t1)
		bodyTy, ok := typeCheckDet(t.body, env2, t2)
		if !ok {
			return nil, false
		}
		// Return Arr(t1, bodyTy) (unify may refine t2 later)
		return ArrType(t1, bodyTy), true
	case *Pair:
		// Application: fun must have Arr(argTy, resTy)
		funTy, ok := typeCheckDet(t.Car(), env, nil)
		if !ok {
			return nil, false
		}
		argTy, resTy, ok := isArrType(funTy)
		if !ok {
			return nil, false
		}
		// Check argument type compatibility
		if _, ok := typeCheckDet(t.Cdr(), env, argTy); !ok {
			return nil, false
		}
		// Return result type
		return resTy, true
	default:
		return nil, false
	}
}

// envLookupDet finds the type bound to name in env (alist). Returns (type, ok).
func envLookupDet(env Term, name *Atom) (Term, bool) {
	switch e := env.(type) {
	case *Var:
		return nil, false
	case *Pair:
		pair, ok := e.Car().(*Pair)
		if ok {
			if atom, ok := pair.Car().(*Atom); ok && atom.Value() == name.Value() {
				return pair.Cdr(), true
			}
		}
		return envLookupDet(e.Cdr(), name)
	case *Atom:
		// Nil or unexpected atom → not found
		return nil, false
	default:
		return nil, false
	}
}

// NewVarOr returns typ if not nil, otherwise a fresh logic var. Helper to guide inference.
func NewVarOr(typ Term) Term {
	if typ == nil {
		return Fresh("T")
	}
	return typ
}
