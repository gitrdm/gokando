package minikanren_test

import (
	"context"
	"fmt"
	"strings"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

func ExampleSolutions_basic() {
	q := Fresh("q")
	goal := Disj(Eq(q, NewAtom(1)), Eq(q, NewAtom(2)))
	out := FormatSolutions(Solutions(goal, q))
	fmt.Println(strings.Join(out, "\n"))
	// Output:
	// q: 1
	// q: 2
}

func ExampleModel_helpers_allDifferent() {
	m := NewModel()
	xs := m.IntVars(3, 1, 3, "x")
	_ = m.AllDifferent(xs...)

	sols, err := SolveN(context.Background(), m, 0)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(sols))
	// Output:
	// 6
}
