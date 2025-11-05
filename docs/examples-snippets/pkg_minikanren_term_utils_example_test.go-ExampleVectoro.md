```go
func ExampleVectoro() {
	slice1 := []int{1, 2, 3}
	slice2 := []string{"a", "b", "c"}

	result := Run(2, func(q *Var) Goal {
		return Conj(
			Vectoro(q),
			Membero(q, List(
				NewAtom(slice1),
				NewAtom("not-a-vector"),
				NewAtom(slice2),
			)),
		)
	})

	fmt.Printf("Vector values: %d results\n", len(result))
	// Output:
	// Vector values: 2 results
}

```


