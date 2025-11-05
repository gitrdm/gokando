package minikanren

import "testing"

func TestTypeCheck_varAndApp(t *testing.T) {
	// Env: b : Int, id : Int->Int
	b := NewAtom("b")
	intT := NewAtom("Int")
	id := NewAtom("id")
	env := EnvExtend(EnvExtend(Nil, b, intT), id, ArrType(intT, intT))

	// (id b) : Int
	term := App(id, b)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, env, intT) })
	if len(results) != 1 {
		t.Fatalf("expected type check success, got %d results", len(results))
	}
}

func TestTypeCheck_lambdaExpectedArrow(t *testing.T) {
	// λa. a : T->T
	a := NewAtom("a")
	T := Fresh("T")
	term := Lambda(a, a)
	typ := ArrType(T, T)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, Nil, typ) })
	if len(results) != 1 {
		t.Fatalf("expected success, got %d results", len(results))
	}
}

func TestTypeCheck_mismatch(t *testing.T) {
	// id : Int->Int, applying to Bool should fail
	id := NewAtom("id")
	intT := NewAtom("Int")
	boolT := NewAtom("Bool")
	x := NewAtom("x")
	env := EnvExtend(EnvExtend(Nil, id, ArrType(intT, intT)), x, boolT)
	term := App(id, x)
	results := Run(1, func(q *Var) Goal { return TypeChecko(term, env, intT) })
	if len(results) != 0 {
		t.Fatalf("expected failure (mismatch), got %d results", len(results))
	}
}
