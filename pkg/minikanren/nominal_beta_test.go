package minikanren

import "testing"

func TestBetaReduce_basic(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	// ((λa. a) b) → b
	term := App(Lambda(a, a), b)

	results := Run(1, func(q *Var) Goal { return BetaReduceo(term, q) })
	if len(results) != 1 || !results[0].Equal(b) {
		t.Fatalf("expected [b], got %v", results)
	}
}

func TestBetaReduce_avoidCapture(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	// ((λa. λb. a) b) → λb'. b (binder renamed to avoid capture)
	term := App(Lambda(a, Lambda(b, a)), b)
	results := Run(1, func(q *Var) Goal { return BetaReduceo(term, q) })
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	tie, ok := results[0].(*TieTerm)
	if !ok {
		t.Fatalf("expected TieTerm, got %T (%v)", results[0], results[0])
	}
	if tie.name.Equal(b) {
		t.Fatalf("binder should be alpha-renamed; got %v", tie.name)
	}
	if !tie.body.Equal(b) {
		t.Fatalf("expected body b, got %v", tie.body)
	}
}

func TestBetaNormalize_twoSteps(t *testing.T) {
	a := NewAtom("a")
	x := NewAtom("x")
	y := NewAtom("y")
	// ((λa. λx. a) y) → λx. y (one step); already normal (second arg absent)
	term := App(Lambda(a, Lambda(x, a)), y)
	results := Run(1, func(q *Var) Goal { return BetaNormalizeo(term, q) })
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	expected := Lambda(x, y)
	if !results[0].Equal(expected) {
		t.Fatalf("expected [%s], got %v", expected.String(), results)
	}
}

func TestBetaReduce_noRedex(t *testing.T) {
	a := NewAtom("a")
	// a is not reducible
	results := Run(1, func(q *Var) Goal { return BetaReduceo(a, q) })
	if len(results) != 0 {
		t.Fatalf("expected 0 results (no redex), got %d: %v", len(results), results)
	}
}

func TestFreeNames_basic(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	c := NewAtom("c")
	term := Lambda(a, App(a, b)) // λa. (a b) — free(b), bound(a)

	results := Run(1, func(q *Var) Goal { return FreeNameso(term, q) })
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Expect list (b)
	expected := NewPair(b, Nil)
	if !results[0].Equal(expected) {
		t.Fatalf("expected [%s], got %v", expected.String(), results)
	}

	// Add another free name c in nested body
	term2 := Lambda(a, App(App(a, b), c))
	results2 := Run(1, func(q *Var) Goal { return FreeNameso(term2, q) })
	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}
	// Sorted list (b c)
	expected2 := NewPair(b, NewPair(c, Nil))
	if !results2[0].Equal(expected2) {
		t.Fatalf("expected [%s], got %v", expected2.String(), results2)
	}
}
