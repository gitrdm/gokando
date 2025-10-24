// Package main demonstrates basic miniKanren usage patterns.package example

// This example shows how to use the core primitives to solve
// simple relational programming problems.
package main

import (
	"context"
	"fmt"
	"time"

	"gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== GoKanren Examples ===")
	fmt.Println()

	basicUnification()
	multipleChoices()
	listOperations()
	relationExample()
	parallelExample()
	performanceComparison()
}

// basicUnification demonstrates simple unification.
func basicUnification() {
	fmt.Println("1. Basic Unification:")

	// Find values for q such that q = "hello"
	results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Eq(q, minikanren.NewAtom("hello"))
	})

	fmt.Printf("   q = \"hello\" => %v\n", results)

	// Find values for q such that q = 42
	results = minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Eq(q, minikanren.NewAtom(42))
	})

	fmt.Printf("   q = 42 => %v\n", results)
	fmt.Println()
}

// multipleChoices demonstrates disjunction (choice points).
func multipleChoices() {
	fmt.Println("2. Multiple Choices (Disjunction):")

	// Find values for q where q can be 1, 2, or 3
	results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Disj(
			minikanren.Eq(q, minikanren.NewAtom(1)),
			minikanren.Eq(q, minikanren.NewAtom(2)),
			minikanren.Eq(q, minikanren.NewAtom(3)),
		)
	})

	fmt.Printf("   q ∈ {1, 2, 3} => %v\n", results)

	// Mix of different types
	results = minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Disj(
			minikanren.Eq(q, minikanren.NewAtom("hello")),
			minikanren.Eq(q, minikanren.NewAtom(42)),
			minikanren.Eq(q, minikanren.NewAtom(true)),
		)
	})

	fmt.Printf("   q ∈ {\"hello\", 42, true} => %v\n", results)
	fmt.Println()
}

// listOperations demonstrates list construction and manipulation.
func listOperations() {
	fmt.Println("3. List Operations:")

	// Create lists using the List helper
	list123 := minikanren.List(
		minikanren.NewAtom(1),
		minikanren.NewAtom(2),
		minikanren.NewAtom(3),
	)

	// Find q such that q equals our list
	results := minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Eq(q, list123)
	})

	fmt.Printf("   q = [1, 2, 3] => %v\n", results)

	// Demonstrate list appending with Appendo
	// Simple case: append two concrete lists
	results = minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		list12 := minikanren.List(minikanren.NewAtom(1), minikanren.NewAtom(2))
		list34 := minikanren.List(minikanren.NewAtom(3), minikanren.NewAtom(4))

		return minikanren.Appendo(list12, list34, q)
	})

	fmt.Printf("   append([1, 2], [3, 4]) => %d result(s)\n", len(results))

	// Reverse: given the result, find what was appended to [3,4] to get [1,2,3,4]
	results = minikanren.Run(1, func(q *minikanren.Var) minikanren.Goal {
		list34 := minikanren.List(minikanren.NewAtom(3), minikanren.NewAtom(4))
		list1234 := minikanren.List(
			minikanren.NewAtom(1), minikanren.NewAtom(2),
			minikanren.NewAtom(3), minikanren.NewAtom(4),
		)

		return minikanren.Appendo(q, list34, list1234)
	})

	fmt.Printf("   What + [3, 4] = [1, 2, 3, 4]? => %d result(s)\n", len(results))
	if len(results) > 0 {
		fmt.Printf("   First result: %v\n", results[0])
	}
	fmt.Println()
}

// relationExample demonstrates a more complex relational program.
func relationExample() {
	fmt.Println("4. Relational Programming:")

	// Define a relation: likes(Person, Food)
	likes := func(person, food minikanren.Term) minikanren.Goal {
		return minikanren.Disj(
			minikanren.Conj(
				minikanren.Eq(person, minikanren.NewAtom("alice")),
				minikanren.Eq(food, minikanren.NewAtom("pizza")),
			),
			minikanren.Conj(
				minikanren.Eq(person, minikanren.NewAtom("bob")),
				minikanren.Eq(food, minikanren.NewAtom("burgers")),
			),
			minikanren.Conj(
				minikanren.Eq(person, minikanren.NewAtom("alice")),
				minikanren.Eq(food, minikanren.NewAtom("salad")),
			),
		)
	}

	// Query: What does Alice like?
	results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return likes(minikanren.NewAtom("alice"), q)
	})

	fmt.Printf("   What does Alice like? => %v\n", results)

	// Query: Who likes pizza?
	results = minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return likes(q, minikanren.NewAtom("pizza"))
	})

	fmt.Printf("   Who likes pizza? => %v\n", results)

	// Query: All person-food pairs
	results = minikanren.Run(10, func(q *minikanren.Var) minikanren.Goal {
		person := minikanren.Fresh("person")
		food := minikanren.Fresh("food")

		return minikanren.Conj(
			likes(person, food),
			minikanren.Eq(q, minikanren.List(person, food)),
		)
	})

	fmt.Printf("   All person-food pairs => %v\n", results)
	fmt.Println()
}

// parallelExample demonstrates parallel execution capabilities.
func parallelExample() {
	fmt.Println("5. Parallel Execution:")

	// Create a computationally intensive goal for demonstration
	heavyGoal := func(value int) func(*minikanren.Var) minikanren.Goal {
		return func(q *minikanren.Var) minikanren.Goal {
			return func(ctx context.Context, store minikanren.ConstraintStore) *minikanren.Stream {
				// Simulate meaningful computational work
				time.Sleep(50 * time.Millisecond) // Increased from 10ms

				// Now bind q to our value
				return minikanren.Eq(q, minikanren.NewAtom(value))(ctx, store)
			}
		}
	}

	// Sequential execution
	start := time.Now()
	results := minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return minikanren.Disj(
			heavyGoal(1)(q),
			heavyGoal(2)(q),
			heavyGoal(3)(q),
			heavyGoal(4)(q),
			heavyGoal(5)(q),
		)
	})
	sequentialTime := time.Since(start)

	fmt.Printf("   Sequential results: %v (took %v)\n", results, sequentialTime)

	// Parallel execution
	start = time.Now()
	executor := minikanren.NewParallelExecutor(nil)
	defer executor.Shutdown()

	results = minikanren.Run(5, func(q *minikanren.Var) minikanren.Goal {
		return executor.ParallelDisj(
			heavyGoal(1)(q),
			heavyGoal(2)(q),
			heavyGoal(3)(q),
			heavyGoal(4)(q),
			heavyGoal(5)(q),
		)
	})
	parallelTime := time.Since(start)

	fmt.Printf("   Parallel results: %v (took %v)\n", results, parallelTime)

	if parallelTime < sequentialTime {
		fmt.Printf("   Speedup: %.2fx\n", float64(sequentialTime)/float64(parallelTime))
	}
	fmt.Println()
}

// performanceComparison shows the difference between sequential and parallel execution.
func performanceComparison() {
	fmt.Println("6. Performance Comparison:")

	const numGoals = 8                      // Reduced for clearer demonstration
	const workDelay = 25 * time.Millisecond // Increased work

	// Create goal functions that work with the query variable
	createGoalFuncs := func() []func(*minikanren.Var) minikanren.Goal {
		goalFuncs := make([]func(*minikanren.Var) minikanren.Goal, numGoals)
		for i := 0; i < numGoals; i++ {
			value := i
			goalFuncs[i] = func(q *minikanren.Var) minikanren.Goal {
				return func(ctx context.Context, store minikanren.ConstraintStore) *minikanren.Stream {
					// Simulate computational work
					time.Sleep(workDelay)

					// Bind the query variable to our value
					return minikanren.Eq(q, minikanren.NewAtom(value))(ctx, store)
				}
			}
		}
		return goalFuncs
	}

	// Test sequential execution
	fmt.Printf("   Testing %d goals with %v work each...\n", numGoals, workDelay)

	start := time.Now()
	results := minikanren.Run(numGoals, func(q *minikanren.Var) minikanren.Goal {
		goalFuncs := createGoalFuncs()
		goals := make([]minikanren.Goal, len(goalFuncs))
		for i, gf := range goalFuncs {
			goals[i] = gf(q)
		}
		return minikanren.Disj(goals...)
	})
	sequentialTime := time.Since(start)

	fmt.Printf("   Sequential: %d results in %v\n", len(results), sequentialTime)

	// Test parallel execution
	config := &minikanren.ParallelConfig{
		MaxWorkers:         4,
		EnableBackpressure: true,
	}
	executor := minikanren.NewParallelExecutor(config)
	defer executor.Shutdown()

	start = time.Now()
	results = minikanren.Run(numGoals, func(q *minikanren.Var) minikanren.Goal {
		goalFuncs := createGoalFuncs()
		goals := make([]minikanren.Goal, len(goalFuncs))
		for i, gf := range goalFuncs {
			goals[i] = gf(q)
		}
		return executor.ParallelDisj(goals...)
	})
	parallelTime := time.Since(start)

	fmt.Printf("   Parallel: %d results in %v\n", len(results), parallelTime)

	if parallelTime < sequentialTime {
		speedup := float64(sequentialTime) / float64(parallelTime)
		fmt.Printf("   Speedup: %.2fx\n", speedup)
	} else {
		fmt.Printf("   No speedup (overhead dominated)\n")
	}

	fmt.Printf("   Theoretical max speedup: %dx (with %d workers)\n", config.MaxWorkers, config.MaxWorkers)
	fmt.Println()
}
