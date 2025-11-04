package minikanren_test

import (
	"context"
	"fmt"

	. "github.com/gitrdm/gokando/pkg/minikanren"
)

// Example_hlapi_ancestor_recursive_sugar shows the HLAPI wrapper RecursiveTablePred
// for defining a true recursive, tabled predicate that accepts native values at call sites.
func Example_hlapi_ancestor_recursive_sugar() {
	parent := MustRel("parent", 2, 0, 1)
	db := DB().MustAddFacts(parent,
		[]interface{}{"john", "mary"},
		[]interface{}{"mary", "alice"},
		[]interface{}{"john", "tom"},
		[]interface{}{"tom", "bob"},
	)

	// ancestor2(X,Y) :- parent(X,Y).
	// ancestor2(X,Y) :- parent(X,Z), ancestor2(Z,Y).
	ancestor2 := RecursiveTablePred(db, parent, "ancestor2",
		func(self func(...Term) Goal, args ...Term) Goal {
			x, y := args[0], args[1]
			z := Fresh("z")
			return Conj(
				db.Query(parent, x, z),
				self(z, y),
			)
		})

	x := Fresh("x")

	// Mix Terms and native values at call sites
	goal := ancestor2(x, "alice")

	ctx := context.Background()
	stores := goal(ctx, NewLocalConstraintStore(NewGlobalConstraintBus()))
	rows, _ := stores.Take(10)
	// john -> mary -> alice, so both john and mary are ancestors of alice
	fmt.Println(len(rows))
	// Output:
	// 2
}

// Example_hlapi_values_projection demonstrates projecting typed values
// from Solutions using ValuesInt and AsInt helpers.
func Example_hlapi_values_projection() {
	x := Fresh("x")
	goal := Disj(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
	sols := Solutions(goal, x)
	ints := ValuesInt(sols, "x")
	// Print count and sum to avoid relying on order
	sum := 0
	for _, v := range ints {
		sum += v
	}
	fmt.Printf("%d %d\n", len(ints), sum)
	// Output:
	// 2 3
}
