```go
func ExampleTimeso_notDivisible() {
	// ? * 3 = 10 has no integer solution
	result := Run(1, func(q *Var) Goal {
		return Timeso(q, NewAtom(3), NewAtom(10))
	})
	fmt.Println(len(result))
	// Output: 0
}

```


