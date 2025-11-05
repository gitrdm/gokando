package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokanlogic/pkg/minikanren"
)

// Example_hlapi_path_twoHop demonstrates a small, tabled two-hop path pattern
// with HLAPI helpers. It shows how tabling composes in joins with minimal boilerplate.
func Example_hlapi_path_twoHop() {
	edge := MustRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	x := Fresh("x")
	z := Fresh("z")
	y := Fresh("y")

	// twoHop(X, Y) :- edge(X, Z), edge(Z, Y)
	goal := Conj(
		Eq(x, NewAtom("a")),
		TQ(db, edge, x, z),
		TQ(db, edge, z, y),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

// Example_hlapi_grandparent converts the tabled grandparent example to HLAPI style.
func Example_hlapi_grandparent() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
	)

	gp := Fresh("gp")
	p := Fresh("p")
	gc := Fresh("gc")

	goal := Conj(
		TQ(db, parent, gp, p),
		TQ(db, parent, p, gc),
		Eq(gp, NewAtom("john")),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

// Example_hlapi_multiRelationLoader shows the map-based multi-relation loader with HLAPI queries.
func Example_hlapi_multiRelationLoader() {
	emp, mgr := MustRel("employee", 2, 0, 1), MustRel("manager", 2, 0, 1)
	rels := map[string]*Relation{"employee": emp, "manager": mgr}
	data := map[string][][]interface{}{
		"employee": {{"alice", "eng"}, {"bob", "eng"}},
		"manager":  {{"bob", "alice"}},
	}
	// Load both relations in one pass
	db, _ := NewDBFromMap(rels, data)

	mgrVar := Fresh("mgr")
	goal := TQ(db, mgr, mgrVar, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	fmt.Println(len(rows))
	// Output:
	// 1
}

// Example_hlapi_ancestor_recursive demonstrates a true recursive, tabled predicate
// using TabledRecursivePredicate and HLAPI helpers.
func Example_hlapi_ancestor_recursive() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// Define ancestor(X,Y): parent(X,Y) OR (parent(X,Z) AND ancestor(Z,Y))
	ancestor := TabledRecursivePredicate(db, parent, "ancestor",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Q(parent, x, z),
				self(z, y),
			)
		},
	)

	x := Fresh("x")
	y := Fresh("y")

	goal := Conj(
		Eq(y, NewAtom("alice")),
		ancestor(x, y),
	)

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	// We just assert we found two rows to keep the example stable.
	fmt.Println(len(rows))
	// Output:
	// 2
}
