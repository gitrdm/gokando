```go
func ExampleLessThano_withArithmetic() {
	// Find x where x + 2 < 10 and x = 3
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Eq(q, NewAtom(3)),
			Pluso(q, NewAtom(2), temp),
			LessThano(temp, NewAtom(10)),
		)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


