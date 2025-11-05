```go
func ExampleExpo_verification() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			Expo(NewAtom(3), NewAtom(4), NewAtom(81)),
			Eq(q, NewAtom("correct")),
		)
	})
	fmt.Println(result[0])
	// Output: correct
}

```


