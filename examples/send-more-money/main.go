// Package sendmoremoney solves the classic "SEND + MORE = MONEY" cryptarithm puzzle.
//
// A cryptarithm is a mathematical puzzle where letters represent digits,
// and the goal is to find digits for each letter that make the equation true.
//
// The puzzle:    SEND
//   - MORE
//     ------
//     = MONEY
//
// Each letter represents a unique digit (0-9), and S and M cannot be zero.
//
// This example demonstrates:
// - True relational arithmetic constraints (Phase 7) for declarative cryptarithm solving
// - FD solver integration with arithmetic relations
// - Unified constraint solving with declarative arithmetic programming
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gitrdm/gokando/pkg/minikanren"
)

// sendMoreMoneyRelational solves the SEND + MORE = MONEY cryptarithm
// using relational arithmetic constraints (Phase 7)
func sendMoreMoneyRelational(result minikanren.Term) minikanren.Goal {
	return func(ctx context.Context, store minikanren.ConstraintStore) minikanren.ResultStream {
		// Create variables for each letter
		s := minikanren.Fresh("S")
		e := minikanren.Fresh("E")
		n := minikanren.Fresh("N")
		d := minikanren.Fresh("D")
		m := minikanren.Fresh("M")
		o := minikanren.Fresh("O")
		r := minikanren.Fresh("R")
		y := minikanren.Fresh("Y")

		// All letters must be different digits (0-9)
		allLetters := []*minikanren.Var{s, e, n, d, m, o, r, y}

		// Build SEND = 1000*S + 100*E + 10*N + D
		sendS := minikanren.Fresh("sendS")     // 1000*S
		sendE := minikanren.Fresh("sendE")     // 100*E
		sendN := minikanren.Fresh("sendN")     // 10*N
		sendSE := minikanren.Fresh("sendSE")   // 1000*S + 100*E
		sendSEN := minikanren.Fresh("sendSEN") // 1000*S + 100*E + 10*N
		sendValue := minikanren.Fresh("send")

		// Build MORE = 1000*M + 100*O + 10*R + E
		moreM := minikanren.Fresh("moreM")     // 1000*M
		moreO := minikanren.Fresh("moreO")     // 100*O
		moreR := minikanren.Fresh("moreR")     // 10*R
		moreMO := minikanren.Fresh("moreMO")   // 1000*M + 100*O
		moreMOR := minikanren.Fresh("moreMOR") // 1000*M + 100*O + 10*R
		moreValue := minikanren.Fresh("more")

		// Build MONEY = 10000*M + 1000*O + 100*N + 10*E + Y
		moneyM := minikanren.Fresh("moneyM")       // 10000*M
		moneyO := minikanren.Fresh("moneyO")       // 1000*O
		moneyN := minikanren.Fresh("moneyN")       // 100*N
		moneyE := minikanren.Fresh("moneyE")       // 10*E
		moneyMO := minikanren.Fresh("moneyMO")     // 10000*M + 1000*O
		moneyMON := minikanren.Fresh("moneyMON")   // 10000*M + 1000*O + 100*N
		moneyMONE := minikanren.Fresh("moneyMONE") // 10000*M + 1000*O + 100*N + 10*E
		moneyValue := minikanren.Fresh("money")

		return minikanren.FDSolve(minikanren.Conj(
			// All different digits
			minikanren.FDAllDifferent(allLetters...),

			// S and M cannot be zero (leading digits)
			minikanren.FDIn(s, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			minikanren.FDIn(m, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}),

			// Build SEND value
			minikanren.FDMultiply(minikanren.NewAtom(1000), s, sendS),
			minikanren.FDMultiply(minikanren.NewAtom(100), e, sendE),
			minikanren.FDMultiply(minikanren.NewAtom(10), n, sendN),
			minikanren.FDPlus(sendS, sendE, sendSE),
			minikanren.FDPlus(sendSE, sendN, sendSEN),
			minikanren.FDPlus(sendSEN, d, sendValue),

			// Build MORE value
			minikanren.FDMultiply(minikanren.NewAtom(1000), m, moreM),
			minikanren.FDMultiply(minikanren.NewAtom(100), o, moreO),
			minikanren.FDMultiply(minikanren.NewAtom(10), r, moreR),
			minikanren.FDPlus(moreM, moreO, moreMO),
			minikanren.FDPlus(moreMO, moreR, moreMOR),
			minikanren.FDPlus(moreMOR, e, moreValue),

			// Build MONEY value
			minikanren.FDMultiply(minikanren.NewAtom(10000), m, moneyM),
			minikanren.FDMultiply(minikanren.NewAtom(1000), o, moneyO),
			minikanren.FDMultiply(minikanren.NewAtom(100), n, moneyN),
			minikanren.FDMultiply(minikanren.NewAtom(10), e, moneyE),
			minikanren.FDPlus(moneyM, moneyO, moneyMO),
			minikanren.FDPlus(moneyMO, moneyN, moneyMON),
			minikanren.FDPlus(moneyMON, moneyE, moneyMONE),
			minikanren.FDPlus(moneyMONE, y, moneyValue),

			// The main constraint: SEND + MORE = MONEY
			minikanren.FDPlus(sendValue, moreValue, moneyValue),

			// Return the solution as (S E N D M O R Y)
			minikanren.Eq(result, minikanren.List(s, e, n, d, m, o, r, y)),
		))(ctx, store)
	}
}

func main() {
	fmt.Println("=== Solving SEND + MORE = MONEY Cryptarithm ===")
	fmt.Println()
	fmt.Println("Finding digits where each letter represents a unique digit (0-9)")
	fmt.Println("and S and M cannot be zero (leading digits)...")
	fmt.Println()

	// Use relational arithmetic to solve the cryptarithm
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results := minikanren.RunWithContext(ctx, 1, func(q *minikanren.Var) minikanren.Goal {
		return sendMoreMoneyRelational(q)
	})

	if len(results) == 0 {
		fmt.Printf("❌ No solutions found\n")
		fmt.Println()
		fmt.Println("This may indicate the search space is too large or there's an issue with the constraints.")
		return
	}

	fmt.Printf("✅ Found %d solution(s) using relational FD arithmetic!\n\n", len(results))

	// Display the solution
	for i, result := range results {
		if len(results) > 1 {
			fmt.Printf("Solution %d:\n", i+1)
		}
		displaySimpleResult(result)
		fmt.Println()
	}

	fmt.Println("🎉 Success! This demonstrates Phase 7 relational arithmetic:")
	fmt.Println("- ✅ FDPlus, FDMultiply work relationally to build complex expressions")
	fmt.Println("- ✅ FDAllDifferent ensures all digits are unique")
	fmt.Println("- ✅ Cryptarithm solved declaratively without manual search")
}

// displaySimpleResult shows the SEND + MORE = MONEY solution
func displaySimpleResult(result minikanren.Term) {
	// Extract the values from miniKanren list: (S E N D M O R Y)
	pair, ok := result.(*minikanren.Pair)
	if !ok {
		fmt.Println("Invalid result format")
		return
	}

	// Extract the 8 digit values
	var values []int
	idx := 0
	for pair != nil && idx < 8 {
		if atom, ok := pair.Car().(*minikanren.Atom); ok {
			if val, ok := atom.Value().(int); ok {
				values = append(values, val)
			}
		}

		if next, ok := pair.Cdr().(*minikanren.Pair); ok {
			pair = next
		} else {
			break
		}
		idx++
	}

	if len(values) == 8 {
		s, e, n, d, m, o, r, y := values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7]

		send := 1000*s + 100*e + 10*n + d
		more := 1000*m + 100*o + 10*r + e
		money := 10000*m + 1000*o + 100*n + 10*e + y

		fmt.Printf("Solution found!\n")
		fmt.Printf("  S=%d E=%d N=%d D=%d M=%d O=%d R=%d Y=%d\n\n", s, e, n, d, m, o, r, y)
		fmt.Printf("    SEND  = %d\n", send)
		fmt.Printf("  + MORE  = %d\n", more)
		fmt.Printf("  ------\n")
		fmt.Printf("  = MONEY = %d\n", money)
		fmt.Printf("\n")

		// Verify the solution
		if send+more == money {
			fmt.Printf("✓ Verification: %d + %d = %d (correct!)\n", send, more, money)
		} else {
			fmt.Printf("✗ Verification failed: %d + %d = %d (expected %d)\n", send, more, send+more, money)
		}
	} else {
		fmt.Printf("Could not extract all 8 letter values (got %d)\n", len(values))
	}
}
