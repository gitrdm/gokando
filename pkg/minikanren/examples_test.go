package minikanren

import (
	"fmt"
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
