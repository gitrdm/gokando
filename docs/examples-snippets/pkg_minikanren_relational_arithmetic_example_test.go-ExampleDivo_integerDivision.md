```go
func ExampleDivo_integerDivision() {
	result := Run(1, func(q *Var) Goal {
		return Divo(NewAtom(7), NewAtom(2), q)
	})
	fmt.Println(result[0])
	// Output: 3
}

```


