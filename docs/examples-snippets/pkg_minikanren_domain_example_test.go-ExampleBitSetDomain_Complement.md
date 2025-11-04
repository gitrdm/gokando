```go
func ExampleBitSetDomain_Complement() {
	// Domain {2,4,6,8} within range 1-10
	evenDigits := minikanren.NewBitSetDomainFromValues(10, []int{2, 4, 6, 8})

	// Complement gives odd digits plus 10
	oddDigits := evenDigits.Complement()

	fmt.Printf("Even: %s\n", evenDigits.String())
	fmt.Printf("Odd: %s\n", oddDigits.String())

	// Output:
	// Even: {2,4,6,8}
	// Odd: {1,3,5,7,9,10}
}

```


