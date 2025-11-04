```go
func ExampleExpo_zeroExponent() {
	result := Run(1, func(q *Var) Goal {
		return Expo(NewAtom(5), NewAtom(0), q)
	})
	fmt.Println(result[0])
	// Output: 1
}

```


