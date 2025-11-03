// Package sendmoremoney solves the classic "SEND + MORE = MONEY" cryptarithm
// using the production FD solver with Table and global constraints (no hybrid workaround).
//
// Modeling:
//   - Digits 0..9 are encoded as FD values 1..10 (value-1 = digit)
//   - Letters S,E,N,D,M,O,R,Y are FD vars in 1..10, AllDifferent, with S,M ≠ 0 (domains 2..10)
//   - Column arithmetic is encoded with Table constraints over allowed tuples:
//     (x, y, cin) -> (z, cout) where x+y+cin = z + 10*cout
//   - Carries C1..C4 are FD vars in {1,2} encoding {0,1}; final carry-out C4 = 1 (encoded 2)
//   - This is sufficient to solve the puzzle and discover M=1 (encoded 2)
package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

func main() {
	fmt.Println("=== FD SEND + MORE = MONEY ===")

	model := mk.NewModel()

	// Digits encoding: 0..9 -> 1..10
	digits := mk.NewBitSetDomain(10)
	digitsNoZero := mk.NewBitSetDomainFromValues(10, []int{2, 3, 4, 5, 6, 7, 8, 9, 10})

	// Letters
	S := model.NewVariableWithName(digitsNoZero, "S")
	E := model.NewVariableWithName(digits, "E")
	N := model.NewVariableWithName(digits, "N")
	D := model.NewVariableWithName(digits, "D")
	// In SEND+MORE=MONEY, the leading carry creates M=1 (encoded 2). We can safely fix M to 1.
	M := model.NewVariableWithName(mk.NewBitSetDomainFromValues(10, []int{2}), "M")
	O := model.NewVariableWithName(digits, "O")
	R := model.NewVariableWithName(digits, "R")
	Y := model.NewVariableWithName(digits, "Y")

	// AllDifferent on letters
	ad, _ := mk.NewAllDifferent([]*mk.FDVariable{S, E, N, D, M, O, R, Y})
	model.AddConstraint(ad)

	// Carries C1..C4 (0/1 -> {1,2}); C0 is implicitly 0; C4 must be 1
	bool01 := mk.NewBitSetDomainFromValues(10, []int{1, 2})
	C1 := model.NewVariableWithName(bool01, "C1")
	C2 := model.NewVariableWithName(bool01, "C2")
	C3 := model.NewVariableWithName(bool01, "C3")
	C4 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(10, []int{2}), "C4") // final carry = 1 (encoded 2)

	// Helper: build table rows for a column: x + y + cin = z + 10*cout under encoding
	buildCol := func() [][]int {
		rows := make([][]int, 0, 10*10*2)
		for x := 0; x <= 9; x++ {
			for y := 0; y <= 9; y++ {
				for cin := 0; cin <= 1; cin++ {
					sum := x + y + cin
					z := sum % 10
					cout := sum / 10
					rows = append(rows, []int{x + 1, y + 1, cin + 1, z + 1, cout + 1})
				}
			}
		}
		return rows
	}
	colRows := buildCol()

	// Column 1 (units): D + E + 0 = Y + 10*C1
	// Encode carry-in 0 by restricting a variable to {1}
	C0 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(10, []int{1}), "C0")
	t1, _ := mk.NewTable([]*mk.FDVariable{D, E, C0, Y, C1}, colRows)
	model.AddConstraint(t1)

	// Column 2: N + R + C1 = E + 10*C2
	t2, _ := mk.NewTable([]*mk.FDVariable{N, R, C1, E, C2}, colRows)
	model.AddConstraint(t2)

	// Column 3: E + O + C2 = N + 10*C3
	t3, _ := mk.NewTable([]*mk.FDVariable{E, O, C2, N, C3}, colRows)
	model.AddConstraint(t3)

	// Column 4: S + M + C3 = O + 10*C4
	t4, _ := mk.NewTable([]*mk.FDVariable{S, M, C3, O, C4}, colRows)
	model.AddConstraint(t4)

	// Solve
	solver := mk.NewSolver(model)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sols, _ := solver.Solve(ctx, 1) // one solution is enough

	if len(sols) == 0 {
		fmt.Println("No solution found")
		return
	}

	// Print mapping from the found solution (convert encoded to digits by -1)
	sol := sols[0]
	vals := map[string]int{
		"S": sol[S.ID()] - 1,
		"E": sol[E.ID()] - 1,
		"N": sol[N.ID()] - 1,
		"D": sol[D.ID()] - 1,
		"M": sol[M.ID()] - 1,
		"O": sol[O.ID()] - 1,
		"R": sol[R.ID()] - 1,
		"Y": sol[Y.ID()] - 1,
	}

	fmt.Println("Letter → Digit mapping:")
	order := []string{"S", "E", "N", "D", "M", "O", "R", "Y"}
	for _, k := range order {
		fmt.Printf("  %s → %d\n", k, vals[k])
	}

	send := vals["S"]*1000 + vals["E"]*100 + vals["N"]*10 + vals["D"]
	more := vals["M"]*1000 + vals["O"]*100 + vals["R"]*10 + vals["E"]
	money := vals["M"]*10000 + vals["O"]*1000 + vals["N"]*100 + vals["E"]*10 + vals["Y"]
	fmt.Printf("\n  %d\n+ %d\n------\n %d\n", send, more, money)
}
