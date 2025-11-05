```go
func ExampleGreaterThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			GreaterThano(NewAtom(10), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


