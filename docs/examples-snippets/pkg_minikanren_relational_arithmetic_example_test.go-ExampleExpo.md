```go
func ExampleExpo() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(2), NewAtom(10), q)
	})
	fmt.Println(result[0])
	// Output: 1024
}

```


