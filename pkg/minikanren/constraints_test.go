package minikanren

import (
	"context"
	"fmt"
	"testing"
)

// TestNeq tests the disequality constraint.
func TestNeq(t *testing.T) {
	t.Run("Neq with different atoms", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Neq(q, NewAtom("forbidden")),
				Eq(q, NewAtom("allowed")),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("allowed")) {
			t.Error("Expected 'allowed', got", results[0])
		}
	})

	t.Run("Neq constraint violation", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Eq(q, NewAtom("forbidden")),
				Neq(q, NewAtom("forbidden")),
			)
		})

		if len(results) != 0 {
			t.Error("Constraint violation should return no results")
		}
	})
}

// TestAbsento tests the absence constraint.
func TestAbsento(t *testing.T) {
	t.Run("Absento with valid structure", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Absento(NewAtom("bad"), q),
				Eq(q, List(NewAtom("good"), NewAtom("ok"))),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("Absento constraint violation", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Eq(q, List(NewAtom("good"), NewAtom("bad"))),
				Absento(NewAtom("bad"), q),
			)
		})

		if len(results) != 0 {
			t.Error("Absence constraint violation should return no results")
		}
	})
}

// TestSymbolo tests the symbol type constraint.
func TestSymbolo(t *testing.T) {
	t.Run("Symbolo with string", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Symbolo(q),
				Eq(q, NewAtom("symbol")),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("symbol")) {
			t.Error("Expected 'symbol', got", results[0])
		}
	})

	t.Run("Symbolo with number fails", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Eq(q, NewAtom(42)),
				Symbolo(q),
			)
		})

		if len(results) != 0 {
			t.Error("Symbol constraint with number should fail")
		}
	})
}

// TestNumbero tests the number type constraint.
func TestNumbero(t *testing.T) {
	t.Run("Numbero with integer", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Numbero(q),
				Eq(q, NewAtom(42)),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(42)) {
			t.Error("Expected 42, got", results[0])
		}
	})

	t.Run("Numbero with float", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Numbero(q),
				Eq(q, NewAtom(3.14)),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("Numbero with string fails", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Eq(q, NewAtom("not-a-number")),
				Numbero(q),
			)
		})

		if len(results) != 0 {
			t.Error("Number constraint with string should fail")
		}
	})
}

// TestMembero tests the membership relation.
func TestMembero(t *testing.T) {
	t.Run("Membero find elements", func(t *testing.T) {
		list := List(NewAtom(1), NewAtom(2), NewAtom(3))

		results := Run(5, func(q *Var) Goal {
			return Membero(q, list)
		})

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		// Check that we got all three elements
		found := make(map[int]bool)
		for _, result := range results {
			if atom, ok := result.(*Atom); ok {
				if val, ok := atom.Value().(int); ok {
					found[val] = true
				}
			}
		}

		for i := 1; i <= 3; i++ {
			if !found[i] {
				t.Errorf("Expected to find %d in results", i)
			}
		}
	})

	t.Run("Membero check membership", func(t *testing.T) {
		list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))

		results := Run(1, func(q *Var) Goal {
			return Conj(
				Membero(NewAtom("b"), list),
				Eq(q, NewAtom("found")),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("Membero non-member", func(t *testing.T) {
		list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))

		results := Run(1, func(q *Var) Goal {
			return Membero(NewAtom("d"), list)
		})

		if len(results) != 0 {
			t.Error("Non-member should return no results")
		}
	})
}

// TestOnceo tests the once constraint.
func TestOnceo(t *testing.T) {
	t.Run("Onceo limits solutions", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			return Onceo(Disj(
				Eq(q, NewAtom(1)),
				Eq(q, NewAtom(2)),
				Eq(q, NewAtom(3)),
			))
		})

		if len(results) != 1 {
			t.Fatalf("Onceo should return only 1 result, got %d", len(results))
		}
	})
}

// TestConda tests committed choice.
func TestConda(t *testing.T) {
	t.Run("Conda first condition succeeds", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			return Conda(
				[]Goal{Eq(q, NewAtom(1)), Eq(q, NewAtom(1))}, // condition succeeds
				[]Goal{Success, Eq(q, NewAtom(2))},           // should not be tried
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(1)) {
			t.Error("Expected 1, got", results[0])
		}
	})

	t.Run("Conda second condition succeeds", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			return Conda(
				[]Goal{Failure, Eq(q, NewAtom(1))}, // condition fails
				[]Goal{Success, Eq(q, NewAtom(2))}, // condition succeeds
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(2)) {
			t.Error("Expected 2, got", results[0])
		}
	})
}

// TestCondu tests committed choice with uniqueness.
func TestCondu(t *testing.T) {
	t.Run("Condu with unique condition", func(t *testing.T) {
		results := Run(5, func(q *Var) Goal {
			return Condu(
				[]Goal{Eq(q, NewAtom(1)), Eq(q, NewAtom(1))}, // unique condition
				[]Goal{Success, Eq(q, NewAtom(2))},           // should not be tried
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(1)) {
			t.Error("Expected 1, got", results[0])
		}
	})

	t.Run("Condu with non-unique condition", func(t *testing.T) {
		x := Fresh("x")
		results := Run(5, func(q *Var) Goal {
			return Condu(
				[]Goal{Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))), Eq(q, NewAtom(1))}, // non-unique
				[]Goal{Success, Eq(q, NewAtom(2))},                                    // should be tried
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(2)) {
			t.Error("Expected 2, got", results[0])
		}
	})
}

// TestProject tests variable projection.
func TestProject(t *testing.T) {
	t.Run("Project extracts values", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			x := Fresh("x")
			y := Fresh("y")

			return Conj(
				Eq(x, NewAtom(10)),
				Eq(y, NewAtom(20)),
				Project([]Term{x, y}, func(values []Term) Goal {
					// Extract the numeric values and add them
					if atom1, ok := values[0].(*Atom); ok {
						if atom2, ok := values[1].(*Atom); ok {
							if val1, ok := atom1.Value().(int); ok {
								if val2, ok := atom2.Value().(int); ok {
									sum := val1 + val2
									return Eq(q, NewAtom(sum))
								}
							}
						}
					}
					return Failure
				}),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom(30)) {
			t.Error("Expected 30, got", results[0])
		}
	})
}

// TestListOperations tests car, cdr, cons, etc.
func TestListOperations(t *testing.T) {
	t.Run("Car extracts first element", func(t *testing.T) {
		list := List(NewAtom("first"), NewAtom("second"))

		results := Run(1, func(q *Var) Goal {
			return Car(list, q)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("first")) {
			t.Error("Expected 'first', got", results[0])
		}
	})

	t.Run("Cdr extracts rest", func(t *testing.T) {
		list := List(NewAtom("first"), NewAtom("second"))

		results := Run(1, func(q *Var) Goal {
			return Cdr(list, q)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		// Result should be (second . nil)
		expectedRest := List(NewAtom("second"))
		if !results[0].Equal(expectedRest) {
			t.Error("Cdr result mismatch")
		}
	})

	t.Run("Cons creates pair", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Cons(NewAtom("head"), Nil, q)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		expected := List(NewAtom("head"))
		if !results[0].Equal(expected) {
			t.Error("Cons result mismatch")
		}
	})

	t.Run("Nullo checks for empty list", func(t *testing.T) {
		results := Run(1, func(q *Var) Goal {
			return Conj(
				Nullo(q),
				Eq(q, Nil),
			)
		})

		if len(results) != 1 {
			t.Error("Nullo should succeed with nil")
		}
	})

	t.Run("Pairo checks for pair", func(t *testing.T) {
		list := List(NewAtom("a"))

		results := Run(1, func(q *Var) Goal {
			return Conj(
				Pairo(list),
				Eq(q, NewAtom("success")),
			)
		})

		if len(results) != 1 {
			t.Error("Pairo should succeed with non-empty list")
		}
	})
}

// TestComplexConstraints tests combinations of constraints.
func TestComplexConstraints(t *testing.T) {
	t.Run("Multiple constraints", func(t *testing.T) {
		results := Run(10, func(q *Var) Goal {
			return Conj(
				Membero(q, List(NewAtom("a"), NewAtom(1), NewAtom("b"), NewAtom(2))),
				Symbolo(q),
				Neq(q, NewAtom("a")),
			)
		})

		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Equal(NewAtom("b")) {
			t.Error("Expected 'b', got", results[0])
		}
	})
}

// TestConstraintBuilder tests the fluent constraint builder API.
func TestConstraintBuilder(t *testing.T) {
	t.Run("Build empty store", func(t *testing.T) {
		builder := NewConstraintBuilder()
		store := builder.Build()

		constraints := store.GetConstraints()
		if len(constraints) != 0 {
			t.Errorf("Expected 0 constraints, got %d", len(constraints))
		}
	})

	t.Run("Build with multiple constraints", func(t *testing.T) {
		x, y, z := Fresh("x"), Fresh("y"), Fresh("z")

		builder := NewConstraintBuilder().
			WithDisequality(x, y).
			WithType(z, SymbolType).
			WithAbsence(NewAtom("bad"), x)

		store := builder.Build()
		constraints := store.GetConstraints()

		if len(constraints) != 3 {
			t.Errorf("Expected 3 constraints, got %d", len(constraints))
		}

		// Test that constraints work
		results := Run(1, func(q *Var) Goal {
			return Conj(
				// Apply the built constraints
				func(ctx context.Context, s ConstraintStore) ResultStream {
					// Add constraints from builder to store
					for _, constraint := range constraints {
						if err := s.AddConstraint(constraint); err != nil {
							stream := NewStream()
							stream.Close() // Close immediately to indicate no solutions
							return stream
						}
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				// Bind variables to satisfy constraints
				Eq(x, NewAtom("good")),
				Eq(y, NewAtom("different")),
				Eq(z, NewAtom("symbol")),
			)
		})

		if len(results) != 1 {
			t.Error("Constraints should allow valid bindings")
		}
	})

	t.Run("BuildConstraints returns constraint list", func(t *testing.T) {
		x, y := Fresh("x"), Fresh("y")

		builder := NewConstraintBuilder().
			WithDisequality(x, y).
			WithMembership(x, List(NewAtom(1), NewAtom(2)))

		constraints := builder.BuildConstraints()

		if len(constraints) != 2 {
			t.Errorf("Expected 2 constraints, got %d", len(constraints))
		}

		// Verify constraint types
		foundDisequality := false
		foundMembership := false

		for _, constraint := range constraints {
			switch constraint.(type) {
			case *DisequalityConstraint:
				foundDisequality = true
			case *MembershipConstraint:
				foundMembership = true
			}
		}

		if !foundDisequality {
			t.Error("Expected to find disequality constraint")
		}
		if !foundMembership {
			t.Error("Expected to find membership constraint")
		}
	})
}

// TestDisequalityBuilder tests the disequality constraint builder.
func TestDisequalityBuilder(t *testing.T) {
	t.Run("NotEqualTo creates constraint", func(t *testing.T) {
		x, y := Fresh("x"), Fresh("y")

		constraint := Disequality(x).NotEqualTo(y)

		if constraint == nil {
			t.Fatal("Constraint should not be nil")
		}

		// Test the constraint works
		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom("a")),
				Eq(y, NewAtom("b")),
			)
		})

		if len(results) != 1 {
			t.Error("Disequality constraint should allow different values")
		}
	})

	t.Run("NotEqualTo constraint violation", func(t *testing.T) {
		x := Fresh("x")

		constraint := Disequality(x).NotEqualTo(x)

		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom("same")),
			)
		})

		if len(results) != 0 {
			t.Error("Disequality constraint should fail when values are equal")
		}
	})
}

// TestAbsenceBuilder tests the absence constraint builder.
func TestAbsenceBuilder(t *testing.T) {
	t.Run("NotIn creates constraint", func(t *testing.T) {
		x := Fresh("x")

		constraint := Absence(NewAtom("bad")).NotIn(x)

		if constraint == nil {
			t.Fatal("Constraint should not be nil")
		}

		// Test the constraint works
		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, List(NewAtom("good"), NewAtom("ok"))),
			)
		})

		if len(results) != 1 {
			t.Error("Absence constraint should allow valid structures")
		}
	})

	t.Run("NotIn constraint violation", func(t *testing.T) {
		x := Fresh("x")

		constraint := Absence(NewAtom("bad")).NotIn(x)

		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, List(NewAtom("good"), NewAtom("bad"))),
			)
		})

		if len(results) != 0 {
			t.Error("Absence constraint should fail when forbidden term is present")
		}
	})
}

// TestTypeBuilder tests the type constraint builder.
func TestTypeBuilder(t *testing.T) {
	t.Run("MustBe SymbolType", func(t *testing.T) {
		x := Fresh("x")

		constraint := Type(x).MustBe(SymbolType)

		if constraint == nil {
			t.Fatal("Constraint should not be nil")
		}

		// Test the constraint works
		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom("symbol")),
			)
		})

		if len(results) != 1 {
			t.Error("Type constraint should allow correct type")
		}
	})

	t.Run("MustBe SymbolType violation", func(t *testing.T) {
		x := Fresh("x")

		constraint := Type(x).MustBe(SymbolType)

		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom(42)), // Number, not symbol
			)
		})

		if len(results) != 0 {
			t.Error("Type constraint should fail with wrong type")
		}
	})

	t.Run("MustBe NumberType", func(t *testing.T) {
		x := Fresh("x")

		constraint := Type(x).MustBe(NumberType)

		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom(42)),
			)
		})

		if len(results) != 1 {
			t.Error("Type constraint should allow numbers")
		}
	})
}

// TestMembershipBuilder tests the membership constraint builder.
func TestMembershipBuilder(t *testing.T) {
	t.Run("In creates constraint", func(t *testing.T) {
		x := Fresh("x")
		list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))

		constraint := Membership(x).In(list)

		if constraint == nil {
			t.Fatal("Constraint should not be nil")
		}

		// Test the constraint works by binding x to a specific value
		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
				Eq(x, NewAtom("b")), // Bind x to something in the list
			)
		})

		if len(results) != 1 {
			t.Error("Membership constraint should succeed when element is in list")
		}
	})

	t.Run("In with specific element", func(t *testing.T) {
		list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))

		constraint := Membership(NewAtom("b")).In(list)

		results := Run(1, func(q *Var) Goal {
			return Conj(
				func(ctx context.Context, s ConstraintStore) ResultStream {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close()
						return stream
					}
					return Eq(q, NewAtom("success"))(ctx, s)
				},
			)
		})

		if len(results) != 1 {
			t.Error("Membership constraint should succeed for existing element")
		}
	})

	t.Run("In constraint violation", func(t *testing.T) {
		list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))

		constraint := Membership(NewAtom("d")).In(list)

		results := Run(1, func(q *Var) Goal {
			return func(ctx context.Context, s ConstraintStore) ResultStream {
				if err := s.AddConstraint(constraint); err != nil {
					stream := NewStream()
					stream.Close() // Close immediately to indicate no solutions
					return stream
				}
				return Eq(q, NewAtom("success"))(ctx, s)
			}
		})

		if len(results) != 0 {
			t.Error("Membership constraint should fail for non-existing element")
		}
	})
}

// ExampleNewConstraintBuilder demonstrates the fluent constraint builder API.
func ExampleNewConstraintBuilder() {
	x, y, z := Fresh("x"), Fresh("y"), Fresh("z")

	// Build constraints using the fluent API
	builder := NewConstraintBuilder().
		WithDisequality(x, y).
		WithType(z, SymbolType).
		WithAbsence(NewAtom("bad"), x)

	store := builder.Build()

	// Use the constraints in a goal
	results := Run(1, func(q *Var) Goal {
		return Conj(
			func(ctx context.Context, s ConstraintStore) ResultStream {
				// Add constraints from builder to store
				for _, constraint := range store.GetConstraints() {
					if err := s.AddConstraint(constraint); err != nil {
						stream := NewStream()
						stream.Close() // Close immediately to indicate no solutions
						return stream
					}
				}
				return Eq(q, NewAtom("success"))(ctx, s)
			},
			// Bind variables to satisfy constraints
			Eq(x, NewAtom("good")),
			Eq(y, NewAtom("different")),
			Eq(z, NewAtom("symbol")),
		)
	})

	fmt.Printf("Constraints satisfied: %t\n", len(results) == 1)

	// Output:
	// Constraints satisfied: true
}
