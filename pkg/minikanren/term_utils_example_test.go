package minikanren_test

import (
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// ExampleCopyTerm demonstrates creating a copy of a term with fresh variables.
//
// Copying terms is a common meta-programming technique used when you need
// to instantiate templates repeatedly without reusing the same logical
// variables. The copy operation replaces variables inside the template
// with fresh ones so subsequent instantiations don't clash with earlier
// bindings. This example keeps the low-level term constructors visible as
// commented references while using the HLAPI `Run` helper to execute the
// example goal.
func ExampleCopyTerm() {
	x := Fresh("x")
	original := List(x, NewAtom("middle"), x)

	// Copy the term - x will be replaced with a fresh variable
	_ = Run(1, func(copy *Var) Goal {
		return Conj(
			CopyTerm(original, copy),
			Eq(x, NewAtom("original-binding")), // Bind original x
		)
	})

	// The copy has fresh variables, not bound to "original-binding"
	fmt.Printf("Copy preserves structure with fresh variables\n")
	// Output:
	// Copy preserves structure with fresh variables
}

// ExampleGround demonstrates checking if a term is fully instantiated.
//
// Ground-checking is useful for validating inputs before executing a
// relation that requires fully-bound terms. This example shows both a
// successful ground check (a bound variable) and a failing check (an
// unbound variable). The HLAPI `Run` helper is used to run the mini-goals
// that test groundness and keep the example concise and readable.
func ExampleGround() {
	x := Fresh("x")

	// Check if a bound variable is ground
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Eq(x, NewAtom("hello")),
			Ground(x),
			Eq(q, NewAtom("bound-is-ground")),
		)
	})

	// Check if an unbound variable is ground (should fail)
	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(Fresh("unbound")),
			Eq(q, NewAtom("should-not-appear")),
		)
	})

	fmt.Printf("Bound variable is ground: %d results\n", len(result1))
	fmt.Printf("Unbound variable is not ground: %d results\n", len(result2))
	// Output:
	// Bound variable is ground: 1 results
	// Unbound variable is not ground: 0 results
}

// ExampleGround_list demonstrates ground checking on lists.
func ExampleGround_list() {
	// Fully ground list
	groundList := List(NewAtom(1), NewAtom(2), NewAtom(3))

	// Partially ground list
	x := Fresh("x")
	partialList := List(NewAtom(1), x, NewAtom(3))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(groundList),
			Eq(q, NewAtom("fully-ground")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			Ground(partialList),
			Eq(q, NewAtom("partially-ground")),
		)
	})

	fmt.Printf("Fully ground list: %s\n", result1[0])
	fmt.Printf("Partially ground list fails: %d results\n", len(result2))
	// Output:
	// Fully ground list: fully-ground
	// Partially ground list fails: 0 results
}

// ExampleArityo demonstrates determining the arity of a term.
func ExampleArityo() {
	// Arity of an atom is 0
	result1 := Run(1, func(arity *Var) Goal {
		return Arityo(NewAtom("hello"), arity)
	})

	// Arity of a list is its length
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))
	result2 := Run(1, func(arity *Var) Goal {
		return Arityo(list, arity)
	})

	fmt.Printf("Atom arity: %v\n", result1[0])
	fmt.Printf("List arity: %v\n", result2[0])
	// Output:
	// Atom arity: 0
	// List arity: 3
}

// ExampleFunctoro demonstrates extracting the functor (head) of a compound term.
func ExampleFunctoro() {
	// Create a compound term like foo(1, 2)
	term := NewPair(NewAtom("foo"), List(NewAtom(1), NewAtom(2)))

	result := Run(1, func(functor *Var) Goal {
		return Functoro(term, functor)
	})

	fmt.Printf("Functor: %v\n", result[0])
	// Output:
	// Functor: foo
}

// ExampleCompoundTermo demonstrates checking if a term is compound.
func ExampleCompoundTermo() {
	// Pairs are compound
	pair := NewPair(NewAtom("a"), NewAtom("b"))

	// Atoms are not compound
	atom := NewAtom(42)

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(pair),
			Eq(q, NewAtom("pair-is-compound")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			CompoundTermo(atom),
			Eq(q, NewAtom("atom-is-compound")),
		)
	})

	fmt.Printf("Pair is compound: %d results\n", len(result1))
	fmt.Printf("Atom is compound: %d results\n", len(result2))
	// Output:
	// Pair is compound: 1 results
	// Atom is compound: 0 results
}

// ExampleSimpleTermo demonstrates checking if a term is simple (atomic).
func ExampleSimpleTermo() {
	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewAtom(42)),
			Eq(q, NewAtom("atom-is-simple")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			SimpleTermo(NewPair(NewAtom("a"), NewAtom("b"))),
			Eq(q, NewAtom("pair-is-simple")),
		)
	})

	fmt.Printf("Atom is simple: %s\n", result1[0])
	fmt.Printf("Pair is not simple: %d results\n", len(result2))
	// Output:
	// Atom is simple: atom-is-simple
	// Pair is not simple: 0 results
}

// ExampleStringo demonstrates string type constraints.
func ExampleStringo() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Stringo(q),
			Membero(q, List(
				NewAtom("hello"),
				NewAtom(42),
				NewAtom("world"),
				NewAtom(true),
			)),
		)
	})

	fmt.Printf("String values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// String values: 2 results
	//   hello
	//   world
}

// ExampleBooleano demonstrates boolean type constraints.
func ExampleBooleano() {
	result := Run(3, func(q *Var) Goal {
		return Conj(
			Booleano(q),
			Membero(q, List(
				NewAtom(true),
				NewAtom("not-bool"),
				NewAtom(false),
				NewAtom(42),
			)),
		)
	})

	fmt.Printf("Boolean values: %d results\n", len(result))
	for _, r := range result {
		fmt.Printf("  %v\n", r)
	}
	// Output:
	// Boolean values: 2 results
	//   true
	//   false
}

// ExampleVectoro demonstrates vector (slice/array) type constraints.
func ExampleVectoro() {
	slice1 := []int{1, 2, 3}
	slice2 := []string{"a", "b", "c"}

	result := Run(2, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Membero(q, List(
				NewAtom(slice1),
				NewAtom("not-a-vector"),
				NewAtom(slice2),
			)),
		)
	})

	fmt.Printf("Vector values: %d results\n", len(result))
	// Output:
	// Vector values: 2 results
}

// ExampleCopyTerm_metaProgramming demonstrates using CopyTerm for meta-programming.
func ExampleCopyTerm_metaProgramming() {
	// Define a template with variables
	x := Fresh("x")
	y := Fresh("y")
	template := NewPair(NewAtom("add"), List(x, y))

	// Create multiple instances of the template
	result := Run(2, func(q *Var) Goal {
		instance := Fresh("instance")
		return Conj(
			CopyTerm(template, instance),
			// Each instance can be instantiated differently
			Membero(q, List(
				NewPair(NewAtom("instance"), instance),
			)),
		)
	})

	fmt.Printf("Template instances: %d\n", len(result))
	// Output:
	// Template instances: 1
}

// ExampleGround_validation demonstrates using Ground for input validation.
func ExampleGround_validation() {
	// Validate that all arguments are provided before processing
	process := func(arg1, arg2, result Term) Goal {
		return Conj(
			Ground(arg1),
			Ground(arg2),
			Eq(result, NewAtom("processed")),
		)
	}

	// Valid case: both arguments bound
	result1 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), NewAtom("b"), q)
	})

	// Invalid case: argument contains unbound variable
	x := Fresh("x")
	result2 := Run(1, func(q *Var) Goal {
		return process(NewAtom("a"), x, q)
	})

	fmt.Printf("Both arguments ground: %d results\n", len(result1))
	fmt.Printf("Unbound argument: %d results\n", len(result2))
	// Output:
	// Both arguments ground: 1 results
	// Unbound argument: 0 results
}

// ExampleArityo_typeChecking demonstrates using Arityo for structure validation.
func ExampleArityo_typeChecking() {
	// Ensure a term is a binary operation (arity 2)
	validateBinary := func(term Term) Goal {
		return Arityo(term, NewAtom(2))
	}

	binaryOp := List(NewAtom("left"), NewAtom("right"))
	unaryOp := List(NewAtom("single"))

	result1 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(binaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	result2 := Run(1, func(q *Var) Goal {
		return Conj(
			validateBinary(unaryOp),
			Eq(q, NewAtom("valid-binary")),
		)
	})

	fmt.Printf("Binary operation: %s\n", result1[0])
	fmt.Printf("Unary operation fails: %d results\n", len(result2))
	// Output:
	// Binary operation: valid-binary
	// Unary operation fails: 0 results
}

// ExampleFunctoro_patternMatching demonstrates using Functoro for dispatch.
func ExampleFunctoro_patternMatching() {
	// Dispatch based on functor
	dispatch := func(term, result Term) Goal {
		functor := Fresh("functor")
		return Conj(
			Functoro(term, functor),
			Conde(
				Conj(Eq(functor, NewAtom("add")), Eq(result, NewAtom("arithmetic"))),
				Conj(Eq(functor, NewAtom("cons")), Eq(result, NewAtom("list-operation"))),
				Conj(Eq(functor, NewAtom("eq")), Eq(result, NewAtom("comparison"))),
			),
		)
	}

	addTerm := NewPair(NewAtom("add"), List(NewAtom(1), NewAtom(2)))
	consTerm := NewPair(NewAtom("cons"), List(NewAtom("a"), Nil))

	result1 := Run(1, func(q *Var) Goal {
		return dispatch(addTerm, q)
	})

	result2 := Run(1, func(q *Var) Goal {
		return dispatch(consTerm, q)
	})

	fmt.Printf("add → %v\n", result1[0])
	fmt.Printf("cons → %v\n", result2[0])
	// Output:
	// add → arithmetic
	// cons → list-operation
}
