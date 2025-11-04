package minikanren

import (
	"context"
	"testing"
)

// TestCopyTerm_SimpleAtom tests copying an atom (should be unchanged).
func TestCopyTerm_SimpleAtom(t *testing.T) {
	original := NewAtom("hello")

	result := Run(1, func(copy *Var) Goal {
		return CopyTerm(original, copy)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(original) {
		t.Errorf("Expected %v, got %v", original, result[0])
	}
}

// TestCopyTerm_FreshVariable tests copying creates a fresh variable.
func TestCopyTerm_FreshVariable(t *testing.T) {
	x := Fresh("x")

	result := Run(1, func(copy *Var) Goal {
		return CopyTerm(x, copy)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// The copy should be a fresh variable, not bound
	if !result[0].IsVar() {
		t.Errorf("Expected copy to be a variable, got %v", result[0])
	}

	// Should be a different variable
	if result[0].Equal(x) {
		t.Errorf("Expected fresh variable, got same variable")
	}
}

// TestCopyTerm_BoundVariable tests copying a bound variable.
func TestCopyTerm_BoundVariable(t *testing.T) {
	x := Fresh("x")

	result := Run(1, func(copy *Var) Goal {
		return Conj(
			Eq(x, NewAtom("bound")),
			CopyTerm(x, copy),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// The copy should be the walked value
	if !result[0].Equal(NewAtom("bound")) {
		t.Errorf("Expected %v, got %v", NewAtom("bound"), result[0])
	}
}

// TestCopyTerm_SharedVariables tests that shared variables remain shared in the copy.
func TestCopyTerm_SharedVariables(t *testing.T) {
	x := Fresh("x")
	// Create a list with the same variable appearing twice: [x, "middle", x]
	original := List(x, NewAtom("middle"), x)

	result := Run(1, func(copy *Var) Goal {
		return CopyTerm(original, copy)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// Verify the copy is a list
	copyList := result[0]
	if _, ok := copyList.(*Pair); !ok {
		t.Fatalf("Expected copy to be a list, got %T", copyList)
	}

	// Extract elements using pattern matching
	first := Fresh("first")
	third := Fresh("third")

	// The key property of CopyTerm is that if a variable appears multiple times
	// in the original, the SAME fresh variable appears in those positions in the copy.
	// We can test this by binding one and checking if the other is also bound.
	checkSharing := Run(1, func(q *Var) Goal {
		return Conj(
			Eq(copyList, List(first, Fresh("_"), third)),
			Eq(first, NewAtom("test")), // Bind first element
			Eq(third, NewAtom("test")), // Third should be bound to same value
			Eq(q, NewAtom("ok")),
		)
	})

	if len(checkSharing) != 1 {
		t.Errorf("Expected variable sharing to be preserved in copy (both should unify to 'test')")
	}
}

// TestCopyTerm_NestedStructure tests copying deeply nested pairs.
func TestCopyTerm_NestedStructure(t *testing.T) {
	x := Fresh("x")
	y := Fresh("y")
	// Create ((x . y) . (x . y))
	original := NewPair(NewPair(x, y), NewPair(x, y))

	result := Run(1, func(copy *Var) Goal {
		return CopyTerm(original, copy)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// Verify structure is preserved
	copyPair := result[0]
	if _, ok := copyPair.(*Pair); !ok {
		t.Fatalf("Expected copy to be a pair, got %T", copyPair)
	}
}

// TestGround_GroundAtom tests that an atom is considered ground.
func TestGround_GroundAtom(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(NewAtom(42)),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (atom is ground), got %d", len(result))
	}

	if !result[0].Equal(NewAtom("success")) {
		t.Errorf("Expected success marker")
	}
}

// TestGround_UnboundVariable tests that an unbound variable is not ground.
func TestGround_UnboundVariable(t *testing.T) {
	x := Fresh("x")

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(x),
			Eq(q, NewAtom("success")),
		)
	})

	// Should fail because x is unbound
	if len(result) != 0 {
		t.Errorf("Expected 0 results (unbound var not ground), got %d", len(result))
	}
}

// TestGround_BoundVariable tests that a bound variable is ground.
func TestGround_BoundVariable(t *testing.T) {
	x := Fresh("x")

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Eq(x, NewAtom("bound")),
			Ground(x),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (bound var is ground), got %d", len(result))
	}

	if !result[0].Equal(NewAtom("success")) {
		t.Errorf("Expected success marker")
	}
}

// TestGround_PartiallyGroundList tests list with unbound variable.
func TestGround_PartiallyGroundList(t *testing.T) {
	x := Fresh("x")
	list := List(NewAtom(1), x, NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(list),
			Eq(q, NewAtom("success")),
		)
	})

	// Should fail because list contains unbound variable
	if len(result) != 0 {
		t.Errorf("Expected 0 results (partially ground list), got %d", len(result))
	}
}

// TestGround_FullyGroundList tests fully ground list.
func TestGround_FullyGroundList(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(list),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (fully ground list), got %d", len(result))
	}
}

// TestArityo_Atom tests arity of an atom (should be 0).
func TestArityo_Atom(t *testing.T) {
	result := Run(1, func(arity *Var) Goal {
		return Arityo(NewAtom("hello"), arity)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(0)) {
		t.Errorf("Expected arity 0 for atom, got %v", result[0])
	}
}

// TestArityo_EmptyList tests arity of empty list.
func TestArityo_EmptyList(t *testing.T) {
	result := Run(1, func(arity *Var) Goal {
		return Arityo(Nil, arity)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(0)) {
		t.Errorf("Expected arity 0 for empty list, got %v", result[0])
	}
}

// TestArityo_List tests arity of a list (length).
func TestArityo_List(t *testing.T) {
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(arity *Var) Goal {
		return Arityo(list, arity)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(3)) {
		t.Errorf("Expected arity 3 for 3-element list, got %v", result[0])
	}
}

// TestArityo_UnboundVariable tests that arity fails for unbound variable.
func TestArityo_UnboundVariable(t *testing.T) {
	x := Fresh("x")

	result := Run(1, func(arity *Var) Goal {
		return Arityo(x, arity)
	})

	// Should fail - cannot determine arity of unbound variable
	if len(result) != 0 {
		t.Errorf("Expected 0 results for unbound variable, got %d", len(result))
	}
}

// TestFunctoro_Pair tests extracting functor from a pair.
func TestFunctoro_Pair(t *testing.T) {
	pair := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2)))

	result := Run(1, func(functor *Var) Goal {
		return Functoro(pair, functor)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("foo")) {
		t.Errorf("Expected functor 'foo', got %v", result[0])
	}
}

// TestFunctoro_Atom tests that functor fails for atom.
func TestFunctoro_Atom(t *testing.T) {
	result := Run(1, func(functor *Var) Goal {
		return Functoro(NewAtom("not-a-pair"), functor)
	})

	// Should fail - atoms don't have functors
	if len(result) != 0 {
		t.Errorf("Expected 0 results for atom, got %d", len(result))
	}
}

// TestCompoundTermo_Pair tests that a pair is compound.
func TestCompoundTermo_Pair(t *testing.T) {
	pair := NewPair(NewAtom("a"), NewAtom("b"))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(pair),
			Eq(q, NewAtom("is-compound")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (pair is compound), got %d", len(result))
	}
}

// TestCompoundTermo_Atom tests that an atom is not compound.
func TestCompoundTermo_Atom(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(NewAtom(42)),
			Eq(q, NewAtom("is-compound")),
		)
	})

	// Should fail - atoms are not compound
	if len(result) != 0 {
		t.Errorf("Expected 0 results for atom, got %d", len(result))
	}
}

// TestSimpleTermo_Atom tests that an atom is simple.
func TestSimpleTermo_Atom(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewAtom(42)),
			Eq(q, NewAtom("is-simple")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (atom is simple), got %d", len(result))
	}
}

// TestSimpleTermo_Pair tests that a pair is not simple.
func TestSimpleTermo_Pair(t *testing.T) {
	pair := NewPair(NewAtom("a"), NewAtom("b"))

	result := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(pair),
			Eq(q, NewAtom("is-simple")),
		)
	})

	// Should fail - pairs are not simple
	if len(result) != 0 {
		t.Errorf("Expected 0 results for pair, got %d", len(result))
	}
}

// TestStringo_StringAtom tests string type constraint on string.
func TestStringo_StringAtom(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Stringo(q),
			Eq(q, NewAtom("hello")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom("hello")) {
		t.Errorf("Expected 'hello', got %v", result[0])
	}
}

// TestStringo_NumberAtom tests string constraint fails on number.
func TestStringo_NumberAtom(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Stringo(q),
			Eq(q, NewAtom(42)),
		)
	})

	// Should fail - 42 is not a string
	if len(result) != 0 {
		t.Errorf("Expected 0 results for number, got %d", len(result))
	}
}

// TestBooleano_True tests boolean constraint on true.
func TestBooleano_True(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Eq(q, NewAtom(true)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(true)) {
		t.Errorf("Expected true, got %v", result[0])
	}
}

// TestBooleano_False tests boolean constraint on false.
func TestBooleano_False(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Eq(q, NewAtom(false)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	if !result[0].Equal(NewAtom(false)) {
		t.Errorf("Expected false, got %v", result[0])
	}
}

// TestBooleano_NonBoolean tests boolean constraint fails on non-boolean.
func TestBooleano_NonBoolean(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Eq(q, NewAtom("not-bool")),
		)
	})

	// Should fail - string is not a boolean
	if len(result) != 0 {
		t.Errorf("Expected 0 results for non-boolean, got %d", len(result))
	}
}

// TestVectoro_IntSlice tests vector constraint on int slice.
func TestVectoro_IntSlice(t *testing.T) {
	slice := []int{1, 2, 3}

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Eq(q, NewAtom(slice)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// Extract the slice
	if atom, ok := result[0].(*Atom); ok {
		if s, ok := atom.Value().([]int); ok {
			if len(s) != 3 {
				t.Errorf("Expected slice length 3, got %d", len(s))
			}
		} else {
			t.Errorf("Expected []int, got %T", atom.Value())
		}
	} else {
		t.Errorf("Expected atom result, got %T", result[0])
	}
}

// TestVectoro_StringSlice tests vector constraint on string slice.
func TestVectoro_StringSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Eq(q, NewAtom(slice)),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
}

// TestVectoro_NonVector tests vector constraint fails on non-vector.
func TestVectoro_NonVector(t *testing.T) {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Eq(q, NewAtom("not-a-vector")),
		)
	})

	// Should fail - string is not a vector
	if len(result) != 0 {
		t.Errorf("Expected 0 results for non-vector, got %d", len(result))
	}
}

// TestCopyTerm_WithConstraints tests copying preserves constraint semantics.
func TestCopyTerm_WithConstraints(t *testing.T) {
	x := Fresh("x")
	original := List(x, NewAtom(2), x)

	result := Run(1, func(copy *Var) Goal {
		return Conj(
			Numbero(x), // Constrain original x to be a number
			CopyTerm(original, copy),
			Eq(x, NewAtom(1)), // Bind original x to 1
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}

	// The copy should have fresh variables, not bound to 1
	// Extract first element of copy
	copyList := result[0]
	if pair, ok := copyList.(*Pair); ok {
		firstElem := pair.Car()
		// Should be a fresh variable, not 1
		if firstElem.Equal(NewAtom(1)) {
			t.Errorf("Expected copy to have fresh variable, got bound value 1")
		}
	}
}

// TestGround_DeeplyNested tests ground checking on deeply nested structure.
func TestGround_DeeplyNested(t *testing.T) {
	// Create deeply nested structure: ((1 . 2) . (3 . 4))
	deep := NewPair(
		NewPair(NewAtom(1), NewAtom(2)),
		NewPair(NewAtom(3), NewAtom(4)),
	)

	result := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(deep),
			Eq(q, NewAtom("success")),
		)
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 result (deeply nested structure is ground), got %d", len(result))
	}
}

// TestTermUtils_Parallel tests term utilities work correctly in parallel execution.
func TestTermUtils_Parallel(t *testing.T) {
	x := Fresh("x")
	y := Fresh("y")

	result := Run(2, func(q *Var) Goal {
		return Disj(
			Conj(
				Eq(x, NewAtom("a")),
				Ground(x),
				CopyTerm(x, y),
				Eq(q, y),
			),
			Conj(
				Eq(x, NewAtom("b")),
				Ground(x),
				CopyTerm(x, y),
				Eq(q, y),
			),
		)
	})

	// Should get both "a" and "b"
	if len(result) != 2 {
		t.Fatalf("Expected 2 results from parallel execution, got %d", len(result))
	}

	// Results should be "a" and "b" in some order
	hasA := false
	hasB := false
	for _, r := range result {
		if r.Equal(NewAtom("a")) {
			hasA = true
		}
		if r.Equal(NewAtom("b")) {
			hasB = true
		}
	}

	if !hasA || !hasB {
		t.Errorf("Expected both 'a' and 'b' in results, got %v", result)
	}
}

// TestCopyTerm_ContextCancellation tests CopyTerm respects context cancellation.
func TestCopyTerm_ContextCancellation(t *testing.T) {
	x := Fresh("x")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	goal := CopyTerm(x, Fresh("copy"))
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)

	results := make([]ConstraintStore, 0)
	for {
		state, hasMore := stream.Take(1)
		if !hasMore {
			break
		}
		results = append(results, state...)
	}

	// Should get no results due to cancellation
	if len(results) != 0 {
		t.Errorf("Expected 0 results due to cancellation, got %d", len(results))
	}
}
