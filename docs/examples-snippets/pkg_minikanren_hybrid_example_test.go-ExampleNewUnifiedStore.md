```go
func ExampleNewUnifiedStore() {
	// Create a new unified store
	store := NewUnifiedStore()

	// Add a relational binding for logic variable 1
	store, _ = store.AddBinding(1, NewAtom(42))

	// Add an FD domain for FD variable 2
	store, _ = store.SetDomain(2, NewBitSetDomain(10))

	// The store can hold both types of information
	fmt.Printf("Store has bindings: %d\n", len(store.getAllBindings()))
	fmt.Printf("Store has domains: %d\n", len(store.getAllDomains()))

	// Output:
	// Store has bindings: 1
	// Store has domains: 1
}

```


