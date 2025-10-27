// Package main solves the famous Zebra puzzle (Einstein's Riddle) using GoKando.
//
// The Zebra puzzle is a logic puzzle with the following constraints:
//   - There are five houses.
//   - The English man lives in the red house.
//   - The Swede has a dog.
//   - The Dane drinks tea.
//   - The green house is immediately to the left of the white house.
//   - They drink coffee in the green house.
//   - The man who smokes Pall Mall has a bird.
//   - In the yellow house they smoke Dunhill.
//   - In the middle house they drink milk.
//   - The Norwegian lives in the first house.
//   - The Blend-smoker lives in the house next to the house with a cat.
//   - In a house next to the house with a horse, they smoke Dunhill.
//   - The man who smokes Blue Master drinks beer.
//   - The German smokes Prince.
//   - The Norwegian lives next to the blue house.
//   - They drink water in a house next to the house where they smoke Blend.
//
// Question: Who owns the zebra?
package main

import (
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Zebra Puzzle with GoKando ===")
	fmt.Println()

	// Solve the puzzle - we expect exactly one solution
	results := Run(1, zebraPuzzle)

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Println("✓ Solution found!")
	fmt.Println()
	fmt.Println()

	displaySolution(results[0])
}

// zebraPuzzle defines the complete puzzle as a miniKanren goal.
// Each house is represented as a list: (nationality color pet drink smoke)
func zebraPuzzle(q *Var) Goal {
	// Define variables for each attribute of each house
	// House 1
	n1, c1, p1, d1, s1 := Fresh("n1"), Fresh("c1"), Fresh("p1"), Fresh("d1"), Fresh("s1")
	// House 2
	n2, c2, p2, d2, s2 := Fresh("n2"), Fresh("c2"), Fresh("p2"), Fresh("d2"), Fresh("s2")
	// House 3
	n3, c3, p3, d3, s3 := Fresh("n3"), Fresh("c3"), Fresh("p3"), Fresh("d3"), Fresh("s3")
	// House 4
	n4, c4, p4, d4, s4 := Fresh("n4"), Fresh("c4"), Fresh("p4"), Fresh("d4"), Fresh("s4")
	// House 5
	n5, c5, p5, d5, s5 := Fresh("n5"), Fresh("c5"), Fresh("p5"), Fresh("d5"), Fresh("s5")

	// Create house structures
	h1 := List(n1, c1, p1, d1, s1)
	h2 := List(n2, c2, p2, d2, s2)
	h3 := List(n3, c3, p3, d3, s3)
	h4 := List(n4, c4, p4, d4, s4)
	h5 := List(n5, c5, p5, d5, s5)

	houses := List(h1, h2, h3, h4, h5)

	return Conj(
		// Return the houses list as the result
		Eq(q, houses),

		// Apply the most restrictive constraints FIRST to prune search space early

		// Constraint 9: The Norwegian lives in the first house (MOST RESTRICTIVE - fixes position)
		Eq(n1, NewAtom("Norwegian")),

		// Constraint 8: In the middle house they drink milk (VERY RESTRICTIVE - fixes position)
		Eq(d3, NewAtom("milk")),

		// Constraint 14: The Norwegian lives next to the blue house
		// Since Norwegian is in house 1, blue must be in house 2
		Eq(c2, NewAtom("blue")),

		// Constraint 4: The green house is immediately to the left of the white house
		// Since c2 is blue, green-white must be 1-2 (no, c2=blue), 3-4, or 4-5
		Disj(
			Conj(Eq(c3, NewAtom("green")), Eq(c4, NewAtom("white"))),
			Conj(Eq(c4, NewAtom("green")), Eq(c5, NewAtom("white"))),
		),

		// Constraint 5: They drink coffee in the green house
		// This links color and drink
		Disj(
			Conj(Eq(c3, NewAtom("green")), Eq(d3, NewAtom("coffee"))), // But d3 = milk! Contradiction
			Conj(Eq(c4, NewAtom("green")), Eq(d4, NewAtom("coffee"))),
			Conj(Eq(c5, NewAtom("green")), Eq(d5, NewAtom("coffee"))),
		),

		// Now we know: c2=blue, d3=milk, and green!=c3 (would conflict with d3=milk)
		// So green is c4 or c5

		// Constraint 1: The English man lives in the red house
		Disj(
			Conj(Eq(n1, NewAtom("English")), Eq(c1, NewAtom("red"))), // But n1=Norwegian!
			Conj(Eq(n2, NewAtom("English")), Eq(c2, NewAtom("red"))), // But c2=blue!
			Conj(Eq(n3, NewAtom("English")), Eq(c3, NewAtom("red"))),
			Conj(Eq(n4, NewAtom("English")), Eq(c4, NewAtom("red"))),
			Conj(Eq(n5, NewAtom("English")), Eq(c5, NewAtom("red"))),
		),

		// Constraint 7: In the yellow house they smoke Dunhill
		Disj(
			Conj(Eq(c1, NewAtom("yellow")), Eq(s1, NewAtom("Dunhill"))),
			Conj(Eq(c3, NewAtom("yellow")), Eq(s3, NewAtom("Dunhill"))),
			Conj(Eq(c4, NewAtom("yellow")), Eq(s4, NewAtom("Dunhill"))),
			Conj(Eq(c5, NewAtom("yellow")), Eq(s5, NewAtom("Dunhill"))),
		),

		// Constraint 2: The Swede has a dog
		Disj(
			Conj(Eq(n2, NewAtom("Swede")), Eq(p2, NewAtom("dog"))),
			Conj(Eq(n3, NewAtom("Swede")), Eq(p3, NewAtom("dog"))),
			Conj(Eq(n4, NewAtom("Swede")), Eq(p4, NewAtom("dog"))),
			Conj(Eq(n5, NewAtom("Swede")), Eq(p5, NewAtom("dog"))),
		),

		// Constraint 3: The Dane drinks tea
		Disj(
			Conj(Eq(n1, NewAtom("Dane")), Eq(d1, NewAtom("tea"))),
			Conj(Eq(n2, NewAtom("Dane")), Eq(d2, NewAtom("tea"))),
			Conj(Eq(n4, NewAtom("Dane")), Eq(d4, NewAtom("tea"))),
			Conj(Eq(n5, NewAtom("Dane")), Eq(d5, NewAtom("tea"))),
		),

		// Constraint 6: The man who smokes Pall Mall has a bird
		Disj(
			Conj(Eq(s1, NewAtom("Pall Mall")), Eq(p1, NewAtom("bird"))),
			Conj(Eq(s2, NewAtom("Pall Mall")), Eq(p2, NewAtom("bird"))),
			Conj(Eq(s3, NewAtom("Pall Mall")), Eq(p3, NewAtom("bird"))),
			Conj(Eq(s4, NewAtom("Pall Mall")), Eq(p4, NewAtom("bird"))),
			Conj(Eq(s5, NewAtom("Pall Mall")), Eq(p5, NewAtom("bird"))),
		),

		// Constraint 12: The man who smokes Blue Master drinks beer
		Disj(
			Conj(Eq(s1, NewAtom("Blue Master")), Eq(d1, NewAtom("beer"))),
			Conj(Eq(s2, NewAtom("Blue Master")), Eq(d2, NewAtom("beer"))),
			Conj(Eq(s4, NewAtom("Blue Master")), Eq(d4, NewAtom("beer"))),
			Conj(Eq(s5, NewAtom("Blue Master")), Eq(d5, NewAtom("beer"))),
		),

		// Constraint 13: The German smokes Prince
		Disj(
			Conj(Eq(n2, NewAtom("German")), Eq(s2, NewAtom("Prince"))),
			Conj(Eq(n3, NewAtom("German")), Eq(s3, NewAtom("Prince"))),
			Conj(Eq(n4, NewAtom("German")), Eq(s4, NewAtom("Prince"))),
			Conj(Eq(n5, NewAtom("German")), Eq(s5, NewAtom("Prince"))),
		),

		// Constraint 10: The Blend-smoker lives next to the house with a cat
		Disj(
			Conj(Eq(s1, NewAtom("Blend")), Eq(p2, NewAtom("cat"))),
			Conj(Eq(s2, NewAtom("Blend")), Disj(Eq(p1, NewAtom("cat")), Eq(p3, NewAtom("cat")))),
			Conj(Eq(s3, NewAtom("Blend")), Disj(Eq(p2, NewAtom("cat")), Eq(p4, NewAtom("cat")))),
			Conj(Eq(s4, NewAtom("Blend")), Disj(Eq(p3, NewAtom("cat")), Eq(p5, NewAtom("cat")))),
			Conj(Eq(s5, NewAtom("Blend")), Eq(p4, NewAtom("cat"))),
		),

		// Constraint 11: In a house next to the house with a horse, they smoke Dunhill
		Disj(
			Conj(Eq(p1, NewAtom("horse")), Eq(s2, NewAtom("Dunhill"))),
			Conj(Eq(p2, NewAtom("horse")), Disj(Eq(s1, NewAtom("Dunhill")), Eq(s3, NewAtom("Dunhill")))),
			Conj(Eq(p3, NewAtom("horse")), Disj(Eq(s2, NewAtom("Dunhill")), Eq(s4, NewAtom("Dunhill")))),
			Conj(Eq(p4, NewAtom("horse")), Disj(Eq(s3, NewAtom("Dunhill")), Eq(s5, NewAtom("Dunhill")))),
			Conj(Eq(p5, NewAtom("horse")), Eq(s4, NewAtom("Dunhill"))),
		),

		// Constraint 15: They drink water in a house next to the house where they smoke Blend
		Disj(
			Conj(Eq(d1, NewAtom("water")), Eq(s2, NewAtom("Blend"))),
			Conj(Eq(d2, NewAtom("water")), Disj(Eq(s1, NewAtom("Blend")), Eq(s3, NewAtom("Blend")))),
			Conj(Eq(d4, NewAtom("water")), Disj(Eq(s3, NewAtom("Blend")), Eq(s5, NewAtom("Blend")))),
			Conj(Eq(d5, NewAtom("water")), Eq(s4, NewAtom("Blend"))),
		),

		// Add distinctness constraints to force unique values
		allDiff(n1, n2, n3, n4, n5),
		allDiff(c1, c2, c3, c4, c5),
		allDiff(p1, p2, p3, p4, p5),
		allDiff(d1, d2, d3, d4, d5),
		allDiff(s1, s2, s3, s4, s5),

		// Ensure zebra is in one of the houses
		Disj(
			Eq(p1, NewAtom("zebra")),
			Eq(p2, NewAtom("zebra")),
			Eq(p3, NewAtom("zebra")),
			Eq(p4, NewAtom("zebra")),
			Eq(p5, NewAtom("zebra")),
		),
	)
}

// sameHouse ensures that if attr1 == val1 in some house, then attr2 == val2 in the same house
func sameHouse(a1_1, a2_1, val1, val2 Term, a1_2, a2_2, a1_3, a2_3, a1_4, a2_4, a1_5, a2_5 Term) Goal {
	return Disj(
		Conj(Eq(a1_1, val1), Eq(a2_1, val2)),
		Conj(Eq(a1_2, val1), Eq(a2_2, val2)),
		Conj(Eq(a1_3, val1), Eq(a2_3, val2)),
		Conj(Eq(a1_4, val1), Eq(a2_4, val2)),
		Conj(Eq(a1_5, val1), Eq(a2_5, val2)),
	)
}

// allDiff ensures all terms are different from each other
func allDiff(terms ...Term) Goal {
	var goals []Goal
	for i := 0; i < len(terms); i++ {
		for j := i + 1; j < len(terms); j++ {
			goals = append(goals, Neq(terms[i], terms[j]))
		}
	}
	return Conj(goals...)
}

// displaySolution pretty-prints the puzzle solution
func displaySolution(result Term) {
	pair, ok := result.(*Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	fmt.Println("House | Nationality | Color  | Pet    | Drink  | Smoke")
	fmt.Println("------|-------------|--------|--------|--------|-------------")

	var zebraOwner string
	position := 1

	for pair != nil {
		houseTerm := pair.Car()
		housePair, ok := houseTerm.(*Pair)
		if !ok {
			break
		}

		// Extract (nationality color pet drink smoke)
		nat := extractAtom(housePair.Car())
		housePair, _ = housePair.Cdr().(*Pair)
		if housePair == nil {
			break
		}

		col := extractAtom(housePair.Car())
		housePair, _ = housePair.Cdr().(*Pair)
		if housePair == nil {
			break
		}

		pet := extractAtom(housePair.Car())
		housePair, _ = housePair.Cdr().(*Pair)
		if housePair == nil {
			break
		}

		drink := extractAtom(housePair.Car())
		housePair, _ = housePair.Cdr().(*Pair)
		if housePair == nil {
			break
		}

		smoke := extractAtom(housePair.Car())

		fmt.Printf("  %d   | %-11s | %-6s | %-6s | %-6s | %s\n",
			position, nat, col, pet, drink, smoke)

		if pet == "zebra" {
			zebraOwner = nat
		}

		pair, _ = pair.Cdr().(*Pair)
		position++
	}

	fmt.Println()
	if zebraOwner != "" {
		fmt.Printf("🦓 Answer: The %s owns the zebra!\n", zebraOwner)
	}
}

// extractAtom extracts the string value from an Atom term
func extractAtom(term Term) string {
	if atom, ok := term.(*Atom); ok {
		return fmt.Sprintf("%v", atom.Value())
	}
	return "?"
}
