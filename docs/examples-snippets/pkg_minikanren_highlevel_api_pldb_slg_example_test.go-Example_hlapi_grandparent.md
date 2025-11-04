```go
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

```


