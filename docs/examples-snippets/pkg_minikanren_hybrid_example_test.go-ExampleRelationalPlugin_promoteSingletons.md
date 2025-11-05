```go
func ExampleRelationalPlugin_promoteSingletons() {
	// Create relational plugin
	plugin := NewRelationalPlugin()

	// Create store with singleton FD domain
	store := NewUnifiedStore()
	singletonDomain := NewBitSetDomainFromValues(10, []int{7})
	store, _ = store.SetDomain(1, singletonDomain)

	// Propagate (should promote singleton)
	result, _ := plugin.Propagate(store)

	// Check if binding was created
	binding := result.GetBinding(1)
	if binding != nil {
		atom := binding.(*Atom)
		fmt.Printf("Singleton promoted to binding: %v\n", atom.Value())
	}

	// Output:
	// Singleton promoted to binding: 7
}

```


