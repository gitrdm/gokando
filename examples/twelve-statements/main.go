// Package main solves the Twelve Statements puzzle using GoKando's FD solver
// with production global constraints (BoolSum, reification, and Table) — no
// hybrid workaround.
//
// The puzzle: Given twelve statements about themselves, determine which are true.
//
//  1. This is a numbered list of twelve statements.
//  2. Exactly 3 of the last 6 statements are true.
//  3. Exactly 2 of the even-numbered statements are true.
//  4. If statement 5 is true, then statements 6 and 7 are both true.
//  5. The 3 preceding statements are all false.
//  6. Exactly 4 of the odd-numbered statements are true.
//  7. Either statement 2 or 3 is true, but not both.
//  8. If statement 7 is true, then 5 and 6 are both true.
//  9. Exactly 3 of the first 6 statements are true.
//
// 10. The next two statements are both true.
// 11. Exactly 1 of statements 7, 8 and 9 are true.
// 12. Exactly 4 of the preceding statements are true.
//
// Modeling approach (FD only):
//   - Each statement Si is a boolean FD variable with domain {1=false, 2=true}
//   - Counting statements use BoolSum over the relevant Si, with a count variable Ki
//     in [1..n+1] (encoding count+1), then reify (Ki == expected+1) with Si via
//     NewValueEqualsReified to enforce Si ↔ (count == expected).
//   - Logical forms (implication, XOR, conjunction) are encoded as small Table
//     constraints over {1,2} tuples mapping inputs to the resulting Si value.
//
// This is fast (tiny search space) and demonstrates idiomatic use of the solver.
package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== Solving the Twelve Statements Puzzle ===")
	fmt.Println()

	model := mk.NewModel()

	// Boolean domain: {1=false, 2=true}
	bdom := mk.NewBitSetDomain(2)

	// Create statement booleans S1..S12
	S := make([]*mk.FDVariable, 12)
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("S%d", i+1)
		// S1 is always true — fix to {2}
		if i == 0 {
			S[i] = model.NewVariableWithName(mk.NewBitSetDomainFromValues(2, []int{2}), name)
		} else {
			S[i] = model.NewVariableWithName(bdom, name)
		}
	}

	// Helper: enforce Si ↔ (exactly exp of vars are true)
	postExactly := func(vars []*mk.FDVariable, exp int, stmt *mk.FDVariable) {
		total := model.NewVariableWithName(mk.NewBitSetDomain(len(vars)+1), fmt.Sprintf("K_%d", stmt.ID()))
		sum, _ := mk.NewBoolSum(vars, total)
		model.AddConstraint(sum)
		// Reify (total == exp+1) with stmt
		reif, _ := mk.NewValueEqualsReified(total, exp+1, stmt)
		model.AddConstraint(reif)
	}

	// Helper: build a small boolean table in {1=false,2=true}
	encode := func(b bool) int {
		if b {
			return 2
		}
		return 1
	}

	// S2: Exactly 3 of the last 6 (S7..S12)
	postExactly([]*mk.FDVariable{S[6], S[7], S[8], S[9], S[10], S[11]}, 3, S[1])

	// S3: Exactly 2 of the even-numbered statements (S2,S4,S6,S8,S10,S12)
	postExactly([]*mk.FDVariable{S[1], S[3], S[5], S[7], S[9], S[11]}, 2, S[2])

	// S6: Exactly 4 of the odd-numbered statements (S1,S3,S5,S7,S9,S11)
	postExactly([]*mk.FDVariable{S[0], S[2], S[4], S[6], S[8], S[10]}, 4, S[5])

	// S9: Exactly 3 of the first 6 statements (S1..S6)
	postExactly([]*mk.FDVariable{S[0], S[1], S[2], S[3], S[4], S[5]}, 3, S[8])

	// S12: Exactly 4 of the preceding statements (S1..S11)
	postExactly([]*mk.FDVariable{S[0], S[1], S[2], S[3], S[4], S[5], S[6], S[7], S[8], S[9], S[10]}, 4, S[11])

	// Logical statements via small tables:
	// S4: (S5 -> (S6 ∧ S7))
	{
		rows := make([][]int, 0, 8)
		for _, v5 := range []int{1, 2} {
			for _, v6 := range []int{1, 2} {
				for _, v7 := range []int{1, 2} {
					p := (v5 == 2) // S5 true
					q := (v6 == 2) && (v7 == 2)
					val := encode(!p || q)
					rows = append(rows, []int{v5, v6, v7, val})
				}
			}
		}
		t, _ := mk.NewTable([]*mk.FDVariable{S[4], S[5], S[6], S[3]}, rows) // S5,S6,S7 → S4
		model.AddConstraint(t)
	}

	// S5: (¬S2 ∧ ¬S3 ∧ ¬S4)
	{
		rows := make([][]int, 0, 8)
		for _, v2 := range []int{1, 2} {
			for _, v3 := range []int{1, 2} {
				for _, v4 := range []int{1, 2} {
					val := encode(v2 == 1 && v3 == 1 && v4 == 1)
					rows = append(rows, []int{v2, v3, v4, val})
				}
			}
		}
		t, _ := mk.NewTable([]*mk.FDVariable{S[1], S[2], S[3], S[4]}, rows) // S2,S3,S4 → S5
		model.AddConstraint(t)
	}

	// S7: XOR(S2, S3)
	{
		rows := make([][]int, 0, 4)
		for _, v2 := range []int{1, 2} {
			for _, v3 := range []int{1, 2} {
				val := encode((v2 == 2) != (v3 == 2))
				rows = append(rows, []int{v2, v3, val})
			}
		}
		t, _ := mk.NewTable([]*mk.FDVariable{S[1], S[2], S[6]}, rows) // S2,S3 → S7
		model.AddConstraint(t)
	}

	// S8: (¬S7 ∨ (S5 ∧ S6))
	{
		rows := make([][]int, 0, 8)
		for _, v7 := range []int{1, 2} {
			for _, v5 := range []int{1, 2} {
				for _, v6 := range []int{1, 2} {
					val := encode((v7 == 1) || ((v5 == 2) && (v6 == 2)))
					rows = append(rows, []int{v7, v5, v6, val})
				}
			}
		}
		t, _ := mk.NewTable([]*mk.FDVariable{S[6], S[4], S[5], S[7]}, rows) // S7,S5,S6 → S8
		model.AddConstraint(t)
	}

	// S10: (S11 ∧ S12)
	{
		rows := make([][]int, 0, 4)
		for _, v11 := range []int{1, 2} {
			for _, v12 := range []int{1, 2} {
				val := encode((v11 == 2) && (v12 == 2))
				rows = append(rows, []int{v11, v12, val})
			}
		}
		t, _ := mk.NewTable([]*mk.FDVariable{S[10], S[11], S[9]}, rows) // S11,S12 → S10
		model.AddConstraint(t)
	}

	// S11: Exactly 1 of (S7,S8,S9)
	postExactly([]*mk.FDVariable{S[6], S[7], S[8]}, 1, S[10])

	// Solve for all solutions
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sols, _ := solver.Solve(ctx, 0) // 0 → enumerate all

	if len(sols) == 0 {
		fmt.Println("❌ No solution found!")
		return
	}

	fmt.Printf("✓ Found %d solution(s)!\n\n", len(sols))
	for i, sol := range sols {
		if len(sols) > 1 {
			fmt.Printf("Solution %d:\n", i+1)
		}
		displaySolutionFD(S, sol)
		if i < len(sols)-1 {
			fmt.Println()
		}
	}
}

// displaySolutionFD pretty-prints the solution using FD assignments
func displaySolutionFD(S []*mk.FDVariable, sol []int) {
	statements := []string{
		"This is a numbered list of twelve statements.",
		"Exactly 3 of the last 6 statements are true.",
		"Exactly 2 of the even-numbered statements are true.",
		"If statement 5 is true, then statements 6 and 7 are both true.",
		"The 3 preceding statements are all false.",
		"Exactly 4 of the odd-numbered statements are true.",
		"Either statement 2 or 3 is true, but not both.",
		"If statement 7 is true, then 5 and 6 are both true.",
		"Exactly 3 of the first 6 statements are true.",
		"The next two statements are both true.",
		"Exactly 1 of statements 7, 8 and 9 are true.",
		"Exactly 4 of the preceding statements are true.",
	}

	var trueStatements []int
	var falseStatements []int
	for i := 0; i < 12; i++ {
		v := sol[S[i].ID()]
		if v == 2 { // true
			trueStatements = append(trueStatements, i+1)
		} else {
			falseStatements = append(falseStatements, i+1)
		}
	}

	fmt.Println("TRUE statements:")
	for _, n := range trueStatements {
		fmt.Printf("%2d. %s\n", n, statements[n-1])
	}

	fmt.Println("\nFALSE statements:")
	for _, n := range falseStatements {
		fmt.Printf("%2d. %s\n", n, statements[n-1])
	}

	fmt.Printf("\n✅ %d true, %d false\n", len(trueStatements), len(falseStatements))
}
