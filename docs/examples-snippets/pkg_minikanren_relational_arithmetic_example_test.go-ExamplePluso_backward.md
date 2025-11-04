```go
func ExamplePluso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(q, NewAtom(3), NewAtom(8))
	})
	fmt.Println(result[0])
	// Output: 5
}

```


