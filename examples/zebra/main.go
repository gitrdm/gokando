// Package main solves the famous Zebra puzzle (Einstein's Riddle) using gokanlogic.
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

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Zebra Puzzle with gokanlogic ===")
	fmt.Println()

	// Solve the puzzle - we expect exactly one solution
	results := Run(1, zebraPuzzle)

	if len(results) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Println("✓ Solution found!")
	fmt.Println()

	displaySolution(results[0])
}

// zebraPuzzle defines the complete puzzle as a miniKanren goal.
func zebraPuzzle(q *Var) Goal {
	// Define variables for each house's attributes
	// Attribute indices: 0=nationality, 1=color, 2=pet, 3=drink, 4=smoke
	houses := [][]Term{
		{Fresh("n1"), Fresh("c1"), Fresh("p1"), Fresh("d1"), Fresh("s1")},
		{Fresh("n2"), Fresh("c2"), Fresh("p2"), Fresh("d2"), Fresh("s2")},
		{Fresh("n3"), Fresh("c3"), Fresh("p3"), Fresh("d3"), Fresh("s3")},
		{Fresh("n4"), Fresh("c4"), Fresh("p4"), Fresh("d4"), Fresh("s4")},
		{Fresh("n5"), Fresh("c5"), Fresh("p5"), Fresh("d5"), Fresh("s5")},
	}

	const (
		NAT   = 0 // Nationality
		COLOR = 1
		PET   = 2
		DRINK = 3
		SMOKE = 4
	)

	// Helper: check if attribute matches value in some house
	member := func(attrIdx int, val Term) Goal {
		goals := make([]Goal, len(houses))
		for i, h := range houses {
			goals[i] = Eq(h[attrIdx], val)
		}
		return Disj(goals...)
	}

	// Helper: two attributes in same house
	sameHouse := func(attr1Idx int, val1 Term, attr2Idx int, val2 Term) Goal {
		goals := make([]Goal, len(houses))
		for i, h := range houses {
			goals[i] = Conj(Eq(h[attr1Idx], val1), Eq(h[attr2Idx], val2))
		}
		return Disj(goals...)
	}

	// Helper: two attributes in adjacent houses
	adjacent := func(attr1Idx int, val1 Term, attr2Idx int, val2 Term) Goal {
		var goals []Goal
		for i := 0; i < len(houses)-1; i++ {
			// val1 in house i, val2 in house i+1
			goals = append(goals, Conj(Eq(houses[i][attr1Idx], val1), Eq(houses[i+1][attr2Idx], val2)))
			// val1 in house i+1, val2 in house i
			goals = append(goals, Conj(Eq(houses[i+1][attr1Idx], val1), Eq(houses[i][attr2Idx], val2)))
		}
		return Disj(goals...)
	}

	// Helper: attribute1 immediately left of attribute2
	leftOf := func(attr1Idx int, val1 Term, attr2Idx int, val2 Term) Goal {
		goals := make([]Goal, len(houses)-1)
		for i := 0; i < len(houses)-1; i++ {
			goals[i] = Conj(Eq(houses[i][attr1Idx], val1), Eq(houses[i+1][attr2Idx], val2))
		}
		return Disj(goals...)
	}

	// Helper: ensure all values in attribute column are distinct
	allDiff := func(attrIdx int) Goal {
		var goals []Goal
		for i := 0; i < len(houses); i++ {
			for j := i + 1; j < len(houses); j++ {
				goals = append(goals, Neq(houses[i][attrIdx], houses[j][attrIdx]))
			}
		}
		return Conj(goals...)
	}

	// Create solution structure
	solution := List(
		List(houses[0]...),
		List(houses[1]...),
		List(houses[2]...),
		List(houses[3]...),
		List(houses[4]...),
	)

	return Conj(
		Eq(q, solution),

		// Constraint 9: Norwegian in first house
		Eq(houses[0][NAT], NewAtom("Norwegian")),

		// Constraint 8: Milk in middle house
		Eq(houses[2][DRINK], NewAtom("milk")),

		// Constraint 14: Norwegian next to blue house (Norwegian in house 0, so blue in house 1)
		Eq(houses[1][COLOR], NewAtom("blue")),

		// Constraint 1: English in red house
		sameHouse(NAT, NewAtom("English"), COLOR, NewAtom("red")),

		// Constraint 2: Swede has dog
		sameHouse(NAT, NewAtom("Swede"), PET, NewAtom("dog")),

		// Constraint 3: Dane drinks tea
		sameHouse(NAT, NewAtom("Dane"), DRINK, NewAtom("tea")),

		// Constraint 4: Green immediately left of white
		leftOf(COLOR, NewAtom("green"), COLOR, NewAtom("white")),

		// Constraint 5: Coffee in green house
		sameHouse(COLOR, NewAtom("green"), DRINK, NewAtom("coffee")),

		// Constraint 6: Pall Mall smoker has bird
		sameHouse(SMOKE, NewAtom("Pall Mall"), PET, NewAtom("bird")),

		// Constraint 7: Yellow house has Dunhill
		sameHouse(COLOR, NewAtom("yellow"), SMOKE, NewAtom("Dunhill")),

		// Constraint 10: Blend smoker next to cat owner
		adjacent(SMOKE, NewAtom("Blend"), PET, NewAtom("cat")),

		// Constraint 11: Dunhill smoker next to horse owner
		adjacent(SMOKE, NewAtom("Dunhill"), PET, NewAtom("horse")),

		// Constraint 12: Blue Master smoker drinks beer
		sameHouse(SMOKE, NewAtom("Blue Master"), DRINK, NewAtom("beer")),

		// Constraint 13: German smokes Prince
		sameHouse(NAT, NewAtom("German"), SMOKE, NewAtom("Prince")),

		// Constraint 15: Water drinker next to Blend smoker
		adjacent(DRINK, NewAtom("water"), SMOKE, NewAtom("Blend")),

		// All attributes must be distinct
		allDiff(NAT),
		allDiff(COLOR),
		allDiff(PET),
		allDiff(DRINK),
		allDiff(SMOKE),

		// Someone owns the zebra
		member(PET, NewAtom("zebra")),
	)
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
