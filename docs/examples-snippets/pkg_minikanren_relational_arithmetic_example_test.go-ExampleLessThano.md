```go
func ExampleLessThano() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessThano(NewAtom(3), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


