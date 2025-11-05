```go
func ExampleSubgoalEntry_dependencies() {
	// Create a dependency chain: path depends on edge
	edgePattern := NewCallPattern("edge", []Term{NewAtom("a"), NewAtom("b")})
	pathPattern := NewCallPattern("path", []Term{NewAtom("a"), NewAtom("b")})

	edgeEntry := NewSubgoalEntry(edgePattern)
	pathEntry := NewSubgoalEntry(pathPattern)

	// path depends on edge
	pathEntry.AddDependency(edgeEntry)

	deps := pathEntry.Dependencies()
	fmt.Printf("Number of dependencies: %d\n", len(deps))
	fmt.Printf("Depends on: %s\n", deps[0].Pattern().String())

	// Output:
	// Number of dependencies: 1
	// Depends on: edge(atom(a),atom(b))
}

```


