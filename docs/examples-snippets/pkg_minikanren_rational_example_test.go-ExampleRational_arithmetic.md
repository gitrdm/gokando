```go
func ExampleRational_arithmetic() {
	a := NewRational(1, 2) // 1/2
	b := NewRational(1, 3) // 1/3

	sum := a.Add(b)
	diff := a.Sub(b)
	product := a.Mul(b)
	quotient := a.Div(b)

	fmt.Printf("1/2 + 1/3 = %s\n", sum)
	fmt.Printf("1/2 - 1/3 = %s\n", diff)
	fmt.Printf("1/2 * 1/3 = %s\n", product)
	fmt.Printf("1/2 / 1/3 = %s\n", quotient)

	// Output:
	// 1/2 + 1/3 = 5/6
	// 1/2 - 1/3 = 1/6
	// 1/2 * 1/3 = 1/6
	// 1/2 / 1/3 = 3/2
}

```


