```go
func ExampleTimeso() {
	result := Run(1, func(q *Var) Goal {
		return Timeso(NewAtom(4), NewAtom(5), q)
	})
	fmt.Println(result[0])
	// Output: 20
}

```


