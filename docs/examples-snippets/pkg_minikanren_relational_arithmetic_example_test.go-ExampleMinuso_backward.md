```go
func ExampleMinuso_backward() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), q, NewAtom(6))
	})
	fmt.Println(result[0])
	// Output: 4
}

```


