package minikanren

import (
	"context"
)

// Neq creates a disequality constraint that ensures two terms are NOT equal.
// This is a constraint that's checked during unification and can cause
// goals to fail if the constraint would be violated.
//
// Example:
//
//	x := Fresh("x")
//	goal := Conj(Neq(x, NewAtom("forbidden")), Eq(x, NewAtom("allowed")))
//
// Neq implements the disequality constraint.
// It ensures that two terms are not equal.
func Neq(t1, t2 Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Create a disequality constraint and add it to the store
		constraint := NewDisequalityConstraint(t1, t2)

		err := store.AddConstraint(constraint)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(ctx, store)
			}
		}()
		return stream
	}
}

// Absento creates a constraint ensuring that a term does not appear anywhere
// within another term (at any level of structure).
//
// Example:
//
//	x := Fresh("x")
//	goal := Conj(Absento(NewAtom("bad"), x), Eq(x, List(NewAtom("good"))))
func Absento(absent, term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Create an absence constraint and add it to the store
		constraint := NewAbsenceConstraint(absent, term)

		err := store.AddConstraint(constraint)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(ctx, store)
			}
		}()
		return stream
	}
}

// occurs checks if a term occurs anywhere in another term's structure
func occurs(needle, haystack Term) bool {
	if needle.Equal(haystack) {
		return true
	}

	if pair, ok := haystack.(*Pair); ok {
		return occurs(needle, pair.Car()) || occurs(needle, pair.Cdr())
	}

	return false
}

// Symbolo constrains a term to be a symbol (string atom).
//
// Example:
//
//	x := Fresh("x")
//	goal := Conj(Symbolo(x), Eq(x, NewAtom("symbol")))
func Symbolo(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Create a type constraint for string symbols
		constraint := NewTypeConstraint(term, SymbolType)

		err := store.AddConstraint(constraint)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(ctx, store)
			}
		}()
		return stream
	}
}

// Numbero constrains a term to be a number.
//
// Example:
//
//	x := Fresh("x")
//	goal := Conj(Numbero(x), Eq(x, NewAtom(42)))
func Numbero(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		// Create a type constraint for numbers
		constraint := NewTypeConstraint(term, NumberType)

		err := store.AddConstraint(constraint)

		stream := NewStream()
		go func() {
			defer stream.Close()
			if err == nil {
				stream.Put(ctx, store)
			}
		}()
		return stream
	}
}

// Membero creates a goal that relates an element to a list it's a member of.
// This is the relational membership predicate.
//
// Example:
//
//	x := Fresh("x")
//	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
//	goal := Membero(x, list) // x can be 1, 2, or 3
func Membero(element, list Term) Goal {
	return Disj(
		// Base case: element is the first item of the list
		func(ctx context.Context, store ConstraintStore) ResultStream {
			car := Fresh("car")
			cdr := Fresh("cdr")

			return Conj(
				Eq(list, NewPair(car, cdr)),
				Eq(element, car),
			)(ctx, store)
		},

		// Recursive case: element is a member of the rest of the list
		func(ctx context.Context, store ConstraintStore) ResultStream {
			car := Fresh("car")
			cdr := Fresh("cdr")

			return Conj(
				Eq(list, NewPair(car, cdr)),
				Membero(element, cdr),
			)(ctx, store)
		},
	)
}

// Onceo ensures a goal succeeds at most once (cuts choice points).
//
// Example:
//
//	goal := Onceo(Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2))))
//	// Will only return the first solution
func Onceo(goal Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		goalStream := goal(ctx, store)

		stream := NewStream()
		go func() {
			defer stream.Close()

			// Take only the first solution
			solutions, _, err := goalStream.Take(ctx, 1)
			if err == nil && len(solutions) > 0 {
				stream.Put(ctx, solutions[0])
			}
		}()

		return stream
	}
}

// Conda implements committed choice (if-then-else with cut).
// Takes pairs of condition-goal clauses and commits to the first
// condition that succeeds.
//
// Example:
//
//	goal := Conda(
//	  []Goal{condition1, thenGoal1},
//	  []Goal{condition2, thenGoal2},
//	  []Goal{Success, elseGoal}, // default case
//	)
func Conda(clauses ...[]Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			for _, clause := range clauses {
				if len(clause) < 2 {
					continue
				}

				condition := clause[0]
				goal := clause[1]

				// Test the condition
				conditionStream := condition(ctx, store)
				solutions, hasMore, err := conditionStream.Take(ctx, 1)
				if err != nil {
					return
				}

				if len(solutions) > 0 {
					// Condition succeeded, commit to this clause
					for _, condStore := range solutions {
						goalStream := goal(ctx, condStore)

						// Forward all solutions from the goal
						for {
							goalSolutions, goalHasMore, goalErr := goalStream.Take(ctx, 1)
							if goalErr != nil {
								return
							}
							if len(goalSolutions) == 0 {
								if !goalHasMore {
									break
								}
								continue
							}

							stream.Put(ctx, goalSolutions[0])
						}
					}

					// If there are more solutions from the condition, process them too
					if hasMore {
						for {
							moreSolutions, moreHasMore, moreErr := conditionStream.Take(ctx, 1)
							if moreErr != nil {
								return
							}
							if len(moreSolutions) == 0 {
								if !moreHasMore {
									break
								}
								continue
							}

							for _, condStore := range moreSolutions {
								goalStream := goal(ctx, condStore)

								for {
									goalSolutions, goalHasMore, goalErr := goalStream.Take(ctx, 1)
									if goalErr != nil {
										return
									}
									if len(goalSolutions) == 0 {
										if !goalHasMore {
											break
										}
										continue
									}

									stream.Put(ctx, goalSolutions[0])
								}
							}
						}
					}

					return // Committed to this clause
				}
			}
		}()

		return stream
	}
}

// Condu implements committed choice with a unique solution requirement.
// Like Conda but only commits if the condition has exactly one solution.
//
// Example:
//
//	goal := Condu(
//	  []Goal{uniqueCondition, thenGoal},
//	  []Goal{Success, elseGoal},
//	)
func Condu(clauses ...[]Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		stream := NewStream()

		go func() {
			defer stream.Close()

			for _, clause := range clauses {
				if len(clause) < 2 {
					continue
				}

				condition := clause[0]
				goal := clause[1]

				// Test the condition and collect all solutions
				conditionStream := condition(ctx, store)
				solutions, _, err := conditionStream.Take(ctx, 2) // Take at most 2 to check uniqueness
				if err != nil {
					return
				}

				if len(solutions) == 1 {
					// Exactly one solution, commit to this clause
					goalStream := goal(ctx, solutions[0])

					// Forward all solutions from the goal
					for {
						goalSolutions, goalHasMore, goalErr := goalStream.Take(ctx, 1)
						if goalErr != nil {
							return
						}
						if len(goalSolutions) == 0 {
							if !goalHasMore {
								break
							}
							continue
						}

						stream.Put(ctx, goalSolutions[0])
					}

					return // Committed to this clause
				}
			}
		}()

		return stream
	}
}

// Project extracts the values of variables from the current substitution
// and passes them to a function that creates a new goal.
//
// Example:
//
//	goal := Project([]Term{x, y}, func(values []Term) Goal {
//	  // values[0] is the value of x, values[1] is the value of y
//	  return someGoalUsing(values)
//	})
func Project(vars []Term, goalFunc func([]Term) Goal) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		values := make([]Term, len(vars))
		for i, v := range vars {
			values[i] = store.GetSubstitution().Walk(v)
		}

		newGoal := goalFunc(values)
		return newGoal(ctx, store)
	}
}

// Nil represents the empty list
var Nil = NewAtom(nil)

// Car extracts the first element of a pair/list.
//
// Example:
//
//	goal := Car(List(NewAtom(1), NewAtom(2)), x) // x = 1
func Car(pair, car Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		cdr := Fresh("cdr")
		return Eq(pair, NewPair(car, cdr))(ctx, store)
	}
}

// Cdr extracts the rest of a pair/list.
//
// Example:
//
//	goal := Cdr(List(NewAtom(1), NewAtom(2)), x) // x = List(NewAtom(2))
func Cdr(pair, cdr Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		car := Fresh("car")
		return Eq(pair, NewPair(car, cdr))(ctx, store)
	}
}

// Cons creates a pair/list construction goal.
//
// Example:
//
//	goal := Cons(NewAtom(1), Nil, x) // x = List(NewAtom(1))
func Cons(car, cdr, pair Term) Goal {
	return Eq(pair, NewPair(car, cdr))
}

// Nullo checks if a term is the empty list (nil).
//
// Example:
//
//	goal := Nullo(x) // x must be nil
func Nullo(term Term) Goal {
	return Eq(term, Nil)
}

// Pairo checks if a term is a pair (non-empty list).
//
// Example:
//
//	goal := Pairo(x) // x must be a pair
func Pairo(term Term) Goal {
	return func(ctx context.Context, store ConstraintStore) ResultStream {
		car := Fresh("car")
		cdr := Fresh("cdr")
		return Eq(term, NewPair(car, cdr))(ctx, store)
	}
}
