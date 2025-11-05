```go
func ExampleTabledRelation_transitiveClosureManual() {
	// Define edge relation
	edge, _ := DbRel("edge", 2, 0, 1)
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "d"},
	)

	// Create tabled edge predicate
	edgeTabled := TabledRelation(db, edge, "edge")

	// Manually define path using disjunction: path(X,Y) :- edge(X,Y) | (edge(X,Z), path(Z,Y))
	// For a proper recursive definition, we'd need fixpoint computation
	// Here we just show multi-hop manually
	x := Fresh("x")
	y := Fresh("y")
	z := Fresh("z")

	// Two-hop path: a->b->c or b->c->d
	twoHop := Conj(
		edgeTabled(x, z),
		edgeTabled(z, y),
	)

	// Bind x to "a" to find paths from a
	goal := Conj(
		Eq(x, NewAtom("a")),
		twoHop,
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	// Should find a->c (via b)
	if len(results) > 0 {
		binding := results[0].GetBinding(y.ID())
		if atom, ok := binding.(*Atom); ok {
			fmt.Printf("a reaches %s in 2 hops\n", atom.String())
		}
	}

	// Output:
	// a reaches c in 2 hops
}

```


