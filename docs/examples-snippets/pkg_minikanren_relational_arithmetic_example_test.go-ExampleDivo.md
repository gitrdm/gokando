```go
func ExampleDivo() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(15), NewAtom(3), q)
	})
	fmt.Println(result[0])
	// Output: 5
}

```


