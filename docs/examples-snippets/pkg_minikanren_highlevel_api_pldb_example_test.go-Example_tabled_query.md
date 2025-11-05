```go
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

```


