```go
func ExampleDatabase_Query_datalog() {
	edge, _ := DbRel("edge", 2, 0, 1)
	// Build a graph: a -> b -> c
	//                ^-------|
	db := DB().MustAddFacts(edge,
		[]interface{}{"a", "b"},
		[]interface{}{"b", "c"},
		[]interface{}{"c", "a"},
	)

	// Query: Find all nodes reachable from 'a' in exactly 2 hops
	// path2(X, Z) :- edge(X, Y), edge(Y, Z)
	start := NewAtom("a")
	middle := Fresh("middle")
	dest := Fresh("destination")

	goal := Conj(
		db.Query(edge, start, middle),
		db.Query(edge, middle, dest),
	)

	ctx := context.Background()
	store := NewLocalConstraintStore(NewGlobalConstraintBus())
	stream := goal(ctx, store)
	results, _ := stream.Take(10)

	fmt.Printf("Nodes reachable from 'a' in 2 hops:\n")
	for _, r := range results {
		val := r.GetBinding(dest.ID())
		if atom, ok := val.(*Atom); ok {
			fmt.Printf("  %v\n", atom.Value())
		}
	}

	// Output:
	// Nodes reachable from 'a' in 2 hops:
	//   c
}

```


