package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// Example_hlapi_disjq demonstrates using DisjQ to OR multiple pldb query patterns.
func Example_hlapi_disjq() {
	rel := MustRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(rel,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	x := Fresh("x")
	y := Fresh("y")

	// Either edge(x, y) or edge(y, x)
	goal := DisjQ(db, rel, []interface{}{x, y}, []interface{}{y, x})

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(100)
	// Both disjuncts contribute: edge(x,y) yields 3 rows, and edge(y,x)
	// yields 3 rows with swapped bindings, for a total of 6.
	fmt.Println(len(rows))
	// Output:
	// 6
}
