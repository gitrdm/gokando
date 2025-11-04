```go
func ExampleDbRel() {
	// Create a binary relation for parent-child relationships
	// Index both columns for fast lookups
	parent, err := DbRel("parent", 2, 0, 1)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Relation: %s (arity=%d)\n", parent.Name(), parent.Arity())
	fmt.Printf("Column 0 indexed: %v\n", parent.IsIndexed(0))
	fmt.Printf("Column 1 indexed: %v\n", parent.IsIndexed(1))

	// Output:
	// Relation: parent (arity=2)
	// Column 0 indexed: true
	// Column 1 indexed: true
}

```


