```go
func ExampleIfte() {
	res := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		cond := Conde(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
		thenG := Project([]Term{x}, func(v []Term) Goal { return Eq(q, v[0]) })
		elseG := Eq(q, NewAtom("none"))
		return Ifte(cond, thenG, elseG)
	})

	// Should get exactly one solution (commits to first)
	fmt.Println(len(res))
	// Output: 1
}

```


