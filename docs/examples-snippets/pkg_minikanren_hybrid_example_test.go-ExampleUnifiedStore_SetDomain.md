```go
func ExampleUnifiedStore_SetDomain() {
	store := NewUnifiedStore()

	// Set domain for variable 1: values {1, 2, 3}
	domain := NewBitSetDomainFromValues(10, []int{1, 2, 3})
	store, _ = store.SetDomain(1, domain)

	// Retrieve and inspect domain
	d := store.GetDomain(1)
	fmt.Printf("Domain size: %d\n", d.Count())
	fmt.Printf("Contains 2: %v\n", d.Has(2))

	// Output:
	// Domain size: 3
	// Contains 2: true
}

```


