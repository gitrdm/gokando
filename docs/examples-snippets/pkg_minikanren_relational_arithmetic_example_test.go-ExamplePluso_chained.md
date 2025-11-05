```go
func ExamplePluso_chained() {
	// x + y = 5, y + z = 7, with x = 2, solve for y and z
	result := Run(1, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")
		z := Fresh("z")
		return Conj(
			Eq(x, NewAtom(2)),
			Pluso(x, y, NewAtom(5)),
			Pluso(y, z, NewAtom(7)),
			Eq(q, List(x, y, z)),
		)
	})

	// Extract list values
	list := result[0]
	var vals []Term
	for {
		if pair, ok := list.(*Pair); ok {
			vals = append(vals, pair.Car())
			list = pair.Cdr()
		} else {
			break
		}
	}
	fmt.Printf("x=%v, y=%v, z=%v\n", vals[0], vals[1], vals[2])
	// Output: x=2, y=3, z=4
}

```


