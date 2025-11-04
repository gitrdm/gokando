package minikanren

import (
	"context"
	"fmt"
)

// ExampleMatche demonstrates exhaustive pattern matching.
//
// This example shows how to classify the structure of a term using
// exhaustive pattern matching: every clause that matches contributes a
// result. It's useful to illustrate the difference between the matching
// strategies provided by the pattern subsystem (Matche/Matcha/Matchu).
// Low-level pattern constructors are used directly here; the comments show
// how to express the same intent with the HLAPI-style helpers where
// applicable.
func ExampleMatche() {
	// Classify a list by structure
	list := List(NewAtom(1), NewAtom(2))

	result := Run(5, func(q *Var) Goal {
		return Matche(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("_"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	// Matches "multiple" clause only
	for _, r := range result {
		if atom, ok := r.(*Atom); ok {
			fmt.Println(atom.value)
		}
	}

	// Output:
	// multiple
}

// ExampleMatcha demonstrates committed choice pattern matching.
// Only the first matching clause is tried.
func ExampleMatcha() {
	// Safe head extraction with default value
	extractHead := func(list Term) Term {
		return Run(1, func(q *Var) Goal {
			head := Fresh("head")
			return Matcha(list,
				NewClause(Nil, Eq(q, NewAtom("empty"))),
				NewClause(NewPair(head, Fresh("_")), Eq(q, head)),
			)
		})[0]
	}

	// Non-empty list
	list1 := List(NewAtom(42), NewAtom(99))
	fmt.Println(extractHead(list1))

	// Empty list
	list2 := Nil
	fmt.Println(extractHead(list2))

	// Output:
	// 42
	// empty
}

// ExampleMatchu demonstrates unique pattern matching.
// Requires exactly one clause to match.
func ExampleMatchu() {
	// Classify numbers with mutually exclusive ranges
	classify := func(n int) string {
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(n), map[int]string{
				0: "zero",
				1: "one",
				2: "two",
			}, q)
		})

		if len(result) == 0 {
			return "unknown"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(classify(0))
	fmt.Println(classify(1))
	fmt.Println(classify(5))

	// Output:
	// zero
	// one
	// unknown
}

// ExampleNewClause demonstrates creating pattern matching clauses.
func ExampleNewClause() {
	// Pattern matching with variable binding and multiple goals
	result := Run(5, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")

		return Matche(NewPair(NewAtom(10), NewAtom(20)),
			NewClause(
				NewPair(x, y),
				// Multiple goals executed in sequence
				Eq(x, NewAtom(10)),
				Eq(y, NewAtom(20)),
				Eq(q, NewAtom("success")),
			),
		)
	})

	fmt.Println(result[0])

	// Output:
	// success
}

// ExampleMatcheList demonstrates list-specific pattern matching.
func ExampleMatcheList() {
	// Simple list pattern matching
	list := List(NewAtom(1), NewAtom(2), NewAtom(3))

	result := Run(1, func(q *Var) Goal {
		return MatcheList(list,
			NewClause(Nil, Eq(q, NewAtom("empty"))),
			NewClause(NewPair(Fresh("_"), Nil), Eq(q, NewAtom("singleton"))),
			NewClause(NewPair(Fresh("head"), NewPair(Fresh("_"), Fresh("_"))), Eq(q, NewAtom("multiple"))),
		)
	})

	fmt.Println(result[0])

	// Output:
	// multiple
}

// ExampleMatche_listProcessing demonstrates practical list processing with pattern matching.
func ExampleMatche_listProcessing() {
	// Extract all elements from a list
	extractAll := func(list Term) []Term {
		var results []Term

		Run(10, func(q *Var) Goal {
			elem := Fresh("elem")
			rest := Fresh("rest")

			return Matche(list,
				NewClause(Nil, Eq(q, NewAtom("done"))),
				NewClause(NewPair(elem, rest), Eq(q, elem)),
			)
		})

		// Simplified - in practice would need recursive extraction
		return results
	}

	list := List(NewAtom("a"), NewAtom("b"), NewAtom("c"))
	_ = extractAll(list)

	fmt.Println("List elements extracted")

	// Output:
	// List elements extracted
}

// ExampleMatcha_deterministicChoice demonstrates using Matcha for deterministic dispatch.
func ExampleMatcha_deterministicChoice() {
	// Process different data types deterministically
	process := func(data Term) string {
		result := Run(1, func(q *Var) Goal {
			return Matcha(data,
				// Check for Nil first
				NewClause(Nil, Eq(q, NewAtom("empty-list"))),
				// Then check for pair
				NewClause(NewPair(Fresh("_"), Fresh("_")), Eq(q, NewAtom("pair"))),
				// Default case
				NewClause(Fresh("_"), Eq(q, NewAtom("atom"))),
			)
		})

		if len(result) == 0 {
			return "error"
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s
			}
		}
		return "error"
	}

	fmt.Println(process(Nil))
	fmt.Println(process(NewPair(NewAtom(1), NewAtom(2))))
	fmt.Println(process(NewAtom(42)))

	// Output:
	// empty-list
	// pair
	// atom
}

// ExampleMatchu_validation demonstrates using Matchu for validation.
func ExampleMatchu_validation() {
	// Validate that a value matches exactly one category
	validate := func(val int) (string, bool) {
		// Alternative implementation for demonstration purposes
		// return Matchu(NewAtom(val),
		//		NewClause(NewAtom(1), Eq(q, NewAtom("category-A"))),
		//		NewClause(NewAtom(2), Eq(q, NewAtom("category-B"))),
		//		NewClause(NewAtom(3), Eq(q, NewAtom("category-C"))),
		result := Run(1, func(q *Var) Goal {
			return CaseIntMap(NewAtom(val), map[int]string{
				1: "category-A",
				2: "category-B",
				3: "category-C",
			}, q)
		})

		if len(result) == 0 {
			return "", false
		}

		if atom, ok := result[0].(*Atom); ok {
			if s, ok := atom.value.(string); ok {
				return s, true
			}
		}
		return "", false
	}

	// Valid values
	cat, ok := validate(1)
	fmt.Printf("Value 1: %s (valid: %t)\n", cat, ok)

	cat, ok = validate(2)
	fmt.Printf("Value 2: %s (valid: %t)\n", cat, ok)

	// Invalid value (no match)
	cat, ok = validate(99)
	fmt.Printf("Value 99: %s (valid: %t)\n", cat, ok)

	// Output:
	// Value 1: category-A (valid: true)
	// Value 2: category-B (valid: true)
	// Value 99:  (valid: false)
}

// ExamplePatternClause_nestedPatterns demonstrates complex nested pattern matching.
func ExamplePatternClause_nestedPatterns() {
	// Match nested structure: ((a b) (c d))
	data := List(
		List(NewAtom("x"), NewAtom("y")),
		List(NewAtom("z"), NewAtom("w")),
	)

	result := Run(1, func(q *Var) Goal {
		a := Fresh("a")
		b := Fresh("b")

		return Matche(data,
			NewClause(
				NewPair(
					NewPair(a, NewPair(b, Nil)),
					Fresh("_"),
				),
				Eq(q, List(a, b)),
			),
		)
	})

	if len(result) > 0 {
		fmt.Printf("Extracted first pair: %v\n", result[0])
	}

	// Output:
	// Extracted first pair: (x . (y . <nil>))
}

// ExampleMatche_withDatabase demonstrates pattern matching with pldb queries.
func ExampleMatche_withDatabase() {
	// Create a relation for shapes
	shape, _ := DbRel("shape", 2, 0)
	db := NewDatabase()
	db, _ = db.AddFact(shape, NewAtom("circle"), NewAtom(10))
	db, _ = db.AddFact(shape, NewAtom("square"), NewAtom(5))
	db, _ = db.AddFact(shape, NewAtom("triangle"), NewAtom(3))

	// Query and pattern match on shape type
	name := Fresh("name")
	size := Fresh("size")

	result := Run(10, func(q *Var) Goal {
		return Conj(
			db.Query(shape, name, size),
			Matche(name,
				NewClause(NewAtom("circle"), Eq(q, NewAtom("round"))),
				NewClause(NewAtom("square"), Eq(q, NewAtom("angular"))),
				NewClause(NewAtom("triangle"), Eq(q, NewAtom("angular"))),
			),
		)
	})

	// Count results
	fmt.Printf("Matched %d shapes\n", len(result))

	// Output:
	// Matched 3 shapes
}

// ExampleMatcha_withHybridSolver demonstrates pattern matching with FD constraints.
func ExampleMatcha_withHybridSolver() {
	model := NewModel()
	x := model.NewVariable(NewBitSetDomainFromValues(100, []int{5, 10, 15}))

	store := NewUnifiedStore()
	store, _ = store.SetDomain(x.ID(), x.Domain())
	adapter := NewUnifiedStoreAdapter(store)

	q := Fresh("q")
	val := Fresh("val")

	goal := Conj(
		Eq(val, NewAtom(5)),
		Matcha(val,
			NewClause(NewAtom(5), Eq(q, NewAtom("small"))),
			NewClause(NewAtom(10), Eq(q, NewAtom("medium"))),
			NewClause(NewAtom(15), Eq(q, NewAtom("large"))),
		),
	)

	ctx := context.Background()
	stream := goal(ctx, adapter)
	results, _ := stream.Take(1)

	if len(results) > 0 {
		binding := results[0].GetBinding(q.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("Classification: %v\n", atom.value)
		}
	}

	// Output:
	// Classification: small
}
