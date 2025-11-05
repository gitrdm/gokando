```go
func ExampleMinuso() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(10), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 7
}

```


