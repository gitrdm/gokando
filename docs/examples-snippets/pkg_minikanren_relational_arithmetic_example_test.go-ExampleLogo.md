```go
func ExampleLogo() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(2), NewAtom(1024), q)
	})
	fmt.Println(result[0])
	// Output: 10
}

```


