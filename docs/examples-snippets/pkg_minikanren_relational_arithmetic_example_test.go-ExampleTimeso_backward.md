```go
func ExampleTimeso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(6), NewAtom(24))
	})
	fmt.Println(result[0])
	// Output: 4
}

```


