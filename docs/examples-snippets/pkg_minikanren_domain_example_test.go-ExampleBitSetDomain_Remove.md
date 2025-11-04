```go
func ExampleBitSetDomain_Remove() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial: %s\n", domain.String())

	// Remove value 3
	domain = domain.Remove(3).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 3: %s\n", domain.String())

	// Remove value 5
	domain = domain.Remove(5).(*minikanren.BitSetDomain)
	fmt.Printf("After removing 5: %s\n", domain.String())

	// Output:
	// Initial: {1..5}
	// After removing 3: {1,2,4,5}
	// After removing 5: {1,2,4}
}

```


