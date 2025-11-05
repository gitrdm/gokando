```go
func ExampleNewHybridRegistry() {
	// Create a registry for tracking relational↔FD variable mappings
	registry := NewHybridRegistry()

	// Setup variables
	model := NewModel()
	age := Fresh("age")
	ageVar := model.NewVariable(NewBitSetDomain(100))

	// Register the mapping
	registry, _ = registry.MapVars(age, ageVar)

	// Query the mapping
	fdID := registry.GetFDVariable(age)
	fmt.Printf("Has mapping: %t\n", fdID >= 0)
	fmt.Printf("Registry has %d mapping(s)\n", registry.MappingCount())

	// Output:
	// Has mapping: true
	// Registry has 1 mapping(s)
}

```


