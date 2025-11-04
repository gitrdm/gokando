package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// Example_pldb_join shows compact pldb usage with HLAPI helpers.
func Example_pldb_join() {
	parent := MustRel("parent", 2, 0, 1)

	db := DB().MustAddFacts(parent,
		[]interface{}{"alice", "bob"},
		[]interface{}{"bob", "charlie"},
		[]interface{}{"charlie", "diana"},
	)

	gp := Fresh("gp")
	gc := Fresh("gc")
	p := Fresh("p")

	// grandparent(GP, GC) :- parent(GP, P), parent(P, GC)
	goal := Conj(
		db.Q(parent, gp, p),
		db.Q(parent, p, gc),
	)

	// Count results for a stable example output
	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 2
}

// Example_tabled_query shows a tabled query using native values.
func Example_tabled_query() {
	edge := MustRel("edge", 2, 0, 1)
	// a -> b, b -> c
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
	)

	x := Fresh("x")
	y := Fresh("y")

	// TQ uses rel.Name() as predicate id and caches answers
	goal := TQ(db, edge, x, y)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 2
}
