```go
func ExamplePluso_composition() {
	// Solve (x + 3) * 2 = 10 for x
	result := Run(1, func(q *Var) Goal {
		temp := Fresh("temp")
		return Conj(
			Timeso(temp, NewAtom(2), NewAtom(10)), // temp = 5
			Pluso(q, NewAtom(3), temp),            // q + 3 = 5
		)
	})
	fmt.Println(result[0])
	// Output: 2
}

```


