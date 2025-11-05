```go
func ExampleLogo_base10() {
	result := Run(1, func(q *Var) Goal {
		return Logo(NewAtom(10), NewAtom(1000), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


