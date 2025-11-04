```go
func ExampleNewBitSetDomainFromValues() {
	// Create a domain with only even digits
	evenDigits := minikanren.NewBitSetDomainFromValues(9, []int{2, 4, 6, 8})

	fmt.Printf("Domain: %s\n", evenDigits.String())
	fmt.Printf("Size: %d\n", evenDigits.Count())
	fmt.Printf("Has 3: %v\n", evenDigits.Has(3))
	fmt.Printf("Has 4: %v\n", evenDigits.Has(4))

	// Output:
	// Domain: {2,4,6,8}
	// Size: 4
	// Has 3: false
	// Has 4: true
}

```


