```go
func ExampleRelationalPlugin() {
	// Create relational plugin
	plugin := NewRelationalPlugin()

	// Create a type constraint for variable 1
	typeConstraint := NewTypeConstraint(Fresh("x"), NumberType)

	fmt.Printf("Plugin name: %s\n", plugin.Name())
	fmt.Printf("Can handle type constraint: %v\n", plugin.CanHandle(typeConstraint))

	// Create store with binding for variable 1
	store := NewUnifiedStore()
	store, _ = store.AddBinding(1, NewAtom(42))
	store = store.AddConstraint(typeConstraint)

	// Propagate (checks constraints)
	result, err := plugin.Propagate(store)

	if err != nil {
		fmt.Println("Constraint violated")
	} else {
		fmt.Printf("Constraint satisfied: %v\n", result != nil)
	}

	// Output:
	// Plugin name: Relational
	// Can handle type constraint: true
	// Constraint satisfied: true
}

```


