```go
func ExampleNewClause() {
	// Pattern matching with variable binding and multiple goals
	result := Run(5, func(q *Var) Goal {
		x := Fresh("x")
		y := Fresh("y")

		return Matche(NewPair(NewAtom(10), NewAtom(20)),
			NewClause(
				NewPair(x, y),
				// Multiple goals executed in sequence
				Eq(x, NewAtom(10)),
				Eq(y, NewAtom(20)),
				Eq(q, NewAtom("success")),
			),
		)
	})

	fmt.Println(result[0])

	// Output:
	// success
}

```


