```go
func ExampleIfa() {
	res := Run(10, func(q *Var) Goal {
		x := Fresh("x")
		cond := Conde(Eq(x, NewAtom(1)), Eq(x, NewAtom(2)))
		thenG := Project([]Term{x}, func(v []Term) Goal { return Eq(q, v[0]) })
		elseG := Eq(q, NewAtom("none"))
		return Ifa(cond, thenG, elseG)
	})

	// Extract and sort values for deterministic output
	var values []int
	for _, r := range res {
		if atom, ok := r.(*Atom); ok {
			if val, ok := atom.Value().(int); ok {
				values = append(values, val)
			}
		}
	}
	sort.Ints(values)
	fmt.Println(values)
	// Output: [1 2]
}

```


