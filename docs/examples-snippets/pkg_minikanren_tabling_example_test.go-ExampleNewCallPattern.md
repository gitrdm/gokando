```go
func ExampleNewCallPattern() {
	// Create a call pattern for edge(a, b)
	args := []Term{NewAtom("a"), NewAtom("b")}
	pattern := NewCallPattern("edge", args)

	fmt.Printf("Predicate: %s\n", pattern.PredicateID())
	fmt.Printf("Structure: %s\n", pattern.ArgStructure())
	fmt.Printf("Full pattern: %s\n", pattern.String())

	// Output:
	// Predicate: edge
	// Structure: atom(a),atom(b)
	// Full pattern: edge(atom(a),atom(b))
}

```


