```go
func ExampleBitSetDomain_IsSingleton() {
	domain := minikanren.NewBitSetDomain(5)
	fmt.Printf("Initial domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())

	// Prune until singleton
	domain = domain.Remove(1).(*minikanren.BitSetDomain)
	domain = domain.Remove(2).(*minikanren.BitSetDomain)
	domain = domain.Remove(4).(*minikanren.BitSetDomain)
	domain = domain.Remove(5).(*minikanren.BitSetDomain)

	fmt.Printf("Domain %s is singleton: %v\n", domain.String(), domain.IsSingleton())
	if domain.IsSingleton() {
		fmt.Printf("Value: %d\n", domain.SingletonValue())
	}

	// Output:
	// Initial domain {1..5} is singleton: false
	// Domain {3} is singleton: true
	// Value: 3
}

```


