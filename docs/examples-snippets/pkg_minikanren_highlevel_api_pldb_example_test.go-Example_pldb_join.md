```go
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

```


