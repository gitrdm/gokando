```go
func ExampleMinuso_negative() {
	result := Run(1, func(q *Var) Goal {
		return Minuso(NewAtom(3), NewAtom(7), q)
	})
	fmt.Println(result[0])
	// Output: -4
}

```


