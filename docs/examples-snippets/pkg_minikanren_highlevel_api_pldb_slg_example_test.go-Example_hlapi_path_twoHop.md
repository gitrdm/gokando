```go
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

```


