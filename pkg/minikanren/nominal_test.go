package minikanren

import (
	"context"
	"testing"
)

// TestFresho_BoundVsFree verifies that binding via Tie prevents a freshness violation.
func TestFresho_BoundVsFree(t *testing.T) {
	a := NewAtom("a")

	// Free occurrence should violate. LocalConstraintStore validates on add and
	// should return an error when a constraint is immediately violated.
	freeTerm := NewPair(a, Nil)
	{
		store := NewLocalConstraintStore(NewGlobalConstraintBus())
		if err := store.AddConstraint(NewFreshnessConstraint(a, freeTerm)); err == nil {
			t.Fatalf("expected error adding immediately violated freshness constraint")
		}
		// Also validate via direct Check on an empty binding map.
		c := NewFreshnessConstraint(a, freeTerm)
		res := c.Check(map[int64]Term{})
		if res != ConstraintViolated {
			t.Fatalf("expected violation for free occurrence, got %v", res)
		}
	}

	// Bound occurrence should satisfy
	boundTerm := Tie(a, a)
	{
		c := NewFreshnessConstraint(a, boundTerm)
		res := c.Check(map[int64]Term{})
		if res != ConstraintSatisfied {
			t.Fatalf("expected satisfied for bound occurrence, got %v", res)
		}
	}
}

// TestFresho_Pending ensures unbound variables yield pending result.
func TestFresho_Pending(t *testing.T) {
	a := NewAtom("a")
	x := Fresh("x")
	c := NewFreshnessConstraint(a, x)
	res := c.Check(map[int64]Term{})
	if res != ConstraintPending {
		t.Fatalf("expected pending, got %v", res)
	}
}

// TestAlphaEq_basic tests canonical alpha-equivalence cases.
func TestAlphaEq_basic(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")

	t1 := Lambda(a, a)
	t2 := Lambda(b, b)

	c := NewAlphaEqConstraint(t1, t2)
	if r := c.Check(map[int64]Term{}); r != ConstraintSatisfied {
		t.Fatalf("expected satisfied, got %v", r)
	}
}

// TestAlphaEq_nested distinguishes non-equivalent cases.
func TestAlphaEq_nested(t *testing.T) {
	a := NewAtom("a")
	b := NewAtom("b")

	t1 := Lambda(a, Lambda(b, a))
	t2 := Lambda(a, Lambda(b, b))

	c := NewAlphaEqConstraint(t1, t2)
	if r := c.Check(map[int64]Term{}); r != ConstraintViolated {
		t.Fatalf("expected violated, got %v", r)
	}
}

// TestAlphaEq_Pairs and non-name atoms are compared structurally.
func TestAlphaEq_pairsAndAtoms(t *testing.T) {
	// (1 . (2 . ()))  alpha-eq should require equal numeric atoms
	p1 := NewPair(NewAtom(1), NewPair(NewAtom(2), Nil))
	p2 := NewPair(NewAtom(1), NewPair(NewAtom(2), Nil))

	if r := NewAlphaEqConstraint(p1, p2).Check(map[int64]Term{}); r != ConstraintSatisfied {
		t.Fatalf("expected satisfied for identical pairs of ints")
	}

	p3 := NewPair(NewAtom(1), NewPair(NewAtom(3), Nil))
	if r := NewAlphaEqConstraint(p1, p3).Check(map[int64]Term{}); r != ConstraintViolated {
		t.Fatalf("expected violated for differing int atoms")
	}
}

// TestNominalPluginIntegration ensures the NominalPlugin participates in hybrid propagation.
func TestNominalPluginIntegration(t *testing.T) {
	// Create a store and add a freshness violation; the plugin must report conflict
	a := NewAtom("a")
	freeTerm := NewPair(a, Nil)

	store := NewUnifiedStore().AddConstraint(NewFreshnessConstraint(a, freeTerm))

	hs := NewHybridSolver(NewRelationalPlugin(), NewNominalPlugin())
	if _, err := hs.Propagate(store); err == nil {
		t.Fatalf("expected propagation error due to freshness violation")
	}

	// Now add a satisfied alpha-equivalence constraint and expect success
	t1 := Lambda(a, a)
	t2 := Lambda(NewAtom("b"), NewAtom("b"))
	okStore := NewUnifiedStore().AddConstraint(NewAlphaEqConstraint(t1, t2))
	if _, err := hs.Propagate(okStore); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGoals_endToEnd runs goals through the LocalConstraintStore to ensure API usability.
func TestGoals_endToEnd(t *testing.T) {
	ctx := context.Background()
	a := NewAtom("a")

	// Case 1: Fresho violated -> no results
	list := NewPair(a, Nil)
	g1 := Fresho(a, list)
	s1 := g1(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	if rs, _ := s1.Take(1); len(rs) != 0 {
		t.Fatalf("expected 0 results for violated Fresho")
	}

	// Case 2: AlphaEq satisfied -> produces the store
	t1 := Lambda(a, a)
	t2 := Lambda(NewAtom("b"), NewAtom("b"))
	g2 := AlphaEqo(t1, t2)
	s2 := g2(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	if rs, _ := s2.Take(1); len(rs) != 1 {
		t.Fatalf("expected 1 result for satisfied AlphaEqo")
	}
}
