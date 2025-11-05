```go
func ExampleNewBitSetDomain() {
	// Create a domain for Sudoku: values 1 through 9
	domain := minikanren.NewBitSetDomain(9)

	fmt.Printf("Domain size: %d\n", domain.Count())
	fmt.Printf("Contains 5: %v\n", domain.Has(5))
	fmt.Printf("Contains 0: %v\n", domain.Has(0))
	fmt.Printf("Contains 10: %v\n", domain.Has(10))

	// Output:
	// Domain size: 9
	// Contains 5: true
	// Contains 0: false
	// Contains 10: false
}

```


