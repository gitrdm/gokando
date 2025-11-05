```go
func ExampleDivo_backward() {
	result := Run(1, func(q *Var) Goal {
		return Divo(q, NewAtom(5), NewAtom(3))
	})
	fmt.Println(result[0])
	// Output: 15
}

```


