package minikanren

import "testing"

func TestSubsto_atom(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")

	out := Fresh("out")
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(a, a, b, out),
			Eq(q, out),
		)
	})
	if len(results) != 1 || !results[0].Equal(b) {
		t.Fatalf("expected [b], got %v", results)
	}
}

func TestSubsto_pair(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	c := NewAtom("c")
	term := NewPair(a, NewPair(c, Nil))

	out := Fresh("out")
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, b, out),
			Eq(q, out),
		)
	})
	expected := NewPair(b, NewPair(c, Nil))
	if len(results) != 1 || !results[0].Equal(expected) {
		t.Fatalf("expected [%s], got %v", expected.String(), results)
	}
}

func TestSubsto_binderBlocks(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	term := Lambda(a, a) // λa.a

	out := Fresh("out")
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, b, out),
			Eq(q, out),
		)
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Equal(term) {
		t.Fatalf("expected [%s], got %v", term.String(), results)
	}
}

func TestSubsto_avoidCapture(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	term := Lambda(b, a) // λb.a

	out := Fresh("out")
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, b, out),
			Eq(q, out),
		)
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Expect λb'. b where b' != b and body is b (free)
	tie, ok := results[0].(*TieTerm)
	if !ok {
		t.Fatalf("expected TieTerm result, got %T (%v)", results[0], results[0])
	}
	if tie.name.Equal(b) {
		t.Fatalf("binder should have been alpha-renamed to avoid capture; got binder %v", tie.name)
	}
	if !tie.body.Equal(b) {
		t.Fatalf("expected body to be b after substitution, got %v", tie.body)
	}
}

func TestSubsto_pendingWhenReplacementUnknown(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")
	x := Fresh("x") // replacement contains unknown -> may be pending
	term := Lambda(b, a)

	out := Fresh("out")
	// Under current semantics, Substo will not emit until enough info; expect 0 results
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Substo(term, a, x, out),
			Eq(q, out),
		)
	})
	if len(results) != 0 {
		t.Fatalf("expected 0 results (pending), got %d: %v", len(results), results)
	}
}
