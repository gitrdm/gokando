package main

import (
	"context"
	"fmt"
	"time"

	mk "github.com/gitrdm/gokando/pkg/minikanren"
)

// A tiny demo of lexicographic ordering X <=lex Y.
func main() {
	fmt.Println("=== Lexicographic Constraint Demo (X ≤lex Y) ===")

	model := mk.NewModel()

	x1 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(9, []int{2, 3, 4}), "x1")
	x2 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(9, []int{1, 2, 3}), "x2")
	y1 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(9, []int{3, 4, 5}), "y1")
	y2 := model.NewVariableWithName(mk.NewBitSetDomainFromValues(9, []int{2, 3, 4}), "y2")

	c, err := mk.NewLexLessEq([]*mk.FDVariable{x1, x2}, []*mk.FDVariable{y1, y2})
	if err != nil {
		panic(err)
	}
	model.AddConstraint(c)

	solver := mk.NewSolver(model)
	// Run a single propagation cycle by asking for zero solutions
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _ = solver.Solve(ctx, 0)

	fmt.Println("x1:", solver.GetDomain(nil, x1.ID()))
	fmt.Println("x2:", solver.GetDomain(nil, x2.ID()))
	fmt.Println("y1:", solver.GetDomain(nil, y1.ID()))
	fmt.Println("y2:", solver.GetDomain(nil, y2.ID()))
}
