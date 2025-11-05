```go
func ExamplePluso() {
	result := Run(1, func(q *Var) Goal {
		return Pluso(NewAtom(2), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

```


