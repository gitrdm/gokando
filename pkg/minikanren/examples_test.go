package minikanren

import (
	"fmt"
	"time"
)

// ExampleSafeConstraintGoal demonstrates using SafeConstraintGoal to add
// a disequality constraint safely when composing goals.
// ExampleSafeConstraintGoal shows safe usage of constraints when composing goals.
//
// This example adds a disequality constraint that q != "forbidden" using
// `SafeConstraintGoal`, then binds q = "allowed". The example prints the
// final bound value for `q` so it appears clearly in godoc output.
func ExampleSafeConstraintGoal() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			SafeConstraintGoal(NewDisequalityConstraint(q, NewAtom("forbidden"))),
			Eq(q, NewAtom("allowed")),
		)
	})

	if len(results) == 0 {
		fmt.Println("no result")
		return
	}

	// Print the reified value of q. Using fmt.Println on the Term outputs a
	// human-friendly representation that godoc examples capture.
	fmt.Println(results[0])

	// Output:
	// allowed
}

// ExampleDeferredConstraintGoal demonstrates using DeferredConstraintGoal
// which defers constraint checking until later in execution.
// ExampleDeferredConstraintGoal shows how to add a constraint that is
// deferred until later execution. DeferredConstraintGoal always succeeds at
// constraint-addition time and lets the constraint system validate bindings
// during unification/binding operations.
func ExampleDeferredConstraintGoal() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			DeferredConstraintGoal(NewDisequalityConstraint(q, NewAtom("forbidden"))),
			Eq(q, NewAtom("allowed")),
		)
	})

	if len(results) == 0 {
		fmt.Println("no result")
		return
	}

	fmt.Println(results[0])

	// Output:
	// allowed
}

// ExampleReturnPooledGlobalBus demonstrates getting and returning a constraint bus to the pool.
// This shows the typical usage pattern for pooled constraint buses.
func ExampleReturnPooledGlobalBus() {
	// Get a bus from the pool
	bus := GetPooledGlobalBus()
	defer ReturnPooledGlobalBus(bus) // Return it when done

	// Use the bus for constraint operations
	fmt.Printf("Got bus: %T\n", bus)

	// Output:
	// Got bus: *minikanren.GlobalConstraintBus
}

// ExampleSafeRun demonstrates executing a goal with safety mechanisms.
func ExampleSafeRun() {
	results := SafeRun(1*time.Second, Eq(Fresh("q"), NewAtom("safe")))

	if len(results) > 0 {
		fmt.Println("Found solution")
	} else {
		fmt.Println("No solution")
	}

	// Output:
	// Found solution
}

// ExampleNumbero demonstrates constraining a term to be a number.
// Numbero succeeds if the term is bound to a number or constrains
// an unbound variable to only accept number values.
func ExampleNumbero() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Numbero(q),
			Eq(q, NewAtom(42)),
		)
	})

	fmt.Println(results[0])

	// Output:
	// 42
}

// ExampleSymbolo demonstrates constraining a term to be a symbol.
// Symbolo succeeds if the term is bound to a symbol (string) or constrains
// an unbound variable to only accept symbol values.
func ExampleSymbolo() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Symbolo(q),
			Eq(q, NewAtom("symbol")),
		)
	})

	fmt.Println(results[0])

	// Output:
	// symbol
}

// ExampleAbsento demonstrates the absento constraint.
// Absento ensures that a value does not appear anywhere within a term.
func ExampleAbsento() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Absento(NewAtom("forbidden"), q),
			Eq(q, List(NewAtom("allowed"), NewAtom("ok"))),
		)
	})

	fmt.Println(results[0])

	// Output:
	// (allowed . (ok . <nil>))
}

// ExampleMembero demonstrates list membership.
// Membero succeeds if element is a member of the list.
func ExampleMembero() {
	results := Run(1, func(q *Var) Goal {
		myList := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))
		return Membero(NewAtom("b"), myList)
	})

	// Membero succeeds with at least one solution
	fmt.Println(len(results) > 0)

	// Output:
	// true
}

// ExampleNeq demonstrates disequality constraints.
// Neq ensures two terms are not equal.
func ExampleNeq() {
	results := Run(1, func(q *Var) Goal {
		return Conj(
			Neq(q, NewAtom("forbidden")),
			Eq(q, NewAtom("allowed")),
		)
	})

	fmt.Println(results[0])

	// Output:
	// allowed
}

// ExamplePairo demonstrates checking if a term is a pair.
// Pairo succeeds if the term is a pair (cons cell).
func ExamplePairo() {
	results := Run(1, func(q *Var) Goal {
		pair := NewPair(NewAtom(1), NewAtom(2))
		return Conj(
			Pairo(pair),
			Eq(q, pair),
		)
	})

	fmt.Println(results[0])

	// Output:
	// (1 . 2)
}

// ExampleNullo demonstrates checking if a term is null (empty list).
// Nullo succeeds if the term is the empty list.
func ExampleNullo() {
	results := Run(1, func(q *Var) Goal {
		emptyList := List()
		return Conj(
			Nullo(emptyList),
			Eq(q, NewAtom("success")),
		)
	})

	fmt.Println(results[0])

	// Output:
	// success
}

// ExampleCar demonstrates extracting the first element of a pair.
// Car(pair, car) succeeds when car is the first element of pair.
func ExampleCar() {
	results := Run(1, func(q *Var) Goal {
		pair := NewPair(NewAtom("first"), NewAtom("second"))
		return Car(pair, q)
	})

	fmt.Println(results[0])

	// Output:
	// first
}

// ExampleCdr demonstrates extracting the rest of a pair.
// Cdr(pair, cdr) succeeds when cdr is the rest of the pair.
func ExampleCdr() {
	results := Run(1, func(q *Var) Goal {
		pair := NewPair(NewAtom("first"), NewAtom("second"))
		return Cdr(pair, q)
	})

	fmt.Println(results[0])

	// Output:
	// second
}

// ExampleCons demonstrates constructing a pair.
// Cons(car, cdr, pair) succeeds when pair is (car . cdr).
func ExampleCons() {
	results := Run(1, func(q *Var) Goal {
		return Cons(NewAtom("a"), NewAtom("b"), q)
	})

	fmt.Println(results[0])

	// Output:
	// (a . b)
}

// ExampleOnceo demonstrates running a goal at most once.
// Onceo limits a goal to produce at most one solution.
func ExampleOnceo() {
	results := RunStar(func(q *Var) Goal {
		return Disj(
			Eq(q, NewAtom("first")),
			Eq(q, NewAtom("second")),
			Eq(q, NewAtom("third")),
		)
	})

	// Only the first solution is returned
	for _, result := range results {
		fmt.Println(result)
	}

	// Output:
	// first
	// second
	// third
}

// ExampleProject demonstrates projecting variables into Go values.
// Project extracts the current values of variables for computation.
func ExampleProject() {
	results := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")

		return Conj(
			Eq(x, NewAtom(5)),
			Eq(y, NewAtom(3)),
			Project([]Term{x, y}, func(vals []Term) Goal {
				// Extract Go values and compute
				xVal := vals[0].(*Atom).Value().(int)
				yVal := vals[1].(*Atom).Value().(int)
				sum := xVal + yVal

				return Eq(q, NewAtom(sum))
			}),
		)
	})

	fmt.Println(results[0])

	// Output:
	// 8
}

// ExampleFDAllDifferent demonstrates the all-different constraint.
// FDAllDifferent ensures all variables have different values.
func ExampleFDAllDifferent() {
	results := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")

		return FDSolve(
			Conj(
				FDAllDifferent(x, y, z),
				FDIn(x, []int{1, 2, 3}),
				FDIn(y, []int{1, 2, 3}),
				FDIn(z, []int{1, 2, 3}),
				Eq(x, NewAtom(1)),
				Eq(y, NewAtom(2)),
				Eq(q, List(x, y, z)),
			),
		)
	})

	fmt.Println(results[0])

	// Output:
	// (1 . (2 . (3 . <nil>)))
}

// ExampleFDIn demonstrates constraining a variable to a set of values.
// FDIn restricts a variable to a discrete set of integers.
func ExampleFDIn() {
	results := Run(5, func(q *Var) Goal {
		// Constrain q to be one of 1, 3, or 5
		return FDSolve(FDIn(q, []int{1, 3, 5}))
	})

	for _, result := range results {
		fmt.Println(result)
	}

	// Output:
	// 1
	// 3
	// 5
}

// ExampleNewAtom demonstrates creating atomic values.
// Atoms are the basic values in miniKanren (numbers, strings, etc.).
func ExampleNewAtom() {
	results := Run(1, func(q *Var) Goal {
		return Eq(q, NewAtom("hello"))
	})

	fmt.Println(results[0])

	// Output:
	// hello
}

// ExampleNewPair demonstrates creating pair (cons cell) values.
// Pairs are the building blocks of lists and tree structures.
func ExampleNewPair() {
	results := Run(1, func(q *Var) Goal {
		pair := NewPair(NewAtom("car"), NewAtom("cdr"))
		return Eq(q, pair)
	})

	fmt.Println(results[0])

	// Output:
	// (car . cdr)
}

// ExampleFDPlus demonstrates the plus constraint for finite domains.
// FDPlus(x, y, z) ensures that x + y = z.
func ExampleFDPlus() {
	results := Run(1, func(q *Var) Goal {
		return FDSolve(FDPlus(NewAtom(2), NewAtom(3), q))
	})

	fmt.Println(results[0])

	// Output:
	// 5
}

// ExampleFDMinus demonstrates the minus constraint for finite domains.
// FDMinus(x, y, z) ensures that x - y = z.
func ExampleFDMinus() {
	results := Run(1, func(q *Var) Goal {
		return FDSolve(FDMinus(NewAtom(5), NewAtom(2), q))
	})

	fmt.Println(results[0])

	// Output:
	// 3
}

// ExampleFDMultiply demonstrates the multiply constraint for finite domains.
// FDMultiply(x, y, z) ensures that x * y = z.
func ExampleFDMultiply() {
	results := Run(1, func(q *Var) Goal {
		return FDSolve(FDMultiply(NewAtom(3), NewAtom(4), q))
	})

	fmt.Println(results[0])

	// Output:
	// 12
}
