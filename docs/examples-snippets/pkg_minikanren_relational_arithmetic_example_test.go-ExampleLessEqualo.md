```go
func ExampleLessEqualo() {
	result := Run(1, func(q *Var) Goal {
		return Conj(
			LessEqualo(NewAtom(5), NewAtom(5)),
			Eq(q, NewAtom("yes")),
		)
	})
	fmt.Println(result[0])
	// Output: yes
}

```


